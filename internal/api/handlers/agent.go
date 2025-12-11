package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rexec/rexec/internal/pubsub"
	"github.com/rexec/rexec/internal/storage"
)

// AgentHandler handles external agent connections
type AgentHandler struct {
	store            *storage.PostgresStore
	agents           map[string]*AgentConnection
	agentsMu         sync.RWMutex
	upgrader         websocket.Upgrader
	jwtSecret        []byte
	eventsHub        *ContainerEventsHub      // For broadcasting agent connect/disconnect
	pubsubHub        *pubsub.Hub              // For horizontal scaling
	remoteSessions   map[string]*AgentSession // Sessions connected to remote agents
	remoteSessionsMu sync.RWMutex
}

type AgentConnection struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	OS          string          `json:"os"`
	Arch        string          `json:"arch"`
	Shell       string          `json:"shell"`
	Tags        []string        `json:"tags,omitempty"`
	UserID      string          `json:"user_id"`
	Status      string          `json:"status"`
	ConnectedAt time.Time       `json:"connected_at"`
	LastPing    time.Time       `json:"last_ping"`
	conn        *websocket.Conn
	sessions    map[string]*AgentSession
	sessionsMu  sync.RWMutex
	// System info from agent
	SystemInfo map[string]interface{} `json:"system_info,omitempty"`
	Stats      map[string]interface{} `json:"stats,omitempty"`
}

type AgentSession struct {
	ID        string
	AgentID   string // Added for remote session tracking
	UserConn  *websocket.Conn
	CreatedAt time.Time
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(store *storage.PostgresStore) *AgentHandler {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "rexec-dev-secret-change-in-production"
	}

	return &AgentHandler{
		store:          store,
		agents:         make(map[string]*AgentConnection),
		remoteSessions: make(map[string]*AgentSession),
		jwtSecret:      []byte(secret),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Prevent Cross-Site WebSocket Hijacking (CSWSH)
				origin := r.Header.Get("Origin")

				// Allow non-browser clients (empty origin) by default
				// Set BLOCK_EMPTY_ORIGIN=true to block them (requires agents/CLI to send Origin)
				if origin == "" {
					return os.Getenv("BLOCK_EMPTY_ORIGIN") != "true"
				}

				allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
				if allowedOriginsStr == "" {
					// Default to allowing all origins if not configured (e.g., in development)
					return true
				}

				allowedOrigins := strings.Split(allowedOriginsStr, ",")
				for _, ao := range allowedOrigins {
					if strings.TrimSpace(ao) == origin {
						return true
					}
				}
				log.Printf("WebSocket connection from disallowed origin: %s", origin)
				return false
			},
			// Optimized buffers for lower memory footprint per connection
			// 32KB to support large pastes/vibe coding
			ReadBufferSize:  32 * 1024,
			WriteBufferSize: 32 * 1024,
			// Enable compression to save bandwidth at the cost of slight CPU
			EnableCompression: true,
		},
	}
}

// SetEventsHub sets the container events hub for agent notifications
func (h *AgentHandler) SetEventsHub(hub *ContainerEventsHub) {
	h.eventsHub = hub
}

// SetPubSubHub sets the redis hub for horizontal scaling
func (h *AgentHandler) SetPubSubHub(hub *pubsub.Hub) {
	h.pubsubHub = hub
	// Subscribe to terminal proxy channel
	if h.pubsubHub != nil {
		h.pubsubHub.Subscribe(pubsub.ChannelTerminalProxy, h.handleTerminalProxyMessage)
	}
}

