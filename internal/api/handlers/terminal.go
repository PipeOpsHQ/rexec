package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	mgr "github.com/rexec/rexec/internal/container"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  64 * 1024,  // 64KB - handles large paste operations
	WriteBufferSize: 64 * 1024,  // 64KB - handles large output bursts
	CheckOrigin: func(r *http.Request) bool {
		// In production, you should validate the origin
		return true
	},
	HandshakeTimeout: 10 * time.Second,
	EnableCompression: true, // Enable per-message compression for large data
}

// TerminalHandler handles WebSocket terminal connections
type TerminalHandler struct {
	containerManager *mgr.Manager
	sessions         map[string]*TerminalSession
	mu               sync.RWMutex
	recordingHandler *RecordingHandler
	collabHandler    *CollabHandler
}

// TerminalSession represents an active terminal session
type TerminalSession struct {
	UserID      string
	ContainerID string
	ExecID      string
	Conn        *websocket.Conn
	Cols        uint
	Rows        uint
	Done        chan struct{}
	mu          sync.Mutex
	closed      bool
}

// TerminalMessage represents messages between client and server
type TerminalMessage struct {
	Type string `json:"type"` // "input", "output", "resize", "ping", "pong", "error", "connected"
	Data string `json:"data,omitempty"`
	Cols uint   `json:"cols,omitempty"`
	Rows uint   `json:"rows,omitempty"`
}

// NewTerminalHandler creates a new terminal handler
func NewTerminalHandler(cm *mgr.Manager) *TerminalHandler {
	h := &TerminalHandler{
		containerManager: cm,
		sessions:         make(map[string]*TerminalSession),
	}

	// Start keepalive goroutine
	go h.keepAliveLoop()

	return h
}

// SetRecordingHandler sets the recording handler for capturing terminal output
func (h *TerminalHandler) SetRecordingHandler(rh *RecordingHandler) {
	h.recordingHandler = rh
}

// SetCollabHandler sets the collab handler to check for shared session access
func (h *TerminalHandler) SetCollabHandler(ch *CollabHandler) {
	h.collabHandler = ch
}

// HasCollabAccess checks if a user has collab access to a container
func (h *TerminalHandler) HasCollabAccess(userID, containerID string) bool {
	if h.collabHandler == nil {
		return false
	}
	
	h.collabHandler.mu.RLock()
	defer h.collabHandler.mu.RUnlock()
	
	for _, session := range h.collabHandler.sessions {
		if session.ContainerID == containerID {
			session.mu.RLock()
			_, hasAccess := session.Participants[userID]
			session.mu.RUnlock()
			if hasAccess {
				return true
			}
		}
	}
	return false
}

// HandleWebSocket handles WebSocket connections for terminal access
func (h *TerminalHandler) HandleWebSocket(c *gin.Context) {
	dockerID := c.Param("containerId")
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Verify user owns this container
	containerInfo, ok := h.containerManager.GetContainer(dockerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error":           "container not found",
			"code":            "container_not_found",
			"hint":            "Container may need to be recreated. Try starting it.",
			"action_required": "start",
		})
		return
	}

	// Verify ownership
	// Verify ownership or collab access
	if containerInfo.UserID != userID.(string) {
		// Check if user has collab access
		if !h.HasCollabAccess(userID.(string), dockerID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
	}

	// Check if container actually exists in Docker (may have been removed externally)
	ctx := context.Background()
	_, err := h.containerManager.GetClient().ContainerInspect(ctx, dockerID)
	if err != nil {
		if dockerclient.IsErrNotFound(err) {
			c.JSON(http.StatusGone, gin.H{
				"error":           "container was removed from server",
				"code":            "container_removed",
				"hint":            "Container was removed from Docker. Click Start to recreate it.",
				"action_required": "start",
				"container_id":    dockerID,
			})
			return
		}
		// Other Docker errors
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "failed to verify container status: " + err.Error(),
			"code":  "docker_error",
		})
		return
	}

	// Check if container is running
	if containerInfo.Status != "running" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           "container is not running",
			"code":            "container_stopped",
			"status":          containerInfo.Status,
			"hint":            "Start the container before connecting to terminal",
			"action_required": "start",
		})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Configure WebSocket for large data handling
	conn.SetReadLimit(1024 * 1024) // 1MB max message size for large pastes
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(120 * time.Second)) // Longer timeout for stability
		return nil
	})
	// Enable compression if client supports it
	conn.EnableWriteCompression(true)
	conn.SetCompressionLevel(6) // Balance between speed and compression

	// Create terminal session
	session := &TerminalSession{
		UserID:      userID.(string),
		ContainerID: dockerID,
		Conn:        conn,
		Cols:        80,
		Rows:        24,
		Done:        make(chan struct{}),
	}

	// Register session
	sessionKey := dockerID + ":" + userID.(string)
	h.mu.Lock()
	// Close existing session if any
	if existingSession, exists := h.sessions[sessionKey]; exists {
		existingSession.Close()
	}
	h.sessions[sessionKey] = session
	h.mu.Unlock()

	// Touch container to update last used time
	h.containerManager.TouchContainer(dockerID)

	// Cleanup on exit
	defer func() {
		h.mu.Lock()
		delete(h.sessions, sessionKey)
		h.mu.Unlock()
		session.Close()
	}()

	// Send connected message
	session.SendMessage(TerminalMessage{
		Type: "connected",
		Data: "Terminal session established",
	})

	// Start terminal session with auto-restart on exit
	h.runTerminalSessionWithRestart(session, containerInfo.ImageType)
}

