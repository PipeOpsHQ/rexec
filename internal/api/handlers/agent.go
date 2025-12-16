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
	"github.com/rexec/rexec/internal/models"
	"github.com/rexec/rexec/internal/pubsub"
	"github.com/rexec/rexec/internal/storage"
)

const wsTokenProtocolPrefix = "rexec.token."

func tokenFromAuthorizationHeader(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.Fields(authHeader)
	if len(parts) >= 2 && strings.EqualFold(parts[0], "bearer") {
		return parts[1]
	}
	return ""
}

func tokenFromWebSocketSubprotocolHeader(headerVal string) string {
	if headerVal == "" {
		return ""
	}
	for _, part := range strings.Split(headerVal, ",") {
		proto := strings.TrimSpace(part)
		if strings.HasPrefix(proto, wsTokenProtocolPrefix) {
			token := strings.TrimPrefix(proto, wsTokenProtocolPrefix)
			if token != "" {
				return token
			}
		}
	}
	return ""
}

func tokenFromWebSocketRequest(c *gin.Context) string {
	if t := tokenFromAuthorizationHeader(c.GetHeader("Authorization")); t != "" {
		return t
	}
	if t := tokenFromWebSocketSubprotocolHeader(c.GetHeader("Sec-WebSocket-Protocol")); t != "" {
		return t
	}
	return c.Query("token")
}

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
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	OS          string    `json:"os"`
	Arch        string    `json:"arch"`
	Shell       string    `json:"shell"`
	Distro      string    `json:"distro,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	UserID      string    `json:"user_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	ConnectedAt time.Time `json:"connected_at"`
	LastPing    time.Time `json:"last_ping"`
	conn        *websocket.Conn
	sessions    map[string]*AgentSession
	sessionsMu  sync.RWMutex
	// remoteSessionRefs tracks which server instances currently have at least one
	// user WebSocket subscribed to a given agent shell session (e.g. "main", "split-...").
	remoteSessionRefs map[string]map[string]struct{} // agentSessionID -> instanceID set
	// System info from agent
	SystemInfo map[string]interface{} `json:"system_info,omitempty"`
	Stats      map[string]interface{} `json:"stats,omitempty"`
}