// handleTerminalProxyMessage handles cross-instance terminal messages
func (h *AgentHandler) handleTerminalProxyMessage(msg pubsub.Message) {
	var proxyMsg pubsub.TerminalProxyMessage
	if err := json.Unmarshal(msg.Payload, &proxyMsg); err != nil {
		log.Printf("Failed to unmarshal proxy message: %v", err)
		return
	}

	// 1. If this message is input/resize intended for a LOCAL AGENT
	if proxyMsg.Type == "input" || proxyMsg.Type == "resize" || proxyMsg.Type == "start_session" || proxyMsg.Type == "stop_session" {
		h.agentsMu.RLock()
		agentConn, ok := h.agents[proxyMsg.AgentID]
		h.agentsMu.RUnlock()

		if ok && agentConn.conn != nil {
			// Found local agent, forward message
			switch proxyMsg.Type {
			case "input":
				agentConn.conn.WriteJSON(map[string]interface{}{
					"type": "shell_input",
					"data": map[string]interface{}{
						"session_id": proxyMsg.SessionID,
						"data":       proxyMsg.Data, // Already bytes
					},
				})
			case "resize":
				agentConn.conn.WriteJSON(map[string]interface{}{
					"type": "shell_resize",
					"data": map[string]interface{}{
						"cols": proxyMsg.Cols,
						"rows": proxyMsg.Rows,
					},
				})
			case "start_session":
				agentConn.conn.WriteJSON(map[string]interface{}{
					"type": "shell_start",
					"data": map[string]string{
						"session_id": proxyMsg.SessionID,
					},
				})
			case "stop_session":
				// Handle stop?
			}
		}
		return
	}

	// 2. If this message is output intended for a REMOTE SESSION (user connected to this instance)
	if proxyMsg.Type == "output" || proxyMsg.Type == "status" {
		// If SessionID is "broadcast", send to all sessions for this AgentID
		if proxyMsg.SessionID == "broadcast" {
			h.remoteSessionsMu.RLock()
			for _, session := range h.remoteSessions {
				if session.AgentID == proxyMsg.AgentID && session.UserConn != nil {
					// Forward to this session
					if proxyMsg.Type == "output" {
						session.UserConn.WriteJSON(map[string]interface{}{
							"type": "output",
							"data": string(proxyMsg.Data),
						})
					}
				}
			}
			h.remoteSessionsMu.RUnlock()
			return
		}

		h.remoteSessionsMu.RLock()
		session, ok := h.remoteSessions[proxyMsg.SessionID]
		h.remoteSessionsMu.RUnlock()

		if ok && session.UserConn != nil {
			if proxyMsg.Type == "output" {
				session.UserConn.WriteJSON(map[string]interface{}{
					"type": "output",
					"data": string(proxyMsg.Data),
				})
			}
		}
	}
}

// RegisterAgent handles agent registration
func (h *AgentHandler) RegisterAgent(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		OS          string   `json:"os"`
		Arch        string   `json:"arch"`
		Shell       string   `json:"shell"`
		Tags        []string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create agent record
	agent := &AgentConnection{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		OS:          req.OS,
		Arch:        req.Arch,
		Shell:       req.Shell,
		Tags:        req.Tags,
		UserID:      userID.(string),
		Status:      "registered",
		sessions:    make(map[string]*AgentSession),
	}

	// Store in database
	ctx := context.Background()
	if err := h.store.CreateAgent(ctx, agent.ID, agent.UserID, agent.Name, agent.Description, agent.OS, agent.Arch, agent.Shell, agent.Tags); err != nil {
		log.Printf("Failed to create agent: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register agent"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":   agent.ID,
		"name": agent.Name,
	})
}

// ListAgents returns all agents for the user
func (h *AgentHandler) ListAgents(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := context.Background()
	agents, err := h.store.GetAgentsByUser(ctx, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch agents"})
		return
	}

	// Add online status
	h.agentsMu.RLock()
	for i := range agents {
		if conn, ok := h.agents[agents[i].ID]; ok {
			agents[i].Status = "online"
			agents[i].ConnectedAt = conn.ConnectedAt
			agents[i].LastPing = conn.LastPing
		} else {
			agents[i].Status = "offline"
		}
	}
	h.agentsMu.RUnlock()

	c.JSON(http.StatusOK, agents)
}

// DeleteAgent removes an agent
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	agentID := c.Param("id")

	ctx := context.Background()

	// Verify ownership
	agent, err := h.store.GetAgent(ctx, agentID)
	if err != nil || agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	if agent.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
		return
	}

	// Disconnect if online
	h.agentsMu.Lock()
	if conn, ok := h.agents[agentID]; ok {
		conn.conn.Close()
		delete(h.agents, agentID)
	}
	h.agentsMu.Unlock()

	// Delete from database
	if err := h.store.DeleteAgent(ctx, agentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete agent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "agent deleted"})
}

