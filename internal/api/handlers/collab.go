package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	mgr "github.com/rexec/rexec/internal/container"
	"github.com/rexec/rexec/internal/storage"
)

// CollabHandler manages collaboration sessions
type CollabHandler struct {
	store            *storage.PostgresStore
	containerManager *mgr.Manager
	terminalHandler  *TerminalHandler
	sessions         map[string]*CollabSession // share_code -> session
	mu               sync.RWMutex
}

// CollabSession represents an active collaboration session
type CollabSession struct {
	ID          string
	ContainerID string
	OwnerID     string
	ShareCode   string
	Mode        string // "view" or "control"
	MaxUsers    int
	ExpiresAt   time.Time
	Participants map[string]*CollabParticipant
	broadcast    chan CollabMessage
	mu           sync.RWMutex
}

// CollabParticipant represents a participant in a session
type CollabParticipant struct {
	ID       string
	UserID   string
	Username string
	Role     string // "owner", "editor", "viewer"
	Conn     *websocket.Conn
	Color    string // Cursor color for this participant
}

// CollabMessage represents a message in a collab session
type CollabMessage struct {
	Type        string      `json:"type"` // "join", "leave", "cursor", "selection", "input", "output", "sync", "participants"
	UserID      string      `json:"user_id,omitempty"`
	Username    string      `json:"username,omitempty"`
	Role        string      `json:"role,omitempty"`
	Color       string      `json:"color,omitempty"`
	Data        interface{} `json:"data,omitempty"`
	Timestamp   int64       `json:"timestamp"`
}

// NewCollabHandler creates a new collaboration handler
func NewCollabHandler(store *storage.PostgresStore, cm *mgr.Manager, th *TerminalHandler) *CollabHandler {
	h := &CollabHandler{
		store:            store,
		containerManager: cm,
		terminalHandler:  th,
		sessions:         make(map[string]*CollabSession),
	}

	// Cleanup expired sessions periodically
	go h.cleanupLoop()

	return h
}

// StartSession creates a new collaboration session
func (h *CollabHandler) StartSession(c *gin.Context) {
	userID, _ := c.Get("userID")
	username, _ := c.Get("username")

	var req struct {
		ContainerID string `json:"container_id" binding:"required"`
		Mode        string `json:"mode"` // "view" or "control", default "view"
		MaxUsers    int    `json:"max_users"`
		Duration    int    `json:"duration_minutes"` // Session duration, default 60
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify ownership - handle both containers and agents
	if strings.HasPrefix(req.ContainerID, "agent:") {
		// Agent terminal - verify agent ownership
		agentID := strings.TrimPrefix(req.ContainerID, "agent:")
		agent, err := h.store.GetAgent(c.Request.Context(), agentID)
		if err != nil || agent == nil || agent.UserID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to share this terminal"})
			return
		}
	} else {
		// Docker container - verify container ownership
		container, ok := h.containerManager.GetContainer(req.ContainerID)
		if !ok || container.UserID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to share this terminal"})
			return
		}
	}

	// Check if session already exists for this container
	h.mu.RLock()
	for _, session := range h.sessions {
		if session.ContainerID == req.ContainerID && session.OwnerID == userID.(string) {
			h.mu.RUnlock()
			c.JSON(http.StatusOK, gin.H{
				"session_id": session.ID,
				"share_code": session.ShareCode,
				"share_url":  "/join/" + session.ShareCode,
				"expires_at": session.ExpiresAt,
			})
			return
		}
	}
	h.mu.RUnlock()

	// Set defaults
	if req.Mode == "" {
		req.Mode = "view"
	}
	if req.MaxUsers <= 0 {
		req.MaxUsers = 5
	}
	if req.Duration <= 0 {
		req.Duration = 60
	}

	// Generate share code
	shareCode := generateShareCode()
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(time.Duration(req.Duration) * time.Minute)

	// Create session record
	record := &storage.CollabSessionRecord{
		ID:          sessionID,
		ContainerID: req.ContainerID,
		OwnerID:     userID.(string),
		ShareCode:   shareCode,
		Mode:        req.Mode,
		MaxUsers:    req.MaxUsers,
		IsActive:    true,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
	}

	if err := h.store.CreateCollabSession(c.Request.Context(), record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	// Create in-memory session
	session := &CollabSession{
		ID:           sessionID,
		ContainerID:  req.ContainerID,
		OwnerID:      userID.(string),
		ShareCode:    shareCode,
		Mode:         req.Mode,
		MaxUsers:     req.MaxUsers,
		ExpiresAt:    expiresAt,
		Participants: make(map[string]*CollabParticipant),
		broadcast:    make(chan CollabMessage, 1024),
	}

	// Add owner as first participant
	ownerParticipant := &CollabParticipant{
		ID:       uuid.New().String(),
		UserID:   userID.(string),
		Username: username.(string),
		Role:     "owner",
		Color:    getParticipantColor(0),
	}
	session.Participants[userID.(string)] = ownerParticipant

	h.mu.Lock()
	h.sessions[shareCode] = session
	h.mu.Unlock()

	// Start broadcast goroutine
	go session.broadcastLoop()

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"share_code": shareCode,
		"share_url":  "/join/" + shareCode,
		"expires_at": expiresAt,
		"mode":       req.Mode,
	})
}