type AgentSession struct {
	// ID is the client-provided connection ID (WebSocket `id` query param).
	// It is stable across reconnects and unique per UI tab/pane.
	ID string

	UserID string

	AgentID string
	// AgentSessionID is the shell session identifier on the agent process.
	// "main" is the shared session; split panes use "split-<id>".
	AgentSessionID string

	NewSession bool
	UserConn   *websocket.Conn
	CreatedAt  time.Time
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

				// Default allowed origins for Rexec
				defaultAllowedOrigins := []string{
					"https://rexec.pipeops.app",
					"https://rexec.pipeops.io",
					"https://rexec.pipeops.sh",
					"https://rexec.io",
					"https://rexec.sh",
					"http://localhost:8080",
					"http://localhost:5173",
					"http://127.0.0.1:8080",
					"http://127.0.0.1:5173",
				}

				// Check against default origins first
				for _, ao := range defaultAllowedOrigins {
					if ao == origin {
						return true
					}
				}

				// Then check additional origins from environment variable
				allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
				if allowedOriginsStr != "" {
					allowedOrigins := strings.Split(allowedOriginsStr, ",")
					for _, ao := range allowedOrigins {
						if strings.TrimSpace(ao) == origin {
							return true
						}
					}
				}

				// If ALLOWED_ORIGINS is not set and origin not in defaults, allow in development
				if allowedOriginsStr == "" {
					return true
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
				// Track that this server instance has an active subscriber for this session.
				sourceInstanceID := msg.InstanceID
				if sourceInstanceID != "" {
					agentConn.sessionsMu.Lock()
					if agentConn.remoteSessionRefs == nil {
						agentConn.remoteSessionRefs = make(map[string]map[string]struct{})
					}
					if _, exists := agentConn.remoteSessionRefs[proxyMsg.SessionID]; !exists {
						agentConn.remoteSessionRefs[proxyMsg.SessionID] = make(map[string]struct{})
					}
					agentConn.remoteSessionRefs[proxyMsg.SessionID][sourceInstanceID] = struct{}{}
					agentConn.sessionsMu.Unlock()
				}

				agentConn.conn.WriteJSON(map[string]interface{}{
					"type": "shell_start",
					"data": map[string]interface{}{
						"session_id":  proxyMsg.SessionID,
						"new_session": proxyMsg.NewSession,
					},
				})
			case "stop_session":
				sourceInstanceID := msg.InstanceID
				if sourceInstanceID != "" {
					agentConn.sessionsMu.Lock()
					if refs, exists := agentConn.remoteSessionRefs[proxyMsg.SessionID]; exists {
						delete(refs, sourceInstanceID)
						if len(refs) == 0 {
							delete(agentConn.remoteSessionRefs, proxyMsg.SessionID)
						} else {
							agentConn.remoteSessionRefs[proxyMsg.SessionID] = refs
						}
					}

					// Determine whether there are still any subscribers (local or remote) for this session.
					hasRemote := false
					if refs, ok := agentConn.remoteSessionRefs[proxyMsg.SessionID]; ok && len(refs) > 0 {
						hasRemote = true
					}
					hasLocal := false
					for _, s := range agentConn.sessions {
						if s != nil && s.AgentSessionID == proxyMsg.SessionID {
							hasLocal = true
							break
						}
					}
					noLocalSessions := len(agentConn.sessions) == 0
					noRemoteSessions := len(agentConn.remoteSessionRefs) == 0
					agentConn.sessionsMu.Unlock()

					// If nobody is subscribed anymore, stop the specific shell session.
					if !hasRemote && !hasLocal {
						agentConn.conn.WriteJSON(map[string]interface{}{
							"type": "shell_stop_session",
							"data": map[string]interface{}{
								"session_id": proxyMsg.SessionID,
							},
						})
					}

					// If absolutely no sessions remain, stop everything (cleans up any stragglers).
					if noLocalSessions && noRemoteSessions {
						agentConn.conn.WriteJSON(map[string]interface{}{
							"type": "shell_stop",
						})
					}
				}
			}
		}
		return
	}

	// 2. If this message is output intended for a REMOTE SESSION (user connected to this instance)
	if proxyMsg.Type == "output" || proxyMsg.Type == "status" {
		h.remoteSessionsMu.RLock()
		for _, session := range h.remoteSessions {
			if session == nil || session.UserConn == nil || session.AgentID != proxyMsg.AgentID {
				continue
			}

			// Backwards compatibility: "broadcast" sends to all sessions for this agent.
			if proxyMsg.SessionID == "broadcast" || proxyMsg.SessionID == "" {
				if proxyMsg.Type == "output" {
					session.UserConn.WriteJSON(map[string]interface{}{
						"type": "output",
						"data": string(proxyMsg.Data),
					})
				}
				continue
			}

			// Otherwise route by agent shell session ID ("main", "split-...").
			if session.AgentSessionID == proxyMsg.SessionID {
				if proxyMsg.Type == "output" {
					session.UserConn.WriteJSON(map[string]interface{}{
						"type": "output",
						"data": string(proxyMsg.Data),
					})
				}
			}
		}
		h.remoteSessionsMu.RUnlock()
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

	ctx := c.Request.Context()
	// Enforce plan-based agent limits (admins are exempt).
	if user, err := h.store.GetUserByID(ctx, userID.(string)); err == nil && user != nil && !user.IsAdmin {
		limit := maxRegisteredAgentsForTier(user.Tier, user.SubscriptionActive)
		if limit <= 0 {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "agent registration not available for your plan",
				"code":  "agent_limit_reached",
			})
			return
		}

		existing, err := h.store.GetAgentsByUser(ctx, user.ID)
		if err == nil && len(existing) >= limit {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "agent limit reached for your plan",
				"code":  "agent_limit_reached",
				"limit": limit,
			})
			return
		}
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

	ctx := c.Request.Context()
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