// HandleAgentWebSocket handles the WebSocket connection from the agent
func (h *AgentHandler) HandleAgentWebSocket(c *gin.Context) {
	agentID := c.Param("id")
	token := c.Query("token")

	// Verify token and get user
	userID, err := h.verifyToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Verify agent ownership
	ctx := context.Background()
	agent, err := h.store.GetAgent(ctx, agentID)
	if err != nil || agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	if agent.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
		return
	}

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade agent connection: %v", err)
		return
	}

	// Create agent connection
	agentConn := &AgentConnection{
		ID:          agentID,
		Name:        agent.Name,
		Description: agent.Description,
		OS:          c.GetHeader("X-Agent-OS"),
		Arch:        c.GetHeader("X-Agent-Arch"),
		Shell:       c.GetHeader("X-Agent-Shell"),
		UserID:      userID,
		Status:      "online",
		ConnectedAt: time.Now(),
		LastPing:    time.Now(),
		conn:        conn,
		sessions:    make(map[string]*AgentSession),
	}

	// Store connection
	h.agentsMu.Lock()
	h.agents[agentID] = agentConn
	h.agentsMu.Unlock()

	log.Printf("Agent connected: %s (%s)", agent.Name, agentID)

	// Register agent location in Redis
	if h.pubsubHub != nil {
		if err := h.pubsubHub.RegisterAgentLocation(agentID, userID); err != nil {
			log.Printf("Failed to register agent location: %v", err)
		}
	}

	// Broadcast agent connected event via WebSocket
	if h.eventsHub != nil {
		h.eventsHub.NotifyAgentConnected(agent.UserID, h.buildAgentData(agentConn))
	}

	// Handle messages
	defer func() {
		h.agentsMu.Lock()
		delete(h.agents, agentID)
		h.agentsMu.Unlock()
		conn.Close()
		log.Printf("Agent disconnected: %s (%s)", agent.Name, agentID)

		// Unregister agent location
		if h.pubsubHub != nil {
			h.pubsubHub.UnregisterAgentLocation(agentID, userID)
		}

		// Broadcast agent disconnected event via WebSocket
		if h.eventsHub != nil {
			h.eventsHub.NotifyAgentDisconnected(agent.UserID, agentID)
		}
	}()

	// Set up ping/pong
	conn.SetPongHandler(func(string) error {
		agentConn.LastPing = time.Now()
		if h.pubsubHub != nil {
			// Refresh TTL
			h.pubsubHub.RefreshAgentLocation(agentID)
		}
		return nil
	})

	// Start ping ticker
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// Read messages from agent
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}

		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "shell_output":
			// Forward to all connected user sessions in the format the frontend expects
			// Agent sends: {"type": "shell_output", "data": {"data": <base64-encoded-bytes>}}
			var outputData struct {
				Data []byte `json:"data"`
			}
			if err := json.Unmarshal(msg.Data, &outputData); err == nil {
				// outputData.Data is now the decoded bytes
				outputMsg := map[string]interface{}{
					"type": "output",
					"data": string(outputData.Data),
				}

				// 1. Write to local sessions
				agentConn.sessionsMu.RLock()
				for _, session := range agentConn.sessions {
					session.UserConn.WriteJSON(outputMsg)
				}
				agentConn.sessionsMu.RUnlock()

				// 2. Publish to Redis for remote sessions (if any)
				if h.pubsubHub != nil {
					// Use "broadcast" session ID to reach all sessions for this agent
					h.pubsubHub.ProxyTerminalData(agentID, "broadcast", "output", outputData.Data, 0, 0)
				}
			} else {
				log.Printf("Failed to unmarshal shell_output: %v", err)
			}

		case "shell_started", "shell_stopped", "shell_error":
			// Forward status to sessions
			agentConn.sessionsMu.RLock()
			for _, session := range agentConn.sessions {
				session.UserConn.WriteJSON(msg)
			}
			agentConn.sessionsMu.RUnlock()

		case "exec_result":
			// Forward to sessions
			agentConn.sessionsMu.RLock()
			for _, session := range agentConn.sessions {
				session.UserConn.WriteJSON(msg)
			}
			agentConn.sessionsMu.RUnlock()

		case "system_info":
			// Store system info from agent
			var sysInfo map[string]interface{}
			if err := json.Unmarshal(msg.Data, &sysInfo); err == nil {
				agentConn.SystemInfo = sysInfo
				log.Printf("Agent %s system info: %v", agentConn.Name, sysInfo)
			}

		case "stats":
			// Store and forward stats to connected user sessions
			var stats map[string]interface{}
			if err := json.Unmarshal(msg.Data, &stats); err == nil {
				agentConn.Stats = stats
				// Forward stats to all connected user sessions
				agentConn.sessionsMu.RLock()
				for _, session := range agentConn.sessions {
					session.UserConn.WriteJSON(map[string]interface{}{
						"type": "stats",
						"data": stats,
					})
				}
				agentConn.sessionsMu.RUnlock()
			}

		case "pong":
			agentConn.LastPing = time.Now()
		}
	}
}

