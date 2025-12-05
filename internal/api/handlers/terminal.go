package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
	"unicode/utf8"
	"strings"

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
	sharedSessions   map[string]*SharedTerminalSession // containerID -> shared session for collab
	mu               sync.RWMutex
	recordingHandler *RecordingHandler
	collabHandler    *CollabHandler
}

// SharedTerminalSession represents a terminal session shared by multiple users (for collaboration)
type SharedTerminalSession struct {
	ContainerID   string
	ExecID        string
	OwnerID       string
	Connections   map[string]*websocket.Conn // userID -> connection
	Cols          uint
	Rows          uint
	Done          chan struct{}
	InputChan     chan []byte              // Channel for input from any participant
	OutputChan    chan []byte              // Channel for output to broadcast
	mu            sync.RWMutex
	closed        bool
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
		sharedSessions:   make(map[string]*SharedTerminalSession),
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
	containerIdOrName := c.Param("containerId")
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Verify user owns this container (lookup by Docker ID or terminal name)
	containerInfo, ok := h.containerManager.GetContainer(containerIdOrName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error":           "container not found",
			"code":            "container_not_found",
			"hint":            "Container may need to be recreated. Try starting it.",
			"action_required": "start",
		})
		return
	}
	
	// Use the actual Docker ID from container info
	dockerID := containerInfo.ID
	isOwner := containerInfo.UserID == userID.(string)
	isCollabUser := false

	// Verify ownership
	// Verify ownership or collab access
	if !isOwner {
		// Check if user has collab access
		if !h.HasCollabAccess(userID.(string), dockerID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		isCollabUser = true
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

	// Touch container to update last used time
	h.containerManager.TouchContainer(dockerID)

	// Check if there's an active collab session for this container
	h.mu.RLock()
	sharedSession, hasSharedSession := h.sharedSessions[dockerID]
	h.mu.RUnlock()

	// If collab user joining, they MUST use shared session
	if isCollabUser {
		if hasSharedSession && !sharedSession.closed {
			h.joinSharedSession(sharedSession, conn, userID.(string), false)
		} else {
			// No shared session exists. Check if owner is connected in a private session.
			ownerID := containerInfo.UserID
			ownerSessionKey := dockerID + ":" + ownerID
			
			h.mu.Lock()
			ownerSession, ownerConnected := h.sessions[ownerSessionKey]
			h.mu.Unlock()

			if ownerConnected {
				log.Printf("[Terminal] Upgrading owner %s to shared session for container %s", ownerID, dockerID)
				
				// 1. Force owner to reconnect (which will join the shared session we are about to create)
				// We send a specific close message or just close it.
				ownerSession.Conn.WriteJSON(TerminalMessage{
					Type: "reconnect", 
					Data: "Upgrading to shared session...",
				})
				ownerSession.Close()
				
				// 2. Create the shared session immediately
				// The owner isn't in it yet, but will be when they reconnect.
				sharedSession = h.getOrCreateSharedSession(dockerID, ownerID, containerInfo.ImageType)
				
				// 3. Join the collab user now
				h.joinSharedSession(sharedSession, conn, userID.(string), false)
				return
			}

			// No shared session exists, and owner not connected
			conn.WriteJSON(TerminalMessage{
				Type: "error",
				Data: "Session owner must connect first",
			})
			conn.Close()
		}
		return
	}

	// Owner connecting - check if there's an active collab that needs shared session
	if hasSharedSession && !sharedSession.closed {
		// Join existing shared session
		h.joinSharedSession(sharedSession, conn, userID.(string), isOwner)
		return
	}

	// Check if owner is starting while there's an active collab session
	if h.hasActiveCollabSession(dockerID) {
		// Create shared session for collab
		sharedSession = h.getOrCreateSharedSession(dockerID, userID.(string), containerInfo.ImageType)
		h.joinSharedSession(sharedSession, conn, userID.(string), isOwner)
		return
	}

	// Regular non-collab session
	// Support multiple connections via unique client-provided ID
	connectionID := c.Query("id")
	if connectionID == "" {
		// Fallback for old clients or single sessions
		connectionID = "default"
	}

	session := &TerminalSession{
		UserID:      userID.(string),
		ContainerID: dockerID,
		Conn:        conn,
		Cols:        80,
		Rows:        24,
		Done:        make(chan struct{}),
	}

	// Register session with unique key to allow multiplexing
	sessionKey := dockerID + ":" + userID.(string) + ":" + connectionID
	h.mu.Lock()
	// Close existing session ONLY if it shares the exact same ID (reconnection)
	if existingSession, exists := h.sessions[sessionKey]; exists {
		existingSession.Close()
	}
	h.sessions[sessionKey] = session
	h.mu.Unlock()

	// Cleanup on exit
	defer func() {
		h.mu.Lock()
		// Only delete if it's still OUR session (race condition protection)
		if currentSession, exists := h.sessions[sessionKey]; exists && currentSession == session {
			delete(h.sessions, sessionKey)
		}
		h.mu.Unlock()
		session.Close()
	}()

	// Send connected message
	session.SendMessage(TerminalMessage{
		Type: "connected",
		Data: "Terminal session established",
	})

	// Kill any orphaned package manager processes from previous sessions
	// This prevents apt-get lock issues after reconnects/deployments
	go h.cleanupOrphanedPackageProcesses(dockerID)

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
		startTime := time.Now()
		shellExited := h.runTerminalSession(session, imageType)

		// Reset restart count if session lasted > 1 minute
		if time.Since(startTime) > 1*time.Minute {
			restartCount = 0
		}

		// If shell exited normally (user typed 'exit'), restart it
		if shellExited && restartCount < maxRestarts {
			restartCount++
			session.SendMessage(TerminalMessage{
				Type: "output",
				Data: "\r\n\x1b[33m[Shell exited. Starting new session...]\x1b[0m\r\n\r\n",
			})

			// Check if container stopped (since shell was likely PID 1)
			ctx := context.Background()
			inspect, err := h.containerManager.GetClient().ContainerInspect(ctx, session.ContainerID)
			if err != nil || !inspect.State.Running {
				// Container doesn't exist or is not running
				// Try to start it, which may recreate it with a new ID
				session.SendMessage(TerminalMessage{
					Type: "output",
					Data: "\r\n\x1b[33m[Container stopped. Restarting...]\x1b[0m\r\n",
				})
				
				if err := h.containerManager.StartContainer(ctx, session.ContainerID); err != nil {
					log.Printf("Failed to auto-restart container %s: %v", session.ContainerID, err)
					// Container might have been recreated with a new ID
					// Send a special message telling frontend to refresh and reconnect
					session.SendMessage(TerminalMessage{
						Type: "container_restart_required",
						Data: "Container needs to be restarted. Please reconnect.",
					})
					// Close with a special code so frontend knows to look up new container ID
					session.CloseWithCode(4100, "container_restart_required")
					return
				}
				
				// Wait a moment for container to fully start
				time.Sleep(500 * time.Millisecond)
			}

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
		// 32KB buffer is sufficient for PTY
		buf := make([]byte, 32*1024)

		for {
			select {
			case <-session.Done:
				return
			case <-ctx.Done():
				return
			default:
				// Blocking read - no timeout, wait for PTY to send data
				// This is most efficient and lowest latency for interactive use
				attachResp.Conn.SetReadDeadline(time.Time{})

				n, err := attachResp.Reader.Read(buf)
				if err != nil {
					if err == io.EOF {
						shellExitChan <- true
						return
					}
					// Filter out normal closure errors
					if !strings.Contains(err.Error(), "use of closed network connection") {
						errChan <- err
					}
					return
				}

				if n > 0 {
					// Sanitize UTF-8 to prevent garbled output in TUI applications
					outputData := sanitizeUTF8(buf[:n])
					
					if err := session.SendMessage(TerminalMessage{
						Type: "output",
						Data: outputData,
					}); err != nil {
						// WebSocket closed
						return
					}

					if h.recordingHandler != nil {
						h.recordingHandler.AddEvent(session.ContainerID, "o", outputData, 0, 0)
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
		log.Printf("[Terminal] isZshSetup: exec create failed for %s: %v", containerID[:12], err)
		return false
	}

	// Must attach to start the exec (Podman compatibility)
	attachResp, err := client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		log.Printf("[Terminal] isZshSetup: exec attach failed for %s: %v", containerID[:12], err)
		return false
	}
	attachResp.Close()

	// Poll exec status instead of fixed sleep - much faster for quick commands
	for i := 0; i < 20; i++ { // Max 200ms total
		inspect, err := client.ContainerExecInspect(ctx, execResp.ID)
		if err != nil {
			log.Printf("[Terminal] isZshSetup: exec inspect failed for %s: %v", containerID[:12], err)
			return false
		}
		if !inspect.Running {
			isSetup := inspect.ExitCode == 0
			log.Printf("[Terminal] isZshSetup: container %s = %v (exit code: %d, poll: %d)", containerID[:12], isSetup, inspect.ExitCode, i)
			return isSetup
		}
		time.Sleep(10 * time.Millisecond)
	}
	log.Printf("[Terminal] isZshSetup: timeout waiting for exec in %s", containerID[:12])
	return false
}

// shellExists checks if a shell exists and is executable
func (h *TerminalHandler) shellExists(ctx context.Context, containerID, shell string) bool {
	client := h.containerManager.GetClient()

	// Use /bin/sh -c to run test command - more portable across distros
	// Some minimal distros don't have standalone 'test' binary
	execConfig := container.ExecOptions{
		Cmd:          []string{"/bin/sh", "-c", fmt.Sprintf("test -x %s", shell)},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		log.Printf("[Terminal] shellExists: exec create failed for %s in %s: %v", shell, containerID[:12], err)
		return false
	}

	// Must attach to start the exec (Podman compatibility)
	attachResp, err := client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		log.Printf("[Terminal] shellExists: exec attach failed for %s in %s: %v", shell, containerID[:12], err)
		return false
	}
	attachResp.Close()

	// Poll exec status instead of fixed sleep - much faster for quick commands
	for i := 0; i < 20; i++ { // Max 200ms total
		inspect, err := client.ContainerExecInspect(ctx, execResp.ID)
		if err != nil {
			log.Printf("[Terminal] shellExists: exec inspect failed for %s in %s: %v", shell, containerID[:12], err)
			return false
		}
		if !inspect.Running {
			exists := inspect.ExitCode == 0
			log.Printf("[Terminal] shellExists: %s in %s = %v (exit code: %d, poll: %d)", shell, containerID[:12], exists, inspect.ExitCode, i)
			return exists
		}
		time.Sleep(10 * time.Millisecond)
	}
	log.Printf("[Terminal] shellExists: timeout waiting for exec in %s", containerID[:12])
	return false
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
	s.CloseWithCode(websocket.CloseNormalClosure, "session ended")
}

// CloseWithCode closes the session with a specific code and reason
func (s *TerminalSession) CloseWithCode(code int, reason string) {
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
		websocket.FormatCloseMessage(code, reason),
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

// cleanupOrphanedPackageProcesses kills any orphaned apt/dpkg/yum processes
// that might be holding locks from previous disconnected sessions
func (h *TerminalHandler) cleanupOrphanedPackageProcesses(containerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Script to kill orphaned package manager processes
	// Only kills processes that are holding locks, not all package manager processes
	cleanupScript := `#!/bin/sh
# Only cleanup if there are stale locks with no active fuser
cleanup_needed=false
for lockfile in /var/lib/dpkg/lock-frontend /var/lib/dpkg/lock /var/lib/apt/lists/lock; do
    if [ -f "$lockfile" ] && ! fuser "$lockfile" >/dev/null 2>&1; then
        cleanup_needed=true
        rm -f "$lockfile" 2>/dev/null || true
    fi
done

# Only run dpkg configure if cleanup was needed
if [ "$cleanup_needed" = "true" ]; then
    dpkg --configure -a >/dev/null 2>&1 || true
fi
exit 0
`

	execConfig := container.ExecOptions{
		Cmd:          []string{"/bin/sh", "-c", cleanupScript},
		AttachStdout: false,
		AttachStderr: false,
		Tty:          false,
	}

	execResp, err := h.containerManager.GetClient().ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		// Don't log error - this is a best-effort cleanup
		return
	}

	if err := h.containerManager.GetClient().ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{}); err != nil {
		return
	}
}

// hasActiveCollabSession checks if there's an active collab session for a container
func (h *TerminalHandler) hasActiveCollabSession(containerID string) bool {
	if h.collabHandler == nil {
		return false
	}
	h.collabHandler.mu.RLock()
	defer h.collabHandler.mu.RUnlock()
	
	for _, session := range h.collabHandler.sessions {
		if session.ContainerID == containerID && len(session.Participants) > 0 {
			return true
		}
	}
	return false
}

// getOrCreateSharedSession gets or creates a shared terminal session for collaboration
func (h *TerminalHandler) getOrCreateSharedSession(containerID, ownerID, imageType string) *SharedTerminalSession {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if existing, ok := h.sharedSessions[containerID]; ok && !existing.closed {
		return existing
	}
	
	sharedSession := &SharedTerminalSession{
		ContainerID: containerID,
		OwnerID:     ownerID,
		Connections: make(map[string]*websocket.Conn),
		Cols:        80,
		Rows:        24,
		Done:        make(chan struct{}),
		InputChan:   make(chan []byte, 4096),
		OutputChan:  make(chan []byte, 4096),
	}
	
	h.sharedSessions[containerID] = sharedSession
	
	// Start the shared exec session
	go h.runSharedTerminalSession(sharedSession, imageType)
	
	return sharedSession
}

// joinSharedSession adds a connection to a shared terminal session
func (h *TerminalHandler) joinSharedSession(session *SharedTerminalSession, conn *websocket.Conn, userID string, isOwner bool) {
	session.mu.Lock()
	// Close existing connection for this user if any
	if oldConn, exists := session.Connections[userID]; exists {
		oldConn.Close()
	}
	session.Connections[userID] = conn
	session.mu.Unlock()
	
	// Send connected message
	conn.WriteJSON(TerminalMessage{
		Type: "connected",
		Data: "Joined shared terminal session",
	})
	
	// Handle this connection's input
	defer func() {
		session.mu.Lock()
		delete(session.Connections, userID)
		session.mu.Unlock()
		conn.Close()
	}()
	
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}
		
		var msg TerminalMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}
		
		switch msg.Type {
		case "input":
			// Only owner and editors can send input
			if isOwner || h.canSendInput(session.ContainerID, userID) {
				session.InputChan <- []byte(msg.Data)
			}
		case "resize":
			if isOwner {
				session.Cols = msg.Cols
				session.Rows = msg.Rows
				// Resize will be handled by the exec session
			}
		case "ping":
			conn.WriteJSON(TerminalMessage{Type: "pong"})
		}
	}
}