// runTerminalSessionWithRestart runs the terminal session and restarts the shell if user types 'exit'
func (h *TerminalHandler) runTerminalSessionWithRestart(session *TerminalSession, imageType string) {
	// Start stats streaming
	statsCtx, statsCancel := context.WithCancel(context.Background())
	defer statsCancel()

	go func() {
		statsCh := make(chan mgr.ContainerResourceStats)
		go func() {
			if err := h.containerManager.StreamContainerStats(statsCtx, session.ContainerID, statsCh); err != nil {
				log.Printf("Stats streaming ended: %v", err)
			}
		}()

		for stats := range statsCh {
			statsData, _ := json.Marshal(stats)
			session.SendMessage(TerminalMessage{
				Type: "stats",
				Data: string(statsData),
			})
		}
	}()

	maxRestarts := 10 // Prevent infinite restart loops
	restartCount := 0

	for {
		select {
		case <-session.Done:
			return
		default:
		}

		// Check if container is still running
		if _, ok := h.containerManager.GetContainer(session.ContainerID); !ok {
			session.SendMessage(TerminalMessage{
				Type: "error",
				Data: "Container is no longer available",
			})
			return
		}

		// Run the terminal session
		shellExited := h.runTerminalSession(session, imageType)

		// If shell exited normally (user typed 'exit'), restart it
		if shellExited && restartCount < maxRestarts {
			restartCount++
			session.SendMessage(TerminalMessage{
				Type: "output",
				Data: "\r\n\x1b[33m[Shell exited. Starting new session...]\x1b[0m\r\n\r\n",
			})
			continue
		}

		// Connection closed or too many restarts
		break
	}
}

