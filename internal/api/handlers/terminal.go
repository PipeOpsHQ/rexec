package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	mgr "github.com/rexec/rexec/internal/container"
	"github.com/rexec/rexec/internal/storage"
	admin_events "github.com/rexec/rexec/internal/api/handlers/admin_events"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  128 * 1024, // 128KB - handles large paste operations
	WriteBufferSize: 128 * 1024, // 128KB - handles large output bursts
	CheckOrigin: func(r *http.Request) bool {
		// In production, you should validate the origin
		return true
	},
	HandshakeTimeout:  10 * time.Second,
	EnableCompression: true, // Enable per-message compression for large data
}

// Terminal optimization constants
const (
	// PTY buffer size - larger buffer for fewer syscalls
	ptyBufferSize = 64 * 1024 // 64KB
	// WebSocket write deadline - prevent slow client blocking
	wsWriteDeadline = 5 * time.Second
	// Minimum bytes before sending (reduces small packet overhead)
	minOutputSize = 1
	// Maximum time to wait for more output before sending
	outputCoalesceTimeout = 2 * time.Millisecond
)

// TerminalHandler handles WebSocket terminal connections
type TerminalHandler struct {
	containerManager *mgr.Manager
	store            *storage.PostgresStore
	sessions         map[string]*TerminalSession
	sharedSessions   map[string]*SharedTerminalSession // containerID -> shared session for collab
	mu               sync.RWMutex
	recordingHandler *RecordingHandler
	collabHandler    *CollabHandler
	adminEventsHub   *admin_events.AdminEventsHub // New field
}