// GetAgent returns a specific agent record for the authenticated user.
// GET /api/agents/:id
func (h *AgentHandler) GetAgent(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	agentID := c.Param("id")
	ctx := c.Request.Context()
	agent, err := h.store.GetAgent(ctx, agentID)
	if err != nil || agent == nil || agent.UserID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Derive online/offline based on heartbeat
	threshold := time.Now().Add(-2 * time.Minute)
	if !agent.LastPing.IsZero() && agent.LastPing.After(threshold) {
		agent.Status = "online"
	} else {
		agent.Status = "offline"
	}

	c.JSON(http.StatusOK, agent)
}

// UpdateAgent updates an agent's name and description
// PATCH /api/agents/:id
func (h *AgentHandler) UpdateAgent(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	agentID := c.Param("id")
	ctx := c.Request.Context()

	// Verify ownership
	agent, err := h.store.GetAgent(ctx, agentID)
	if err != nil || agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	if agent.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
		return
	}

	// Parse request body
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Validate name
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = agent.Name // Keep existing name if not provided
	}

	description := strings.TrimSpace(req.Description)

	// Update agent in database
	if err := h.store.UpdateAgent(ctx, agentID, name, description, agent.Tags); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update agent"})
		return
	}

	// Update in-memory agent connection if online
	h.agentsMu.Lock()
	if conn, ok := h.agents[agentID]; ok {
		conn.Name = name
		conn.Description = description
	}
	h.agentsMu.Unlock()

	// Notify via events hub
	if h.eventsHub != nil {
		h.eventsHub.NotifyContainerUpdated(userID, gin.H{
			"id":          "agent:" + agentID,
			"name":        name,
			"description": description,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          agentID,
		"name":        name,
		"description": description,
		"message":     "Agent updated successfully",
	})
}

// DeleteAgent removes an agent
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	agentID := c.Param("id")

	ctx := c.Request.Context()

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
	token := tokenFromWebSocketRequest(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

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
	ctx := c.Request.Context()
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
		ID:                agentID,
		Name:              agent.Name,
		Description:       agent.Description,
		OS:                c.GetHeader("X-Agent-OS"),
		Arch:              c.GetHeader("X-Agent-Arch"),
		Shell:             c.GetHeader("X-Agent-Shell"),
		Distro:            c.GetHeader("X-Agent-Distro"),
		UserID:            userID,
		Status:            "online",
		CreatedAt:         agent.CreatedAt,
		ConnectedAt:       time.Now(),
		LastPing:          time.Now(),
		conn:              conn,
		sessions:          make(map[string]*AgentSession),
		remoteSessionRefs: make(map[string]map[string]struct{}),
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
	h.store.UpdateAgentMetadata(context.Background(), agentID, agentConn.OS, agentConn.Arch, agentConn.Shell, agentConn.Distro)

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
			var outputData struct {
				SessionID string `json:"session_id"`
				Data      []byte `json:"data"`
			}
			if err := json.Unmarshal(msg.Data, &outputData); err == nil {
				outputMsg := map[string]interface{}{
					"type": "output",
					"data": string(outputData.Data),
				}

				// 1. Route output to local sessions subscribed to this agent shell session.
				agentConn.sessionsMu.RLock()
				matched := false
				if outputData.SessionID == "" || outputData.SessionID == "broadcast" {
					for _, session := range agentConn.sessions {
						if session != nil && session.UserConn != nil {
							session.UserConn.WriteJSON(outputMsg)
						}
					}
					matched = true
				} else {
					for _, session := range agentConn.sessions {
						if session != nil && session.UserConn != nil && session.AgentSessionID == outputData.SessionID {
							session.UserConn.WriteJSON(outputMsg)
							matched = true
						}
					}
				}
				// Backwards compatibility: if an older agent sends a non-split session ID
				// and we don't have an exact match, treat it as "main".
				if !matched && outputData.SessionID != "" && !strings.HasPrefix(outputData.SessionID, "split-") {
					for _, session := range agentConn.sessions {
						if session != nil && session.UserConn != nil && session.AgentSessionID == "main" {
							session.UserConn.WriteJSON(outputMsg)
						}
					}
				}
				agentConn.sessionsMu.RUnlock()

				// 2. Publish to Redis for remote sessions (if any), routed by agent session ID.
				if h.pubsubHub != nil {
					targetSessionID := outputData.SessionID
					if targetSessionID == "" {
						targetSessionID = "broadcast"
					}
					h.pubsubHub.ProxyTerminalData(agentID, targetSessionID, "output", outputData.Data, 0, 0, false)
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

				// Also broadcast to events hub so dashboard updates without active terminal
				if h.eventsHub != nil {
					h.eventsHub.NotifyAgentStatsUpdated(agentConn.UserID, agentID, stats)
				}
			}

		case "pong":
			agentConn.LastPing = time.Now()
		}
	}
}