// runTerminalSession manages the terminal session lifecycle
// Returns true if the shell exited normally (user typed 'exit'), false otherwise
func (h *TerminalHandler) runTerminalSession(session *TerminalSession, imageType string) bool {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := h.containerManager.GetClient()

	// Detect available shell
	shell := h.detectShell(ctx, session.ContainerID, imageType)

	// Create exec instance
	execConfig := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{shell},
		Env: []string{
			"TERM=xterm-256color",
			"COLORTERM=truecolor",
		},
	}

	execResp, err := client.ContainerExecCreate(ctx, session.ContainerID, execConfig)
	if err != nil {
		session.SendError("Failed to create terminal session: " + err.Error())
		return false
	}

	session.ExecID = execResp.ID

	// Attach to exec (this also starts it for Podman compatibility)
	// ContainerExecAttach implicitly starts the exec session
	attachResp, err := client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{
		Tty: true,
	})
	if err != nil {
		session.SendError("Failed to attach to terminal: " + err.Error())
		return false
	}
	defer attachResp.Close()

	// Handle bidirectional communication
	var wg sync.WaitGroup
	wg.Add(2)

	// Error channel to capture any errors
	// shellExitChan signals when shell exits normally (EOF)
	errChan := make(chan error, 2)
	shellExitChan := make(chan bool, 1)

	// Container output -> WebSocket
	go func() {
		defer wg.Done()
		// Large buffer for handling big output bursts (e.g., cat large files, log dumps)
		buf := make([]byte, 64*1024) // 64KB buffer
		// Accumulator for batching small outputs to reduce WebSocket messages
		accumulator := make([]byte, 0, 64*1024)
		flushTicker := time.NewTicker(16 * time.Millisecond) // ~60fps max update rate
		defer flushTicker.Stop()
		
		flushAccumulator := func() {
			if len(accumulator) > 0 {
				outputData := sanitizeUTF8(accumulator)
				if err := session.SendMessage(TerminalMessage{
					Type: "output",
					Data: outputData,
				}); err != nil {
					errChan <- err
					return
				}
				// Record output if recording is active
				if h.recordingHandler != nil {
					h.recordingHandler.AddEvent(session.ContainerID, "o", outputData, 0, 0)
				}
				accumulator = accumulator[:0] // Reset without reallocating
			}
		}
		
		for {
			select {
			case <-session.Done:
				flushAccumulator()
				return
			case <-ctx.Done():
				flushAccumulator()
				return
			case <-flushTicker.C:
				// Periodic flush to ensure responsiveness
				flushAccumulator()
			default:
				// Set read deadline
				attachResp.Conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))

				n, err := attachResp.Reader.Read(buf)
				if err != nil {
					if err == io.EOF {
						flushAccumulator()
						shellExitChan <- true
						return
					}
					// Timeout is expected, continue
					if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
						continue
					}
					flushAccumulator()
					errChan <- err
					return
				}
				if n > 0 {
					accumulator = append(accumulator, buf[:n]...)
					// Flush immediately if buffer is getting large (>32KB)
					if len(accumulator) > 32*1024 {
						flushAccumulator()
					}
				}
			}
		}
	}()

	// WebSocket -> Container input
	go func() {
		defer wg.Done()
		for {
			select {
			case <-session.Done:
				return
			case <-ctx.Done():
				return
			default:
				_, message, err := session.Conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Printf("WebSocket error: %v", err)
					}
					cancel()
					return
				}

				var msg TerminalMessage
				if err := json.Unmarshal(message, &msg); err != nil {
					// Treat as raw input for backward compatibility
					attachResp.Conn.Write(message)
					h.containerManager.TouchContainer(session.ContainerID)
					continue
				}

				switch msg.Type {
				case "input":
					if _, err := attachResp.Conn.Write([]byte(msg.Data)); err != nil {
						log.Printf("Failed to write to container: %v", err)
						return
					}
					// Touch container on input
					h.containerManager.TouchContainer(session.ContainerID)
					// Record input if recording is active
					if h.recordingHandler != nil {
						h.recordingHandler.AddEvent(session.ContainerID, "i", msg.Data, 0, 0)
					}

				case "resize":
					if msg.Cols > 0 && msg.Rows > 0 {
						session.mu.Lock()
						session.Cols = msg.Cols
						session.Rows = msg.Rows
						session.mu.Unlock()

						if err := client.ContainerExecResize(ctx, session.ExecID, container.ResizeOptions{
							Height: msg.Rows,
							Width:  msg.Cols,
						}); err != nil {
							log.Printf("Failed to resize terminal: %v", err)
						}
						// Record resize if recording is active
						if h.recordingHandler != nil {
							h.recordingHandler.AddEvent(session.ContainerID, "r", "", int(msg.Cols), int(msg.Rows))
						}
					}

				case "ping":
					session.SendMessage(TerminalMessage{Type: "pong"})

				case "pong":
					// Received pong, connection is alive
				}
			}
		}
	}()

	// Wait for either goroutine to finish
	go func() {
		wg.Wait()
		cancel()
	}()

	// Wait for context cancellation or session done
	select {
	case <-ctx.Done():
		return false
	case <-session.Done:
		return false
	case <-shellExitChan:
		// Shell exited normally, can restart
		return true
	case err := <-errChan:
		if err != nil {
			log.Printf("Terminal session error: %v", err)
		}
		return false
	}
}

