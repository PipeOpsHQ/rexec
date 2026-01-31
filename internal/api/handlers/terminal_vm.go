package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rexec/rexec/internal/providers"
)

// handleVMWebSocket handles WebSocket connections for VM terminals
func (h *TerminalHandler) handleVMWebSocket(c *gin.Context, terminalID, userID string) {
	// Extract VM ID (remove "vm:" prefix)
	vmID := strings.TrimPrefix(terminalID, "vm:")

	// Get provider registry
	registry, ok := h.providerRegistry.(*providers.Registry)
	if !ok || registry == nil {
		log.Printf("[Terminal] Provider registry not available for VM %s", vmID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "provider registry not configured"})
		return
	}

	// Get Firecracker provider
	provider, ok := registry.Get("firecracker")
	if !ok {
		log.Printf("[Terminal] Firecracker provider not available for VM %s", vmID)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "firecracker provider not available"})
		return
	}

	// Verify ownership
	terminal, err := provider.Get(c.Request.Context(), vmID)
	if err != nil {
		log.Printf("[Terminal] VM %s not found: %v", vmID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "VM not found"})
		return
	}

	if terminal.UserID != userID {
		log.Printf("[Terminal] VM %s owned by different user", vmID)
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
		return
	}

	// Get terminal size from query params
	cols := uint16(80)
	rows := uint16(24)
	if c.Query("cols") != "" {
		if c, err := strconv.ParseUint(c.Query("cols"), 10, 16); err == nil {
			cols = uint16(c)
		}
	}
	if c.Query("rows") != "" {
		if r, err := strconv.ParseUint(c.Query("rows"), 10, 16); err == nil {
			rows = uint16(r)
		}
	}

	// Connect to VM terminal via provider
	termConn, err := provider.ConnectTerminal(c.Request.Context(), vmID, cols, rows)
	if err != nil {
		log.Printf("[Terminal] Failed to connect to VM %s: %v", vmID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to VM terminal"})
		return
	}
	defer termConn.Close()

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[Terminal] WebSocket upgrade failed for VM %s: %v", vmID, err)
		return
	}
	defer conn.Close()

	log.Printf("[Terminal] WebSocket connected for VM %s (user: %s)", vmID, userID)

	// Create session
	session := &TerminalSession{
		UserID:      userID,
		ContainerID: terminalID, // Use full ID with "vm:" prefix
		CreatedAt:   time.Now(),
		Conn:        conn,
		Cols:        uint(cols),
		Rows:        uint(rows),
		Done:        make(chan struct{}),
		IsOwner:     true,
	}

	h.mu.Lock()
	h.sessions[terminalID] = session
	h.mu.Unlock()

	// Handle terminal I/O
	go h.handleVMOutput(session, termConn.Reader)
	h.handleVMInput(session, termConn.Writer, termConn.Resize)

	// Cleanup
	h.mu.Lock()
	delete(h.sessions, terminalID)
	h.mu.Unlock()
}

// handleVMOutput forwards output from VM to WebSocket
func (h *TerminalHandler) handleVMOutput(session *TerminalSession, reader io.Reader) {
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		select {
		case <-session.Done:
			return
		default:
			n, err := reader.Read(buf)
			if n > 0 {
				session.mu.Lock()
				if !session.closed {
					if err := session.Conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
						log.Printf("[Terminal] Failed to write VM output: %v", err)
						session.mu.Unlock()
						return
					}
				}
				session.mu.Unlock()
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("[Terminal] VM output read error: %v", err)
				}
				return
			}
		}
	}
}

// handleVMInput forwards input from WebSocket to VM
func (h *TerminalHandler) handleVMInput(session *TerminalSession, writer io.Writer, resize func(uint16, uint16) error) {
	for {
		select {
		case <-session.Done:
			return
		default:
			messageType, data, err := session.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[Terminal] VM WebSocket error: %v", err)
				}
				return
			}

			switch messageType {
			case websocket.TextMessage:
				// Handle JSON messages (resize, etc.)
				var msg TerminalMessage
				if err := json.Unmarshal(data, &msg); err == nil {
					if msg.Type == "resize" && resize != nil {
						if err := resize(uint16(msg.Cols), uint16(msg.Rows)); err != nil {
							log.Printf("[Terminal] Failed to resize VM terminal: %v", err)
						}
						session.Cols = uint(msg.Cols)
						session.Rows = uint(msg.Rows)
					}
					continue
				}
				// Fall through to write as text
				fallthrough
			case websocket.BinaryMessage:
				// Write to VM terminal
				if _, err := writer.Write(data); err != nil {
					log.Printf("[Terminal] Failed to write to VM: %v", err)
					return
				}
			}
		}
	}
}