// SharedTerminalSession represents a terminal session shared by multiple users (for collaboration)
type SharedTerminalSession struct {
	ContainerID string
	ExecID      string
	OwnerID     string
	Connections map[string]*websocket.Conn // userID -> connection
	Cols        uint
	Rows        uint
	Done        chan struct{}
	InputChan   chan []byte // Channel for input from any participant
	OutputChan  chan []byte // Channel for output to broadcast
	mu          sync.RWMutex
	closed      bool
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
func NewTerminalHandler(cm *mgr.Manager, store *storage.PostgresStore, adminEventsHub *admin_events.AdminEventsHub) *TerminalHandler {
	h := &TerminalHandler{
		containerManager: cm,
		store:            store,
		sessions:         make(map[string]*TerminalSession),
		sharedSessions:   make(map[string]*SharedTerminalSession),
		adminEventsHub: adminEventsHub, // Assign the hub
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
	// We allow connections during configuring state so users can connect during long role setups
	if containerInfo.Status != "running" && containerInfo.Status != "configuring" {
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

	// Persist session to database for admin visibility
	go func() {
		// Generate a proper UUID for the session ID to fit VARCHAR(36)
		sessionID := uuid.New().String()
		dbSession := &storage.SessionRecord{
			ID:          sessionID,
			UserID:      userID.(string),
			ContainerID: dockerID,
			CreatedAt:   time.Now(),
			LastPingAt:  time.Now(),
		}
		if err := h.store.CreateSession(context.Background(), dbSession); err != nil {
			log.Printf("Failed to create db session: %v", err)
		}
		// Broadcast session created event to admin hub
		if h.adminEventsHub != nil {
			h.adminEventsHub.Broadcast("session_created", dbSession)
		}
	}()

	// Cleanup on exit
	defer func() {
		h.mu.Lock()
		// Only delete if it's still OUR session (race condition protection)
		if currentSession, exists := h.sessions[sessionKey]; exists && currentSession == session {
			delete(h.sessions, sessionKey)
		}
		h.mu.Unlock()
		
		// Broadcast session deleted event to admin hub
		if h.adminEventsHub != nil {
			h.adminEventsHub.Broadcast("session_deleted", gin.H{"id": sessionKey, "user_id": userID.(string), "container_id": dockerID})
		}

		// Remove from database
		go func() {
			if err := h.store.DeleteSession(context.Background(), sessionKey); err != nil {
				log.Printf("Failed to delete db session: %v", err)
			}
		}()

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

	// macOS containers are VMs - they have their own internal shell management
	// Don't aggressively restart shells for macOS as it causes instability
	isMacOS := strings.Contains(strings.ToLower(imageType), "macos") || strings.Contains(strings.ToLower(imageType), "osx")

	maxRestarts := 10 // Prevent infinite restart loops
	if isMacOS {
		maxRestarts = 3 // Fewer restarts for macOS VMs, but allow a few for CPU throttle recovery
	}
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

		// Reset restart count if session lasted > 1 minute (5 min for macOS VMs)
		minSessionDuration := 1 * time.Minute
		if isMacOS {
			minSessionDuration = 2 * time.Minute // Reduced from 5 min - macOS VMs can disconnect due to CPU throttling
		}
		if time.Since(startTime) > minSessionDuration {
			restartCount = 0
		}

		// If shell exited normally (user typed 'exit'), restart it
		// For macOS, add a delay to let the VM stabilize
		if shellExited && restartCount < maxRestarts {
			restartCount++

			if isMacOS {
				session.SendMessage(TerminalMessage{
					Type: "output",
					Data: "\r\n\x1b[33m[Shell exited. Waiting for VM to stabilize (high CPU can cause disconnections)...]\x1b[0m\r\n",
				})
				time.Sleep(5 * time.Second) // Give macOS VM more time to stabilize after CPU throttling
			}

			session.SendMessage(TerminalMessage{
				Type: "output",
				Data: "\r\n\x1b[33m[Starting new session...]\x1b[0m\r\n\r\n",
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
		if isMacOS && restartCount >= maxRestarts {
			session.SendMessage(TerminalMessage{
				Type: "output",
				Data: "\r\n\x1b[31m[macOS VM connection unstable due to high CPU usage. Try reconnecting or restarting the container.]\x1b[0m\r\n",
			})
		}
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

	// Check if this is a macOS container
	isMacOS := strings.Contains(strings.ToLower(imageType), "macos") || strings.Contains(strings.ToLower(imageType), "osx")

	// Create exec instance
	// Use login shell (-l) to ensure .zshrc/.bashrc is sourced
	execConfig := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{shell, "-l"},
		Env: []string{
			"TERM=xterm-256color",
			"COLORTERM=truecolor",
			"PATH=/root/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
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

	// Set initial terminal size immediately after attach
	// This is critical for TUI applications like opencode that need proper dimensions
	session.mu.Lock()
	initialCols := session.Cols
	initialRows := session.Rows
	session.mu.Unlock()

	if initialCols > 0 && initialRows > 0 {
		if err := client.ContainerExecResize(ctx, execResp.ID, container.ResizeOptions{
			Height: initialRows,
			Width:  initialCols,
		}); err != nil {
			log.Printf("[Terminal] Initial resize failed for %s: %v", session.ContainerID[:12], err)
		}
	} else {
		// Default size if not set
		if err := client.ContainerExecResize(ctx, execResp.ID, container.ResizeOptions{
			Height: 24,
			Width:  80,
		}); err != nil {
			log.Printf("[Terminal] Default resize failed for %s: %v", session.ContainerID[:12], err)
		}
	}

	// Handle bidirectional communication
	var wg sync.WaitGroup
	wg.Add(2)

	// Error channel to capture any errors
	// shellExitChan signals when shell exits normally (EOF)
	errChan := make(chan error, 2)
	shellExitChan := make(chan bool, 1)

	// Container output -> WebSocket (optimized for low latency)
	go func() {
		defer wg.Done()
		// Large buffer for efficient reads
		buf := make([]byte, ptyBufferSize)

		// Track consecutive EOFs for macOS stability
		consecutiveEOFs := 0
		maxConsecutiveEOFs := 1
		if isMacOS {
			maxConsecutiveEOFs = 3 // macOS VMs can have transient EOFs
		}

		for {
			select {
			case <-session.Done:
				return
			case <-ctx.Done():
				return
			default:
				// No read deadline - blocking read is most efficient for PTY
				attachResp.Conn.SetReadDeadline(time.Time{})

				n, err := attachResp.Reader.Read(buf)
				if err != nil {
					if err == io.EOF {
						consecutiveEOFs++
						if consecutiveEOFs >= maxConsecutiveEOFs {
							shellExitChan <- true
							return
						}
						// For macOS, wait a bit and retry on first EOF
						if isMacOS && consecutiveEOFs < maxConsecutiveEOFs {
							time.Sleep(500 * time.Millisecond)
							continue
						}
						shellExitChan <- true
						return
					}
					// Filter out normal closure errors
					if !strings.Contains(err.Error(), "use of closed network connection") {
						errChan <- err
					}
					return
				}

				// Reset EOF counter on successful read
				consecutiveEOFs = 0

				if n > 0 {
					// Fast path: check if already valid UTF-8 (common case)
					outputData := string(buf[:n])
					if !isValidUTF8Fast(buf[:n]) {
						outputData = sanitizeUTF8(buf[:n])
					}
					// Filter mouse tracking sequences
					outputData = filterMouseTracking(outputData)

					// Set write deadline to prevent slow clients from blocking
					session.Conn.SetWriteDeadline(time.Now().Add(wsWriteDeadline))

					if err := session.SendMessage(TerminalMessage{
						Type: "output",
						Data: outputData,
					}); err != nil {
						// WebSocket closed or write timeout
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

	// Check if this is a macOS container
	isMacOS := strings.Contains(strings.ToLower(imageType), "macos") || strings.Contains(strings.ToLower(imageType), "osx")

	// Define shell preference order based on image type
	var shells []string
	switch {
	case isMacOS:
		// macOS VM - use bash or zsh, don't probe too aggressively as it's a VM
		shells = []string{"/bin/bash", "/bin/zsh", "/bin/sh"}
	case imageType == "alpine" || imageType == "alpine-3.18":
		shells = []string{"/bin/zsh", "/bin/ash", "/bin/sh"}
	case imageType == "ubuntu" || imageType == "ubuntu-24" || imageType == "ubuntu-20" ||
		imageType == "debian" || imageType == "debian-11" || imageType == "kali" || imageType == "parrot":
		shells = []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
	case imageType == "fedora" || imageType == "fedora-39" || imageType == "centos" ||
		imageType == "rocky" || imageType == "alma" || imageType == "oracle" || imageType == "amazonlinux":
		shells = []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
	case imageType == "archlinux" || imageType == "opensuse" || imageType == "gentoo" ||
		imageType == "void" || imageType == "nixos":
		shells = []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
	default:
		shells = []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
	}

	// For macOS, skip shell existence checks - just return bash
	// The VM handles its own shell availability
	if isMacOS {
		return "/bin/bash"
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

	// Check both /root and /home/user (standard rexec user home)
	cmd := "test -f /root/.zshrc || test -f /home/user/.zshrc"
	
	execConfig := container.ExecOptions{
		Cmd:          []string{"/bin/sh", "-c", cmd},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return false
	}

	// Use ContainerExecAttach instead of ContainerExecStart for Podman compatibility
	attachResp, err := client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return false
	}
	attachResp.Close()

	inspect, err := client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return false
	}

	return inspect.ExitCode == 0
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
	// Aggressively kills processes holding locks to ensure terminal usability
	cleanupScript := `#!/bin/sh
# Check for lock files
for lockfile in /var/lib/dpkg/lock-frontend /var/lib/dpkg/lock /var/lib/apt/lists/lock; do
    if [ -f "$lockfile" ]; then
        # Try to find and kill the process holding the lock
        if command -v fuser >/dev/null 2>&1; then
            fuser -k -KILL "$lockfile" >/dev/null 2>&1 || true
        fi
        # Remove the lock file
        rm -f "$lockfile" 2>/dev/null || true
        cleanup_needed=true
    fi
done

# Run dpkg configure to fix interrupted installs
if [ "$cleanup_needed" = "true" ] || [ -f /var/lib/dpkg/updates/0000 ]; then
    export DEBIAN_FRONTEND=noninteractive
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

	// Use login shell (-l) to ensure .zshrc/.bashrc is sourced
	execConfig := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{shell, "-l"},
		Env: []string{
			"TERM=xterm-256color",
			"COLORTERM=truecolor",
			"PATH=/root/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
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
		for key, session := range h.sessions {
			session.SendMessage(TerminalMessage{Type: "ping"})
			
			// Update database timestamp for admin visibility
			// Capture variables for goroutine
			sessionID := key
			go func() {
				if err := h.store.UpdateSessionLastPing(context.Background(), sessionID); err != nil {
					// Log verbose only on error to avoid spam
					// log.Printf("Failed to update session ping: %v", err)
				}
				// Broadcast session_updated event to admin hub
				if h.adminEventsHub != nil {
					// Fetch the updated session record to send full payload
					updatedSession, err := h.store.GetSessionByID(context.Background(), sessionID)
					if err == nil && updatedSession != nil {
						h.adminEventsHub.Broadcast("session_updated", updatedSession)
					} else {
						log.Printf("Warning: Failed to fetch updated session record for admin broadcast: %v", err)
					}
				}
			}()
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

// isValidUTF8Fast does a quick check if the data is valid UTF-8
// This is faster than utf8.Valid for the common case of ASCII-only data
func isValidUTF8Fast(data []byte) bool {
	// Quick check: if all bytes are ASCII, it's valid UTF-8
	for _, b := range data {
		if b >= 0x80 {
			// Has non-ASCII bytes, do full validation
			return utf8.Valid(data)
		}
	}
	return true
}

// filterMouseTracking removes SGR mouse tracking sequences from terminal output
// These sequences can cause display issues when split across WebSocket messages
// SGR format: \x1b[<button;x;yM or \x1b[<button;x;ym
func filterMouseTracking(data string) string {
	// Fast path - check for escape character first (most common case: no escapes)
	hasEscape := false
	for i := 0; i < len(data); i++ {
		if data[i] == '\x1b' {
			hasEscape = true
			break
		}
	}
	if !hasEscape {
		return data
	}

	// Check for mouse sequence specifically
	if !strings.Contains(data, "\x1b[<") {
		return data
	}

	// Remove complete SGR mouse sequences
	result := strings.Builder{}
	result.Grow(len(data))

	i := 0
	for i < len(data) {
		// Check for SGR mouse sequence start
		if i+2 < len(data) && data[i] == '\x1b' && data[i+1] == '[' && data[i+2] == '<' {
			// Find the end of the sequence (M or m)
			j := i + 3
			for j < len(data) && ((data[j] >= '0' && data[j] <= '9') || data[j] == ';') {
				j++
			}
			// Check if it ends with M or m
			if j < len(data) && (data[j] == 'M' || data[j] == 'm') {
				// Skip this mouse sequence
				i = j + 1
				continue
			}
		}
		result.WriteByte(data[i])
		i++
	}

	return result.String()
}

// sanitizeUTF8 ensures the byte slice is valid UTF-8, replacing invalid sequences
// while preserving terminal escape sequences (which are valid ASCII/UTF-8)
func sanitizeUTF8(data []byte) string {
	// Fast path - already valid (common case)
	if utf8.Valid(data) {
		return string(data)
	}

	// Build a valid UTF-8 string, skipping invalid bytes
	result := make([]byte, 0, len(data))
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			// Invalid byte - skip it for clean terminal output
			data = data[1:]
			continue
		}
		result = append(result, data[:size]...)
		data = data[size:]
	}
	return string(result)
}