// canSendInput checks if a user can send input (is editor in collab session)
func (h *TerminalHandler) canSendInput(containerID, userID string) bool {
	if h.collabHandler == nil {
		return false
	}
	h.collabHandler.mu.RLock()
	defer h.collabHandler.mu.RUnlock()
	
	for _, session := range h.collabHandler.sessions {
		if session.ContainerID == containerID {
			if p, ok := session.Participants[userID]; ok {
				return p.Role == "owner" || p.Role == "editor"
			}
		}
	}
	return false
}

// runSharedTerminalSession manages a shared terminal session for collaboration
func (h *TerminalHandler) runSharedTerminalSession(session *SharedTerminalSession, imageType string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		session.mu.Lock()
		session.closed = true
		// Close all connections
		for _, conn := range session.Connections {
			conn.WriteJSON(TerminalMessage{
				Type: "error",
				Data: "Shared session ended",
			})
			conn.Close()
		}
		session.Connections = make(map[string]*websocket.Conn)
		session.mu.Unlock()
		
		h.mu.Lock()
		delete(h.sharedSessions, session.ContainerID)
		h.mu.Unlock()
	}()
	
	client := h.containerManager.GetClient()
	shell := h.detectShell(ctx, session.ContainerID, imageType)
	
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
		session.broadcastError("Failed to create shared terminal: " + err.Error())
		return
	}
	
	session.ExecID = execResp.ID
	
	attachResp, err := client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{
		Tty: true,
	})
	if err != nil {
		session.broadcastError("Failed to attach to terminal: " + err.Error())
		return
	}
	defer attachResp.Close()
	
	var wg sync.WaitGroup
	wg.Add(2)
	
	// Read output and broadcast to all connections
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, err := attachResp.Reader.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				session.broadcastOutput(buf[:n])
			}
		}
	}()
	
	// Write input from any participant
	go func() {
		defer wg.Done()
		for {
			select {
			case <-session.Done:
				return
			case input := <-session.InputChan:
				attachResp.Conn.Write(input)
			}
		}
	}()
	
	wg.Wait()
}