// JoinSession allows a user to join a collaboration session
func (h *CollabHandler) JoinSession(c *gin.Context) {
	shareCode := c.Param("code")
	userID, _ := c.Get("userID")
	_, _ = c.Get("username") // username used in websocket handler

	h.mu.RLock()
	session, exists := h.sessions[shareCode]
	h.mu.RUnlock()

	if !exists {
		// Try to load from database
		record, err := h.store.GetCollabSessionByShareCode(c.Request.Context(), shareCode)
		if err != nil || record == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found or expired"})
			return
		}

		// Recreate in-memory session
		session = &CollabSession{
			ID:           record.ID,
			ContainerID:  record.ContainerID,
			OwnerID:      record.OwnerID,
			ShareCode:    record.ShareCode,
			Mode:         record.Mode,
			MaxUsers:     record.MaxUsers,
			ExpiresAt:    record.ExpiresAt,
			Participants: make(map[string]*CollabParticipant),
			broadcast:    make(chan CollabMessage, 1024),
		}

		h.mu.Lock()
		h.sessions[shareCode] = session
		h.mu.Unlock()

		go session.broadcastLoop()
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": "session has expired"})
		return
	}

	// Check max users
	session.mu.RLock()
	participantCount := len(session.Participants)
	session.mu.RUnlock()

	if participantCount >= session.MaxUsers {
		c.JSON(http.StatusForbidden, gin.H{"error": "session is full"})
		return
	}

	// Determine role
	role := "viewer"
	if session.Mode == "control" {
		role = "editor"
	}
	if userID.(string) == session.OwnerID {
		role = "owner"
	}

	// Pre-register the user as a participant so they have terminal access immediately
	// This allows the terminal WebSocket to connect before the collab WebSocket
	session.mu.Lock()
	var participantID string
	if existingParticipant, exists := session.Participants[userID.(string)]; !exists {
		username, _ := c.Get("username")
		usernameStr := ""
		if username != nil {
			usernameStr = username.(string)
		}
		colorIndex := len(session.Participants)
		participantID = uuid.New().String()
		session.Participants[userID.(string)] = &CollabParticipant{
			ID:       participantID,
			UserID:   userID.(string),
			Username: usernameStr,
			Role:     role,
			Conn:     nil, // Will be set when WebSocket connects
			Color:    getParticipantColor(colorIndex),
		}
	} else {
		participantID = existingParticipant.ID
	}
	session.mu.Unlock()

	// Also store participant in database for persistence
	h.store.AddCollabParticipant(c.Request.Context(), &storage.CollabParticipantRecord{
		ID:        participantID,
		SessionID: session.ID,
		UserID:    userID.(string),
		Username:  func() string { if u, _ := c.Get("username"); u != nil { return u.(string) }; return "" }(),
		Role:      role,
		JoinedAt:  time.Now(),
	})

	// Get container info for better display
	containerName := session.ContainerID
	if len(containerName) > 12 {
		containerName = containerName[:12] // Default to truncated ID
	}
	if strings.HasPrefix(session.ContainerID, "agent:") {
		agentID := strings.TrimPrefix(session.ContainerID, "agent:")
		if agent, err := h.store.GetAgent(c.Request.Context(), agentID); err == nil && agent != nil {
			containerName = agent.Name
		}
	} else if container, err := h.store.GetContainerByDockerID(c.Request.Context(), session.ContainerID); err == nil && container != nil {
		containerName = container.Name
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":     session.ID,
		"container_id":   session.ContainerID,
		"container_name": containerName,
		"mode":           session.Mode,
		"role":           role,
		"expires_at":     session.ExpiresAt,
	})
}