// HandleUserWebSocket handles the WebSocket connection from a user to an agent
func (h *AgentHandler) HandleUserWebSocket(c *gin.Context) {
	agentID := c.Param("id")
	token := c.Query("token")

	// Verify token and get user
	userID, err := h.verifyToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Check if agent is online locally
	h.agentsMu.RLock()
	agentConn, isLocal := h.agents[agentID]
	h.agentsMu.RUnlock()

	var isRemote bool
	if !isLocal {
		// Check Redis for remote agent
		if h.pubsubHub != nil {
			if _, found := h.pubsubHub.GetAgentLocation(agentID); found {
				isRemote = true
			}
		}

		if !isRemote {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not online"})
			return
		}
	} else {
		// Local agent - verify ownership
		if agentConn.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
			return
		}
	}

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade user connection: %v", err)
		return
	}

	sessionID := uuid.New().String()
	session := &AgentSession{
		ID:        sessionID,
		AgentID:   agentID,
		UserConn:  conn,
		CreatedAt: time.Now(),
	}

	// === LOCAL AGENT HANDLING ===
	if isLocal {
		// Add session
		agentConn.sessionsMu.Lock()
		agentConn.sessions[sessionID] = session
		agentConn.sessionsMu.Unlock()

		// Send initial stats if available
		if agentConn.Stats != nil {
			conn.WriteJSON(map[string]interface{}{
				"type": "stats",
				"data": agentConn.Stats,
			})
		}

		// Check if this is a new session request (for split panes)
		newSession := c.Query("newSession") == "true"

		// Tell agent to start shell
		agentConn.conn.WriteJSON(map[string]interface{}{
			"type": "shell_start",
			"data": map[string]interface{}{
				"session_id":  sessionID,
				"new_session": newSession,
			},
		})

		defer func() {
			agentConn.sessionsMu.Lock()
			delete(agentConn.sessions, sessionID)
			agentConn.sessionsMu.Unlock()
			conn.Close()

			// Tell agent to stop shell if no more sessions
			agentConn.sessionsMu.RLock()
			if len(agentConn.sessions) == 0 {
				agentConn.conn.WriteJSON(map[string]interface{}{
					"type": "shell_stop",
				})
			}
			agentConn.sessionsMu.RUnlock()
		}()

		// Read messages from user and forward to agent
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			if messageType == websocket.BinaryMessage {
				// Forward input to agent
				agentConn.conn.WriteJSON(map[string]interface{}{
					"type": "shell_input",
					"data": map[string]interface{}{
						"session_id": sessionID,
						"data":       message,
					},
				})
			} else {
				// Parse JSON message
				var msg struct {
					Type string          `json:"type"`
					Data json.RawMessage `json:"data"`
				}

				if err := json.Unmarshal(message, &msg); err != nil {
					continue
				}

				switch msg.Type {
				case "input":
					var inputStr string
					if err := json.Unmarshal(msg.Data, &inputStr); err != nil {
						continue
					}
					agentConn.conn.WriteJSON(map[string]interface{}{
						"type": "shell_input",
						"data": map[string]interface{}{
							"session_id": sessionID,
							"data":       []byte(inputStr),
						},
					})

				case "resize":
					var resizeMsg struct {
						Cols int `json:"cols"`
						Rows int `json:"rows"`
					}
					if err := json.Unmarshal(message, &resizeMsg); err == nil && resizeMsg.Cols > 0 && resizeMsg.Rows > 0 {
						agentConn.conn.WriteJSON(map[string]interface{}{
							"type": "shell_resize",
							"data": map[string]int{
								"cols": resizeMsg.Cols,
								"rows": resizeMsg.Rows,
							},
						})
					}

				case "exec":
					agentConn.conn.WriteJSON(map[string]interface{}{
						"type": "exec",
						"data": msg.Data,
					})
				}
			}
		}
		return
	}

	// === REMOTE AGENT HANDLING ===
	if isRemote {
		// Register remote session
		h.remoteSessionsMu.Lock()
		h.remoteSessions[sessionID] = session
		h.remoteSessionsMu.Unlock()

		// Notify remote agent to start shell via Redis
		if h.pubsubHub != nil {
			h.pubsubHub.ProxyTerminalData(agentID, sessionID, "start_session", nil, 0, 0)
		}

		defer func() {
			h.remoteSessionsMu.Lock()
			delete(h.remoteSessions, sessionID)
			h.remoteSessionsMu.Unlock()
			conn.Close()

			if h.pubsubHub != nil {
				h.pubsubHub.ProxyTerminalData(agentID, sessionID, "stop_session", nil, 0, 0)
			}
		}()

		// Forward user messages to Redis
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			if h.pubsubHub == nil {
				break
			}

			if messageType == websocket.BinaryMessage {
				h.pubsubHub.ProxyTerminalData(agentID, sessionID, "input", message, 0, 0)
			} else {
				var msg struct {
					Type string          `json:"type"`
					Data json.RawMessage `json:"data"`
				}
				if err := json.Unmarshal(message, &msg); err != nil {
					continue
				}

				switch msg.Type {
				case "input":
					var inputStr string
					if err := json.Unmarshal(msg.Data, &inputStr); err == nil {
						h.pubsubHub.ProxyTerminalData(agentID, sessionID, "input", []byte(inputStr), 0, 0)
					}
				case "resize":
					var resizeMsg struct {
						Cols int `json:"cols"`
						Rows int `json:"rows"`
					}
					// Parse the top-level message again to get cols/rows
					if err := json.Unmarshal(message, &resizeMsg); err == nil {
						h.pubsubHub.ProxyTerminalData(agentID, sessionID, "resize", nil, resizeMsg.Cols, resizeMsg.Rows)
					}
				}
			}
		}
	}
}

