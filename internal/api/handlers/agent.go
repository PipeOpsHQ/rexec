package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rexec/rexec/internal/storage"
)

// AgentHandler handles external agent connections
type AgentHandler struct {
	store       *storage.PostgresStore
	agents      map[string]*AgentConnection
	agentsMu    sync.RWMutex
	upgrader    websocket.Upgrader
	jwtSecret   []byte
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
	SystemInfo  map[string]interface{} `json:"system_info,omitempty"`
	Stats       map[string]interface{} `json:"stats,omitempty"`
}

type AgentSession struct {
	ID        string
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
		store:     store,
		agents:    make(map[string]*AgentConnection),
		jwtSecret: []byte(secret),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
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

	// Handle messages
	defer func() {
		h.agentsMu.Lock()
		delete(h.agents, agentID)
		h.agentsMu.Unlock()
		conn.Close()
		log.Printf("Agent disconnected: %s (%s)", agent.Name, agentID)
	}()

	// Set up ping/pong
	conn.SetPongHandler(func(string) error {
		agentConn.LastPing = time.Now()
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
				agentConn.sessionsMu.RLock()
				for _, session := range agentConn.sessions {
					session.UserConn.WriteJSON(outputMsg)
				}
				agentConn.sessionsMu.RUnlock()
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

	// Check if agent is online
	h.agentsMu.RLock()
	agentConn, ok := h.agents[agentID]
	h.agentsMu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not online"})
		return
	}

	// Verify ownership
	if agentConn.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
		return
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
		UserConn:  conn,
		CreatedAt: time.Now(),
	}

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

	// Tell agent to start shell
	agentConn.conn.WriteJSON(map[string]interface{}{
		"type": "shell_start",
		"data": map[string]string{
			"session_id": sessionID,
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
				// Forward text input to agent as shell_input
				// msg.Data is a JSON-encoded string, need to unmarshal it first
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
				// Frontend sends cols/rows at top level, extract and forward
				var resizeMsg struct {
					Cols int `json:"cols"`
					Rows int `json:"rows"`
				}
				// Re-unmarshal the original message to get cols/rows
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