// broadcastOutput sends terminal output to all connected participants
func (s *SharedTerminalSession) broadcastOutput(data []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	msg := TerminalMessage{
		Type: "output",
		Data: string(data),
	}
	
	for _, conn := range s.Connections {
		conn.WriteJSON(msg)
	}
}

// broadcastError sends an error to all connected participants
func (s *SharedTerminalSession) broadcastError(errMsg string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	msg := TerminalMessage{
		Type: "error",
		Data: errMsg,
	}
	
	for _, conn := range s.Connections {
		conn.WriteJSON(msg)
	}
}

// keepAliveLoop sends periodic pings to keep connections alive
func (h *TerminalHandler) keepAliveLoop() {
	ticker := time.NewTicker(15 * time.Second) // More frequent pings for connection stability
	defer ticker.Stop()

	for range ticker.C {
		h.mu.RLock()
		// Ping regular sessions
		for _, session := range h.sessions {
			session.SendMessage(TerminalMessage{Type: "ping"})
		}
		// Ping shared session connections
		for _, sharedSession := range h.sharedSessions {
			sharedSession.mu.RLock()
			for _, conn := range sharedSession.Connections {
				conn.WriteJSON(TerminalMessage{Type: "ping"})
			}
			sharedSession.mu.RUnlock()
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