// HandleCollabWebSocket handles WebSocket connections for collaboration
func (h *CollabHandler) HandleCollabWebSocket(c *gin.Context) {
	shareCode := c.Param("code")
	userID, _ := c.Get("userID")
	username, _ := c.Get("username")

	h.mu.RLock()
	session, exists := h.sessions[shareCode]
	h.mu.RUnlock()

	if !exists {
		// Try to load from database (session might have been lost on server restart)
		record, err := h.store.GetCollabSessionByShareCode(c.Request.Context(), shareCode)
		if err != nil || record == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}

		// Check expiration before recreating
		if time.Now().After(record.ExpiresAt) {
			c.JSON(http.StatusGone, gin.H{"error": "session has expired"})
			return
		}

		// Recreate in-memory session from database
		session = &CollabSession{
			ID:           record.ID,
			ContainerID:  record.ContainerID,
			OwnerID:      record.OwnerID,
			ShareCode:    record.ShareCode,
			Mode:         record.Mode,
			MaxUsers:     record.MaxUsers,
			ExpiresAt:    record.ExpiresAt,
			Participants: make(map[string]*CollabParticipant),
			broadcast:    make(chan CollabMessage, 1024),
		}

		h.mu.Lock()
		h.sessions[shareCode] = session
		h.mu.Unlock()

		go session.broadcastLoop()
		log.Printf("[Collab] Restored session %s from database", shareCode)
	}

	// Upgrade to WebSocket with subprotocol support
	responseHeader := http.Header{}
	requestedProtocols := c.GetHeader("Sec-WebSocket-Protocol")
	if strings.Contains(requestedProtocols, "rexec.v1") {
		responseHeader.Set("Sec-WebSocket-Protocol", "rexec.v1")
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, responseHeader)
	if err != nil {
		log.Printf("Collab WebSocket upgrade failed: %v", err)
		return
	}

	// Determine role and color
	role := "viewer"
	if session.Mode == "control" {
		role = "editor"
	}
	if userID.(string) == session.OwnerID {
		role = "owner"
	}

	session.mu.Lock()
	// Check if participant was pre-registered via JoinSession REST API
	var participant *CollabParticipant
	if existingParticipant, exists := session.Participants[userID.(string)]; exists {
		// Update existing participant with WebSocket connection
		existingParticipant.Conn = conn
		participant = existingParticipant
	} else {
		// New participant connecting directly via WebSocket
		colorIndex := len(session.Participants)
		participant = &CollabParticipant{
			ID:       uuid.New().String(),
			UserID:   userID.(string),
			Username: username.(string),
			Role:     role,
			Conn:     conn,
			Color:    getParticipantColor(colorIndex),
		}
		session.Participants[userID.(string)] = participant
	}
	session.mu.Unlock()

	// Store participant in database (upsert behavior - safe to call multiple times)
	h.store.AddCollabParticipant(c.Request.Context(), &storage.CollabParticipantRecord{
		ID:        participant.ID,
		SessionID: session.ID,
		UserID:    userID.(string),
		Username:  username.(string),
		Role:      role,
		JoinedAt:  time.Now(),
	})

	// Broadcast join message (non-blocking)
	select {
	case session.broadcast <- CollabMessage{
		Type:      "join",
		UserID:    userID.(string),
		Username:  username.(string),
		Role:      role,
		Color:     participant.Color,
		Timestamp: time.Now().UnixMilli(),
	}:
	default:
		log.Printf("Collab broadcast channel full, dropping join message for %s", userID)
	}

	// Send current participants list
	h.sendParticipantsList(session, conn)

	// Handle messages
	defer func() {
		session.mu.Lock()
		delete(session.Participants, userID.(string))
		session.mu.Unlock()

		h.store.RemoveCollabParticipant(c.Request.Context(), session.ID, userID.(string))

		// Broadcast leave message (non-blocking)
		select {
		case session.broadcast <- CollabMessage{
			Type:      "leave",
			UserID:    userID.(string),
			Username:  username.(string),
			Timestamp: time.Now().UnixMilli(),
		}:
		default:
			log.Printf("Collab broadcast channel full, dropping leave message for %s", userID)
		}

		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg CollabMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		msg.UserID = userID.(string)
		msg.Username = username.(string)
		msg.Color = participant.Color
		msg.Timestamp = time.Now().UnixMilli()

		// Handle different message types
		switch msg.Type {
		case "cursor":
			// Broadcast cursor position to all except sender
			h.broadcastExcept(session, msg, userID.(string))

		case "input":
			// Only allow input from owners and editors
			if role == "owner" || role == "editor" {
				select {
				case session.broadcast <- msg:
				default:
					log.Printf("Collab broadcast channel full, dropping input message for session %s", shareCode)
				}
			}

		case "selection":
			// Broadcast text selection
			h.broadcastExcept(session, msg, userID.(string))

		default:
			// Non-blocking broadcast
			select {
			case session.broadcast <- msg:
			default:
				// Channel full, drop message to prevent blocking
				log.Printf("Collab broadcast channel full, dropping message type %s for session %s", msg.Type, shareCode)
			}
		}
	}
}