// GetAgentStatus returns the status of an agent
func (h *AgentHandler) GetAgentStatus(c *gin.Context) {
	agentID := c.Param("id")

	h.agentsMu.RLock()
	agent, ok := h.agents[agentID]
	h.agentsMu.RUnlock()

	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"status": "offline",
		})
		return
	}

	response := gin.H{
		"status":       "online",
		"connected_at": agent.ConnectedAt,
		"last_ping":    agent.LastPing,
		"sessions":     len(agent.sessions),
		"os":           agent.OS,
		"arch":         agent.Arch,
	}

	// Include system info if available
	if agent.SystemInfo != nil {
		response["system_info"] = agent.SystemInfo
	}

	// Include latest stats if available
	if agent.Stats != nil {
		response["stats"] = agent.Stats
	}

	c.JSON(http.StatusOK, response)
}

// GetOnlineAgentsForUser returns all online agents for a specific user
// Used by ContainerHandler to include agents in the containers list
func (h *AgentHandler) GetOnlineAgentsForUser(userID string) []gin.H {
	h.agentsMu.RLock()
	defer h.agentsMu.RUnlock()

	agents := make([]gin.H, 0)
	// Create a slice of agents to sort
	type sortableAgent struct {
		Agent *AgentConnection
		Data  gin.H
	}
	var sortedAgents []sortableAgent

	for _, agent := range h.agents {
		if agent.UserID == userID && agent.Status == "online" {
			agentData := gin.H{
				"id":           "agent:" + agent.ID,
				"name":         agent.Name,
				"image":        agent.OS + "/" + agent.Arch,
				"status":       "running",
				"session_type": "agent",
				"created_at":   agent.ConnectedAt,
				"last_used_at": agent.LastPing,
				"os":           agent.OS,
				"arch":         agent.Arch,
				"shell":        agent.Shell,
			}

			// Add resources from system info/stats
			resources := gin.H{
				"memory_mb":  0,
				"cpu_shares": 1024,
				"disk_mb":    0,
			}

			if agent.SystemInfo != nil {
				if numCPU, ok := agent.SystemInfo["num_cpu"].(int); ok {
					resources["cpu_shares"] = numCPU * 1024
				}
				if mem, ok := agent.SystemInfo["memory"].(map[string]interface{}); ok {
					if total, ok := mem["total"].(float64); ok {
						resources["memory_mb"] = int(total / 1024 / 1024)
					}
				}
				if disk, ok := agent.SystemInfo["disk"].(map[string]interface{}); ok {
					if total, ok := disk["total"].(float64); ok {
						resources["disk_mb"] = int(total / 1024 / 1024)
					}
				}
				if hostname, ok := agent.SystemInfo["hostname"].(string); ok {
					agentData["hostname"] = hostname
				}
			}

			if agent.Stats != nil {
				if memLimit, ok := agent.Stats["memory_limit"].(float64); ok && memLimit > 0 {
					resources["memory_mb"] = int(memLimit / 1024 / 1024)
				}
				if diskLimit, ok := agent.Stats["disk_limit"].(float64); ok && diskLimit > 0 {
					resources["disk_mb"] = int(diskLimit / 1024 / 1024)
				}
				agentData["stats"] = agent.Stats
			}

			agentData["resources"] = resources
			sortedAgents = append(sortedAgents, sortableAgent{Agent: agent, Data: agentData})
		}
	}

	// Sort agents by ConnectedAt descending (newest first), then by Name
	sort.Slice(sortedAgents, func(i, j int) bool {
		if sortedAgents[i].Agent.ConnectedAt.Equal(sortedAgents[j].Agent.ConnectedAt) {
			return sortedAgents[i].Agent.Name < sortedAgents[j].Agent.Name
		}
		return sortedAgents[i].Agent.ConnectedAt.After(sortedAgents[j].Agent.ConnectedAt)
	})

	for _, sa := range sortedAgents {
		agents = append(agents, sa.Data)
	}

	return agents
}