// HandleUserWebSocket handles the WebSocket connection from a user to an agent
func (h *AgentHandler) HandleUserWebSocket(c *gin.Context) {
	agentID := c.Param("id")
	token := tokenFromWebSocketRequest(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Verify token and get user
	userID, err := h.verifyToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Verify agent exists and ownership (required for both local and remote agents).
	ctx := c.Request.Context()
	agentRecord, err := h.store.GetAgent(ctx, agentID)
	if err != nil || agentRecord == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}
	if agentRecord.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
		return
	}

	// Enforce concurrent agent terminal limits (admins are exempt).
	if user, err := h.store.GetUserByID(ctx, userID); err == nil && user != nil && !user.IsAdmin {
		limit := maxConcurrentAgentTerminalsForTier(user.Tier, user.SubscriptionActive)
		if limit <= 0 {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "agent terminals not available for your plan",
				"code":  "agent_terminal_limit_reached",
			})
			return
		}

		// Only enforce when connecting to an additional agent (multiple tabs to the same agent are allowed).
		if !h.userHasAnyAgentSession(userID, agentID) {
			current := h.countDistinctAgentTerminalsForUser(userID)
			if current >= limit {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "agent terminal limit reached for your plan",
					"code":  "agent_terminal_limit_reached",
					"limit": limit,
				})
				return
			}
		}
	}

	// Connection/session identifiers:
	// - `id` is a client-provided stable ID (tab/pane). We use it as the server session key.
	// - `newSession=true` means "split pane": create an independent shell session on the agent.
	connectionID := c.Query("id")
	if connectionID == "" {
		connectionID = uuid.New().String()
	}
	newSession := c.Query("newSession") == "true"
	agentSessionID := "main"
	if newSession {
		agentSessionID = "split-" + connectionID
	}

	// Check if agent is online locally
	h.agentsMu.RLock()
	agentConn, isLocal := h.agents[agentID]
	h.agentsMu.RUnlock()

	isRemote := false
	if !isLocal {
		instanceID, err := h.store.GetAgentConnectedInstance(ctx, agentID)
		if err == nil && instanceID != "" {
			isRemote = true
		}
		if !isRemote {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not online"})
			return
		}
	}

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade user connection: %v", err)
		return
	}

	session := &AgentSession{
		ID:             connectionID,
		UserID:         userID,
		AgentID:        agentID,
		AgentSessionID: agentSessionID,
		NewSession:     newSession,
		UserConn:       conn,
		CreatedAt:      time.Now(),
	}

	// === LOCAL AGENT HANDLING ===
	if isLocal {
		// Add session (replace on reconnect)
		var previous *AgentSession
		agentConn.sessionsMu.Lock()
		previous = agentConn.sessions[connectionID]
		agentConn.sessions[connectionID] = session
		agentConn.sessionsMu.Unlock()
		if previous != nil && previous.UserConn != nil {
			previous.UserConn.Close()
		}

		// Send initial stats if available
		if agentConn.Stats != nil {
			conn.WriteJSON(map[string]interface{}{
				"type": "stats",
				"data": agentConn.Stats,
			})
		}

		// Tell agent to start shell
		agentConn.conn.WriteJSON(map[string]interface{}{
			"type": "shell_start",
			"data": map[string]interface{}{
				"session_id":  agentSessionID,
				"new_session": newSession,
			},
		})

		defer func() {
			agentConn.sessionsMu.Lock()
			if current, ok := agentConn.sessions[connectionID]; ok && current == session {
				delete(agentConn.sessions, connectionID)
			}

			// Determine whether we should stop the underlying agent session.
			hasLocal := false
			for _, s := range agentConn.sessions {
				if s != nil && s.AgentSessionID == agentSessionID {
					hasLocal = true
					break
				}
			}
			hasRemote := false
			if refs, ok := agentConn.remoteSessionRefs[agentSessionID]; ok && len(refs) > 0 {
				hasRemote = true
			}
			noLocalSessions := len(agentConn.sessions) == 0
			noRemoteSessions := len(agentConn.remoteSessionRefs) == 0
			agentConn.sessionsMu.Unlock()
			conn.Close()

			// For split panes, stop the specific session when nobody is subscribed anymore.
			if newSession && !hasLocal && !hasRemote {
				agentConn.conn.WriteJSON(map[string]interface{}{
					"type": "shell_stop_session",
					"data": map[string]interface{}{
						"session_id": agentSessionID,
					},
				})
			}

			// If no sessions remain anywhere (local or remote), stop everything.
			if noLocalSessions && noRemoteSessions {
				agentConn.conn.WriteJSON(map[string]interface{}{
					"type": "shell_stop",
				})
			}
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
						"session_id": agentSessionID,
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
							"session_id": agentSessionID,
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
								"session_id": agentSessionID,
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
		if h.pubsubHub == nil {
			conn.Close()
			return
		}

		// Register remote session and only start the agent session once per instance+session.
		shouldStart := false
		var previous *AgentSession
		h.remoteSessionsMu.Lock()
		// Replace on reconnect
		previous = h.remoteSessions[connectionID]
		for _, s := range h.remoteSessions {
			if s != nil && s.AgentID == agentID && s.AgentSessionID == agentSessionID {
				shouldStart = false
				goto registered
			}
		}
		shouldStart = true
	registered:
		h.remoteSessions[connectionID] = session
		h.remoteSessionsMu.Unlock()
		if previous != nil && previous.UserConn != nil {
			previous.UserConn.Close()
		}

		if shouldStart {
			h.pubsubHub.ProxyTerminalData(agentID, agentSessionID, "start_session", nil, 0, 0, newSession)
		}

		defer func() {
			shouldStop := false
			h.remoteSessionsMu.Lock()
			if current, ok := h.remoteSessions[connectionID]; ok && current == session {
				delete(h.remoteSessions, connectionID)
			}
			hasAny := false
			for _, s := range h.remoteSessions {
				if s != nil && s.AgentID == agentID && s.AgentSessionID == agentSessionID {
					hasAny = true
					break
				}
			}
			shouldStop = !hasAny
			h.remoteSessionsMu.Unlock()
			conn.Close()

			if shouldStop {
				h.pubsubHub.ProxyTerminalData(agentID, agentSessionID, "stop_session", nil, 0, 0, newSession)
			}
		}()

		// Forward user messages to Redis
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			if messageType == websocket.BinaryMessage {
				h.pubsubHub.ProxyTerminalData(agentID, agentSessionID, "input", message, 0, 0, false)
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
						h.pubsubHub.ProxyTerminalData(agentID, agentSessionID, "input", []byte(inputStr), 0, 0, false)
					}
				case "resize":
					var resizeMsg struct {
						Cols int `json:"cols"`
						Rows int `json:"rows"`
					}
					// Parse the top-level message again to get cols/rows
					if err := json.Unmarshal(message, &resizeMsg); err == nil {
						h.pubsubHub.ProxyTerminalData(agentID, agentSessionID, "resize", nil, resizeMsg.Cols, resizeMsg.Rows, false)
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
	ctx := c.Request.Context()
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
				if localConn.SystemInfo != nil {
					if hostname, ok := localConn.SystemInfo["hostname"].(string); ok && hostname != "" {
						agentData["hostname"] = hostname
					}
					if region, ok := localConn.SystemInfo["region"].(string); ok && region != "" {
						agentData["region"] = region
					}
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
					if region, ok := agent.SystemInfo["region"].(string); ok && region != "" {
						agentData["region"] = region
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
		"description":  agent.Description,
		"image":        agent.OS + "/" + agent.Arch,
		"status":       "running",
		"session_type": "agent",
		"created_at":   agent.CreatedAt,
		"last_used_at": agent.LastPing,
		"os":           agent.OS,
		"arch":         agent.Arch,
		"shell":        agent.Shell,
		"distro":       agent.Distro,
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
		if region, ok := agent.SystemInfo["region"].(string); ok && region != "" {
			agentData["region"] = region
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

func maxRegisteredAgentsForTier(tier string, subscriptionActive bool) int {
	// Use centralized limits from models
	limits := models.GetUserResourceLimits(tier, subscriptionActive)
	return int(limits.MaxAgents)
}

func maxConcurrentAgentTerminalsForTier(tier string, subscriptionActive bool) int {
	// Keep this aligned with registered limits
	return maxRegisteredAgentsForTier(tier, subscriptionActive)
}

func (h *AgentHandler) userHasAnyAgentSession(userID, agentID string) bool {
	// Local sessions (agent is connected to this instance)
	h.agentsMu.RLock()
	agentConn := h.agents[agentID]
	h.agentsMu.RUnlock()
	if agentConn != nil {
		agentConn.sessionsMu.RLock()
		for _, s := range agentConn.sessions {
			if s != nil && s.UserID == userID && s.UserConn != nil {
				agentConn.sessionsMu.RUnlock()
				return true
			}
		}
		agentConn.sessionsMu.RUnlock()
	}

	// Remote sessions (agent is connected to another instance)
	h.remoteSessionsMu.RLock()
	defer h.remoteSessionsMu.RUnlock()
	for _, s := range h.remoteSessions {
		if s != nil && s.UserID == userID && s.AgentID == agentID && s.UserConn != nil {
			return true
		}
	}
	return false
}

func (h *AgentHandler) countDistinctAgentTerminalsForUser(userID string) int {
	agentIDs := make(map[string]struct{})

	// Local agent sessions on this instance
	h.agentsMu.RLock()
	for id, agentConn := range h.agents {
		if agentConn == nil || agentConn.UserID != userID {
			continue
		}
		agentConn.sessionsMu.RLock()
		hasAny := false
		for _, s := range agentConn.sessions {
			if s != nil && s.UserID == userID && s.UserConn != nil {
				hasAny = true
				break
			}
		}
		agentConn.sessionsMu.RUnlock()
		if hasAny {
			agentIDs[id] = struct{}{}
		}
	}
	h.agentsMu.RUnlock()

	// Remote agent sessions on this instance
	h.remoteSessionsMu.RLock()
	for _, s := range h.remoteSessions {
		if s != nil && s.UserID == userID && s.UserConn != nil {
			agentIDs[s.AgentID] = struct{}{}
		}
	}
	h.remoteSessionsMu.RUnlock()

	return len(agentIDs)
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
