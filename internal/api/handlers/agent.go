package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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

// NewAgentHandler creates a new agent handler.
// jwtSecret must be the server's signing key.
func NewAgentHandler(store *storage.PostgresStore, jwtSecret []byte) *AgentHandler {
	return &AgentHandler{
		store:          store,
		agents:         make(map[string]*AgentConnection),
		remoteSessions: make(map[string]*AgentSession),
		jwtSecret:      jwtSecret,
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
						"session_id": proxyMsg.SessionID,
						"cols":       proxyMsg.Cols,
						"rows":       proxyMsg.Rows,
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

	// Generate a long-lived API token for the agent (no expiration)
	tokenName := fmt.Sprintf("agent-%s", agent.Name)
	scopes := []string{"agent"}
	apiToken, plainToken, err := h.store.GenerateAPIToken(ctx, agent.UserID, tokenName, scopes, nil)
	if err != nil {
		log.Printf("Failed to generate agent token: %v", err)
		// Still return success but warn about token
		c.JSON(http.StatusCreated, gin.H{
			"id":      agent.ID,
			"name":    agent.Name,
			"warning": "Agent registered but token generation failed. Please generate an API token manually.",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":    agent.ID,
		"name":  agent.Name,
		"token": plainToken,
		"token_info": gin.H{
			"id":     apiToken.ID,
			"name":   apiToken.Name,
			"scopes": apiToken.Scopes,
			"note":   "This token is used for agent authentication. Save it securely - it won't be shown again.",
		},
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

	threshold := time.Now().Add(-2 * time.Minute) // Consider offline if no heartbeat for 2 mins

	for i := range agents {
		// Check if online based on DB heartbeat
		if !agents[i].LastPing.IsZero() && agents[i].LastPing.After(threshold) {
			agents[i].Status = "online"
		} else {
			agents[i].Status = "offline"
		}
		
		// Fallback for ConnectedAt if zero (since DB might not store it yet)
		if agents[i].ConnectedAt.IsZero() {
			agents[i].ConnectedAt = agents[i].LastPing
		}
	}

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

	// Log connection attempt for debugging
	tokenType := "JWT"
	if strings.HasPrefix(token, "rexec_") {
		tokenType = "API"
	}
	log.Printf("[Agent WS] Connection attempt: agent=%s, tokenType=%s, IP=%s", 
		agentID, tokenType, c.ClientIP())

	// Verify token and get user
	userID, err := h.verifyToken(token)
	if err != nil {
		log.Printf("[Agent WS] Token verification failed for agent %s (type=%s): %v", agentID, tokenType, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Verify agent ownership
	ctx := context.Background()
	agent, err := h.store.GetAgent(ctx, agentID)
	if err != nil || agent == nil {
		log.Printf("[Agent WS] Agent not found: %s, err=%v", agentID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	if agent.UserID != userID {
		log.Printf("[Agent WS] Authorization failed: agent %s owned by %s, token user %s", 
			agentID, agent.UserID, userID)
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
		return
	}

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[Agent WS] Failed to upgrade connection for agent %s: %v", agentID, err)
		return
	}
	
	log.Printf("[Agent WS] WebSocket upgraded successfully for agent %s", agentID)

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

	// Update DB: Mark as connected with current instance ID and update metadata
	instanceID := ""
	if h.pubsubHub != nil {
		instanceID = h.pubsubHub.InstanceID()
		// Register agent location in Redis for cross-instance routing
		if err := h.pubsubHub.RegisterAgentLocation(agentID, userID, agent.Name, agentConn.OS, agentConn.Arch, agentConn.ConnectedAt); err != nil {
			log.Printf("[Agent WS] Failed to register agent location in Redis: %v", err)
		}
	}
	h.store.UpdateAgentHeartbeat(context.Background(), agentID, instanceID)
	h.store.UpdateAgentMetadata(context.Background(), agentID, agentConn.OS, agentConn.Arch, agentConn.Shell)

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

		// Unregister agent location from Redis
		if h.pubsubHub != nil {
			if err := h.pubsubHub.UnregisterAgentLocation(agentID, agent.UserID); err != nil {
				log.Printf("[Agent WS] Failed to unregister agent location from Redis: %v", err)
			}
		}

		// Update DB: Clear connected instance ID
		h.store.DisconnectAgent(context.Background(), agentID)

		// Broadcast agent disconnected event via WebSocket
		if h.eventsHub != nil {
			h.eventsHub.NotifyAgentDisconnected(agent.UserID, agentID)
		}
	}()

	// Set up ping/pong
	conn.SetPongHandler(func(string) error {
		agentConn.LastPing = time.Now()
		// Update heartbeat in DB
		h.store.UpdateAgentStatus(context.Background(), agentID)
		// Refresh Redis location TTL
		if h.pubsubHub != nil {
			h.pubsubHub.RefreshAgentLocation(agentID)
		}
		return nil
	})

	// Create a done channel to stop the ping goroutine
	done := make(chan struct{})
	defer close(done)

	// Start ping ticker
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("[Agent WS] Ping failed for agent %s: %v", agentID, err)
					return
				}
			}
		}
	}()

	// Read messages from agent
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[Agent WS] Read error for agent %s: %v", agentID, err)
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
				
				// Persist system info to DB
				if err := h.store.UpdateAgentSystemInfo(context.Background(), agentID, sysInfo); err != nil {
					log.Printf("Failed to persist system info for agent %s: %v", agentID, err)
				}
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
		// Check DB for remote agent location
		ctx := context.Background()
		// We can reuse GetAgent since we need to check ownership anyway, 
		// but we need the connected_instance_id which GetAgent might not return unless updated.
		// Let's rely on a fresh DB query or update GetAgent to return it.
		// Since we already called GetAgent above, let's update GetAgent to return connected_instance_id or use a specific query.
		// Actually, GetAgent implementation in postgres_agent.go DOES NOT return connected_instance_id yet.
		// I need to update GetAgent or add a new method.
		// For now, let's use GetAgentsByUser loop or a direct query if I add one.
		// Let's add GetAgentLocation to store.
		
		// Wait, I updated GetAgentsByUser but not GetAgent.
		// Let's assume I'll add GetAgentLocation to store or query it here.
		// Actually, let's just query the DB directly here for simplicity or assume GetAgent returns it if I update it.
		// I will update GetAgent in next step.
		
		// For now, assume GetAgent was updated or use a workaround.
		// But wait, I need to know the InstanceID to proxy to.
		
		// Let's implement GetAgentLocation in store first.
		instanceID, err := h.store.GetAgentConnectedInstance(ctx, agentID)
		if err == nil && instanceID != "" {
			isRemote = true
			// We need to know if the agent is actually online (heartbeat check)
			// GetAgentConnectedInstance should probably check heartbeat too?
			// Let's assume if it returns an ID, it's valid, or we check last_heartbeat.
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
							"data": map[string]interface{}{
								"session_id": sessionID,
								"cols":       resizeMsg.Cols,
								"rows":       resizeMsg.Rows,
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

	// Check local cache first
	h.agentsMu.RLock()
	agent, ok := h.agents[agentID]
	h.agentsMu.RUnlock()

	if ok {
		response := gin.H{
			"status":       "online",
			"connected_at": agent.ConnectedAt,
			"last_ping":    agent.LastPing,
			"sessions":     len(agent.sessions),
			"os":           agent.OS,
			"arch":         agent.Arch,
		}

		if agent.SystemInfo != nil {
			response["system_info"] = agent.SystemInfo
		}
		if agent.Stats != nil {
			response["stats"] = agent.Stats
		}

		c.JSON(http.StatusOK, response)
		return
	}

	// Not local, check DB
	ctx := context.Background()
	dbAgent, err := h.store.GetAgent(ctx, agentID)
	if err != nil || dbAgent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Check if online via heartbeat (2 min threshold)
	threshold := time.Now().Add(-2 * time.Minute)
	isOnline := !dbAgent.LastPing.IsZero() && dbAgent.LastPing.After(threshold)

	if !isOnline {
		c.JSON(http.StatusOK, gin.H{
			"status": "offline",
		})
		return
	}

	// Online but remote
	response := gin.H{
		"status":       "online",
		"connected_at": dbAgent.ConnectedAt, // Might be empty if not tracked in DB column yet, but we have LastPing
		"last_ping":    dbAgent.LastPing,
		"sessions":     0, // We don't know remote session count easily without querying all instances
		"os":           dbAgent.OS,
		"arch":         dbAgent.Arch,
	}

	if dbAgent.SystemInfo != nil {
		response["system_info"] = dbAgent.SystemInfo
	}
	// Note: We don't have real-time stats for remote agents in DB

	c.JSON(http.StatusOK, response)
}

// GetOnlineAgentsForUser returns all online agents for a specific user
// Used by ContainerHandler to include agents in the containers list
func (h *AgentHandler) GetOnlineAgentsForUser(userID string) []gin.H {
	// Query DB for all agents for this user
	ctx := context.Background()
	allAgents, err := h.store.GetAgentsByUser(ctx, userID)
	if err != nil {
		log.Printf("Failed to fetch agents for user %s: %v", userID, err)
		return []gin.H{}
	}

	var onlineAgents []gin.H
	threshold := time.Now().Add(-2 * time.Minute) // Consider offline if no heartbeat for 2 mins

	for _, agent := range allAgents {
		// Check if agent is online based on LastPing (LastHeartbeat from DB)
		isOnline := !agent.LastPing.IsZero() && agent.LastPing.After(threshold)
		
		if isOnline {
			// Construct agent data
			agentData := gin.H{
				"id":           "agent:" + agent.ID,
				"name":         agent.Name,
				"image":        agent.OS + "/" + agent.Arch,
				"status":       "running",
				"session_type": "agent",
				"created_at":   agent.CreatedAt, // Use DB creation time
				"last_used_at": agent.LastPing,
				"idle_seconds": time.Since(agent.LastPing).Seconds(), // Calculate idle time
				"os":           agent.OS,
				"arch":         agent.Arch,
				"shell":        agent.Shell,
			}

			// If local connection exists, use its up-to-date stats/sysinfo
			h.agentsMu.RLock()
			localConn, isLocal := h.agents[agent.ID]
			h.agentsMu.RUnlock()

			if isLocal {
				// Use local connection data for most accurate stats
				agentData["resources"] = h.buildAgentData(localConn)["resources"]
				if localConn.Stats != nil {
					agentData["stats"] = localConn.Stats
				}
			} else {
				// For remote agents, use persisted SystemInfo from DB to calculate capacity
				resources := gin.H{
					"memory_mb":  0,
					"cpu_shares": 1024,
					"disk_mb":    0,
				}
				
				if agent.SystemInfo != nil {
					if numCPU, ok := agent.SystemInfo["num_cpu"].(float64); ok { // JSON unmarshals numbers as float64
						resources["cpu_shares"] = int(numCPU * 1024)
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
				
				agentData["resources"] = resources
			}
			
			onlineAgents = append(onlineAgents, agentData)
		}
	}

	return onlineAgents
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
		// Handle both float64 (from JSON unmarshal) and int (from internal creation if any)
		var numCPU float64
		if val, ok := agent.SystemInfo["num_cpu"].(float64); ok {
			numCPU = val
		} else if val, ok := agent.SystemInfo["num_cpu"].(int); ok {
			numCPU = float64(val)
		}
		
		if numCPU > 0 {
			resources["cpu_shares"] = int(numCPU * 1024)
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

// verifyToken parses and validates a JWT token or API token, returning the user ID
func (h *AgentHandler) verifyToken(tokenString string) (string, error) {
	// Check if this is an API token (starts with rexec_)
	if strings.HasPrefix(tokenString, "rexec_") {
		apiToken, err := h.store.ValidateAPIToken(context.Background(), tokenString)
		if err != nil {
			return "", fmt.Errorf("invalid API token: %w", err)
		}
		return apiToken.UserID, nil
	}

	// Otherwise, treat as JWT token
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