// buildAgentData builds agent data in the same format as container data
func (h *AgentHandler) buildAgentData(agent *AgentConnection) gin.H {
	agentData := gin.H{
		"id":           "agent:" + agent.ID,
		"name":         agent.Name,
		"image":        agent.OS + "/" + agent.Arch,
		"status":       "running",
		"session_type": "agent",
		"created_at":   agent.ConnectedAt,
		"last_used_at": agent.LastPing,
		"os":           agent.OS,
		"arch":         agent.Arch,
		"shell":        agent.Shell,
	}

	resources := gin.H{
		"memory_mb":  0,
		"cpu_shares": 1024,
		"disk_mb":    0,
	}

	if agent.SystemInfo != nil {
		if numCPU, ok := agent.SystemInfo["num_cpu"].(int); ok {
			resources["cpu_shares"] = numCPU * 1024
		}
		if mem, ok := agent.SystemInfo["memory"].(map[string]interface{}); ok {
			if total, ok := mem["total"].(float64); ok {
				resources["memory_mb"] = int(total / 1024 / 1024)
			}
		}
		if disk, ok := agent.SystemInfo["disk"].(map[string]interface{}); ok {
			if total, ok := disk["total"].(float64); ok {
				resources["disk_mb"] = int(total / 1024 / 1024)
			}
		}
		if hostname, ok := agent.SystemInfo["hostname"].(string); ok {
			agentData["hostname"] = hostname
		}
	}

	if agent.Stats != nil {
		if memLimit, ok := agent.Stats["memory_limit"].(float64); ok && memLimit > 0 {
			resources["memory_mb"] = int(memLimit / 1024 / 1024)
		}
		if diskLimit, ok := agent.Stats["disk_limit"].(float64); ok && diskLimit > 0 {
			resources["disk_mb"] = int(diskLimit / 1024 / 1024)
		}
		agentData["stats"] = agent.Stats
	}

	agentData["resources"] = resources
	return agentData
}

// verifyToken parses and validates a JWT token, returning the user ID
func (h *AgentHandler) verifyToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", jwt.ErrTokenInvalidClaims
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", jwt.ErrTokenInvalidClaims
	}

	return userID, nil
}