// EndSession ends a collaboration session
func (h *CollabHandler) EndSession(c *gin.Context) {
	sessionID := c.Param("id")
	userID, _ := c.Get("userID")

	h.mu.Lock()
	defer h.mu.Unlock()

	// First try to find in-memory session
	for code, session := range h.sessions {
		if session.ID == sessionID {
			if session.OwnerID != userID.(string) {
				c.JSON(http.StatusForbidden, gin.H{"error": "only owner can end session"})
				return
			}

			participantIDs := make([]string, 0, len(session.Participants))
			session.mu.RLock()
			for pid := range session.Participants {
				participantIDs = append(participantIDs, pid)
			}
			session.mu.RUnlock()

			// Close all participant connections
			session.mu.Lock()
			for _, p := range session.Participants {
				if p.Conn != nil {
					p.Conn.WriteJSON(CollabMessage{
						Type:      "ended",
						Data:      "Session ended by owner",
						Timestamp: time.Now().UnixMilli(),
					})
				}
			}
			session.mu.Unlock()
			
			// Give time for messages to be delivered before closing connections
			time.Sleep(100 * time.Millisecond)
			
			session.mu.Lock()
			for _, p := range session.Participants {
				if p.Conn != nil {
					p.Conn.Close()
				}
			}
			session.mu.Unlock()

			// Mark as inactive in database
			h.store.EndCollabSession(c.Request.Context(), sessionID)

			delete(h.sessions, code)

			// For control mode, terminate collaborator terminal sessions (docker containers only).
			if session.Mode == "control" && h.terminalHandler != nil && !strings.HasPrefix(session.ContainerID, "agent:") {
				h.terminalHandler.CleanupControlCollab(session.ContainerID, session.OwnerID, participantIDs)
			}

			c.JSON(http.StatusOK, gin.H{"message": "session ended"})
			return
		}
	}

	// Fallback: Try to find session in database (may exist if server restarted)
	dbSession, err := h.store.GetCollabSessionByID(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to lookup session"})
		return
	}
	
	if dbSession != nil {
		// Verify ownership
		if dbSession.OwnerID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only owner can end session"})
			return
		}
		
		// Mark as inactive in database
		if err := h.store.EndCollabSession(c.Request.Context(), sessionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to end session"})
			return
		}

		// For control mode, terminate collaborator terminal sessions (docker containers only).
		if dbSession.Mode == "control" && h.terminalHandler != nil && !strings.HasPrefix(dbSession.ContainerID, "agent:") {
			participants, err := h.store.GetCollabParticipants(c.Request.Context(), sessionID)
			if err == nil {
				participantIDs := make([]string, 0, len(participants))
				for _, p := range participants {
					if p != nil && p.UserID != "" {
						participantIDs = append(participantIDs, p.UserID)
					}
				}
				h.terminalHandler.CleanupControlCollab(dbSession.ContainerID, dbSession.OwnerID, participantIDs)
			}
		}
		
		c.JSON(http.StatusOK, gin.H{"message": "session ended"})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
}