// detectShell finds the best available shell in the container
// Prefers zsh if oh-my-zsh is installed, then bash, then sh
func (h *TerminalHandler) detectShell(ctx context.Context, containerID, imageType string) string {
	// First, check if zsh with oh-my-zsh is set up (best experience)
	if h.isZshSetup(ctx, containerID) {
		// Verify zsh exists
		if h.shellExists(ctx, containerID, "/bin/zsh") {
			return "/bin/zsh"
		}
		if h.shellExists(ctx, containerID, "/usr/bin/zsh") {
			return "/usr/bin/zsh"
		}
	}

	// Define shell preference order based on image type
	var shells []string
	switch imageType {
	case "alpine", "alpine-3.18":
		shells = []string{"/bin/zsh", "/bin/ash", "/bin/sh"}
	case "ubuntu", "ubuntu-24", "ubuntu-20", "debian", "debian-11", "kali", "parrot":
		shells = []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
	case "fedora", "fedora-39", "centos", "rocky", "alma", "oracle", "amazonlinux":
		shells = []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
	case "archlinux", "opensuse", "gentoo", "void", "nixos":
		shells = []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
	default:
		shells = []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
	}

	for _, shell := range shells {
		if h.shellExists(ctx, containerID, shell) {
			return shell
		}
	}

	// Last resort - /bin/sh should always exist
	return "/bin/sh"
}

// isZshSetup checks if oh-my-zsh is installed in the container
func (h *TerminalHandler) isZshSetup(ctx context.Context, containerID string) bool {
	client := h.containerManager.GetClient()

	execConfig := container.ExecOptions{
		Cmd:          []string{"/bin/sh", "-c", "test -d /root/.oh-my-zsh && test -f /root/.zshrc"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return false
	}

	client.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{})
	inspect, err := client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return false
	}

	return inspect.ExitCode == 0
}

// shellExists checks if a shell exists and is executable
func (h *TerminalHandler) shellExists(ctx context.Context, containerID, shell string) bool {
	client := h.containerManager.GetClient()

	execConfig := container.ExecOptions{
		Cmd:          []string{"test", "-x", shell},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return false
	}

	client.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{})
	inspect, err := client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return false
	}

	return inspect.ExitCode == 0
}

// SendMessage sends a message to the WebSocket client
func (s *TerminalSession) SendMessage(msg TerminalMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return s.Conn.WriteJSON(msg)
}

// SendError sends an error message to the client
func (s *TerminalSession) SendError(message string) {
	log.Printf("Terminal error: %s", message)
	s.SendMessage(TerminalMessage{
		Type: "error",
		Data: message,
	})
}

// Close closes the terminal session
func (s *TerminalSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	s.closed = true

	// Signal done
	select {
	case <-s.Done:
	default:
		close(s.Done)
	}

	// Close WebSocket with proper close message
	s.Conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "session ended"),
	)
	s.Conn.Close()
}

// GetActiveSession returns an active session by session key
func (h *TerminalHandler) GetActiveSession(containerID, userID string) (*TerminalSession, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	sessionKey := containerID + ":" + userID
	session, ok := h.sessions[sessionKey]
	return session, ok
}

// CloseSession closes a terminal session
func (h *TerminalHandler) CloseSession(containerID, userID string) {
	h.mu.Lock()
	sessionKey := containerID + ":" + userID
	session, ok := h.sessions[sessionKey]
	if ok {
		delete(h.sessions, sessionKey)
	}
	h.mu.Unlock()

	if ok && session != nil {
		session.Close()
	}
}

// CloseAllContainerSessions closes all sessions for a container
func (h *TerminalHandler) CloseAllContainerSessions(containerID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for key, session := range h.sessions {
		if session.ContainerID == containerID {
			session.Close()
			delete(h.sessions, key)
		}
	}
}

// keepAliveLoop sends periodic pings to keep connections alive
func (h *TerminalHandler) keepAliveLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.RLock()
		for _, session := range h.sessions {
			session.SendMessage(TerminalMessage{Type: "ping"})
		}
		h.mu.RUnlock()
	}
}

// GetSessionCount returns the number of active sessions
func (h *TerminalHandler) GetSessionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sessions)
}

// GetContainerSessionCount returns the number of active sessions for a container
func (h *TerminalHandler) GetContainerSessionCount(containerID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, session := range h.sessions {
		if session.ContainerID == containerID {
			count++
		}
	}
	return count
}

// sanitizeUTF8 ensures the byte slice is valid UTF-8, replacing invalid sequences
// while preserving terminal escape sequences (which are valid ASCII/UTF-8)
func sanitizeUTF8(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}

	// Build a valid UTF-8 string, replacing invalid bytes
	result := make([]byte, 0, len(data))
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			// Invalid byte - replace with replacement character or skip
			// For terminal output, we'll just skip invalid bytes to avoid display issues
			data = data[1:]
			continue
		}
		result = append(result, data[:size]...)
		data = data[size:]
	}
	return string(result)
}