// GetActiveSessions returns all active sessions for a user
func (h *CollabHandler) GetActiveSessions(c *gin.Context) {
	userID, _ := c.Get("userID")

	h.mu.RLock()
	defer h.mu.RUnlock()

	var sessions []gin.H
	for _, session := range h.sessions {
		if session.OwnerID == userID.(string) {
			session.mu.RLock()
			participantCount := len(session.Participants)
			session.mu.RUnlock()

			sessions = append(sessions, gin.H{
				"id":           session.ID,
				"share_code":   session.ShareCode,
				"container_id": session.ContainerID,
				"mode":         session.Mode,
				"participants": participantCount,
				"max_users":    session.MaxUsers,
				"expires_at":   session.ExpiresAt,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// Helper functions

func (s *CollabSession) broadcastLoop() {
	for msg := range s.broadcast {
		s.mu.RLock()
		for _, p := range s.Participants {
			if p.Conn != nil {
				// Use a short deadline to prevent slow clients from blocking the broadcast
				p.Conn.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
				if err := p.Conn.WriteJSON(msg); err != nil {
					log.Printf("Failed to write to collab participant %s: %v", p.UserID, err)
					// Don't close here, let the read loop handle disconnection or next write try
				}
			}
		}
		s.mu.RUnlock()
	}
}

func (h *CollabHandler) broadcastExcept(session *CollabSession, msg CollabMessage, exceptUserID string) {
	session.mu.RLock()
	defer session.mu.RUnlock()

	for userID, p := range session.Participants {
		if userID != exceptUserID && p.Conn != nil {
			p.Conn.WriteJSON(msg)
		}
	}
}

func (h *CollabHandler) sendParticipantsList(session *CollabSession, conn *websocket.Conn) {
	session.mu.RLock()
	defer session.mu.RUnlock()

	var participants []gin.H
	for _, p := range session.Participants {
		participants = append(participants, gin.H{
			"user_id":  p.UserID,
			"username": p.Username,
			"role":     p.Role,
			"color":    p.Color,
		})
	}

	conn.WriteJSON(CollabMessage{
		Type:      "participants",
		Data:      participants,
		Timestamp: time.Now().UnixMilli(),
	})
}

func (h *CollabHandler) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.Lock()
		for code, session := range h.sessions {
			if time.Now().After(session.ExpiresAt) {
				participantIDs := make([]string, 0, len(session.Participants))
				session.mu.RLock()
				for pid := range session.Participants {
					participantIDs = append(participantIDs, pid)
				}
				session.mu.RUnlock()

				// Close all connections
				session.mu.Lock()
				for _, p := range session.Participants {
					if p.Conn != nil {
						p.Conn.WriteJSON(CollabMessage{
							Type:      "expired",
							Timestamp: time.Now().UnixMilli(),
						})
						p.Conn.Close()
					}
				}
				session.mu.Unlock()

				// Mark as inactive in database (best-effort)
				_ = h.store.EndCollabSession(context.Background(), session.ID)

				// For control mode, terminate collaborator terminal sessions (docker containers only).
				if session.Mode == "control" && h.terminalHandler != nil && !strings.HasPrefix(session.ContainerID, "agent:") {
					h.terminalHandler.CleanupControlCollab(session.ContainerID, session.OwnerID, participantIDs)
				}

				delete(h.sessions, code)
			}
		}
		h.mu.Unlock()
	}
}

func generateShareCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	code := base64.URLEncoding.EncodeToString(b)
	return strings.ToUpper(code[:6])
}

func getParticipantColor(index int) string {
	colors := []string{
		"#FF6B6B", // Red
		"#4ECDC4", // Teal
		"#45B7D1", // Blue
		"#96CEB4", // Green
		"#FFEAA7", // Yellow
		"#DDA0DD", // Plum
		"#98D8C8", // Mint
		"#F7DC6F", // Gold
	}
	return colors[index%len(colors)]
}
