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
	admin_events "github.com/rexec/rexec/internal/api/handlers/admin_events"
	mgr "github.com/rexec/rexec/internal/container"
	"github.com/rexec/rexec/internal/storage"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  32 * 1024, // 32KB - Better for large pastes/vibe coding
	WriteBufferSize: 32 * 1024, // 32KB - Better for large output bursts
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for WebSocket connections.
		// Security is handled by the AuthMiddleware which validates the token
		// before we even reach the WebSocket upgrade. This is essential for
		// the embed widget which can be loaded from ANY third-party domain.
		//
		// The auth middleware ensures:
		// 1. Valid JWT token or API token is present
		// 2. User is authorized to access the container
		//
		// Without a valid token, the request is rejected before reaching this handler.
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
	providerRegistry interface{} // *providers.Registry - will be set via SetProviderRegistry
	sessions         map[string]*TerminalSession
	sharedSessions   map[string]*SharedTerminalSession // containerID -> shared session for collab
	mu               sync.RWMutex
	recordingHandler *RecordingHandler
	collabHandler    *CollabHandler
	adminEventsHub   *admin_events.AdminEventsHub

	// Caches to speed up reconnection
	shellCache map[string]string // containerID -> shell path
	tmuxCache  map[string]bool   // containerID -> hasTmux
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
	UserID          string
	ContainerID     string
	DBSessionID     string
	CreatedAt       time.Time
	ExecID          string
	Conn            *websocket.Conn
	Cols            uint
	Rows            uint
	Done            chan struct{}
	mu              sync.Mutex
	closed          bool
	ForceNewSession bool   // If true, create new tmux session instead of resuming main
	IsOwner         bool   // Container owner (vs collab participant)
	TmuxSessionName string // Set when tmux is used ("main", "user-...", "split-...")
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
		adminEventsHub:   adminEventsHub, // Assign the hub
		shellCache:       make(map[string]string),
		tmuxCache:        make(map[string]bool),
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

// SetProviderRegistry sets the provider registry for VM terminal support
func (h *TerminalHandler) SetProviderRegistry(registry interface{}) {
	h.providerRegistry = registry
}

// HasCollabAccess checks if a user has collab access to a container.
// ctx should be request-scoped so DB lookups cancel on disconnect.
func (h *TerminalHandler) HasCollabAccess(ctx context.Context, userID, containerID string) bool {
	if h.collabHandler == nil {
		log.Printf("[Terminal] HasCollabAccess: collabHandler is nil")
		return false
	}

	// First check in-memory sessions
	h.collabHandler.mu.RLock()
	for _, session := range h.collabHandler.sessions {
		// Check exact match or prefix match (in case IDs are truncated)
		matches := session.ContainerID == containerID
		if !matches && len(containerID) >= 12 && len(session.ContainerID) >= 12 {
			matches = strings.HasPrefix(session.ContainerID, containerID[:12]) ||
				strings.HasPrefix(containerID, session.ContainerID[:12])
		}
		if matches {
			session.mu.RLock()
			_, hasAccess := session.Participants[userID]
			session.mu.RUnlock()
			if hasAccess {
				h.collabHandler.mu.RUnlock()
				log.Printf("[Terminal] HasCollabAccess: user %s has in-memory access to container %s", userID, containerID[:12])
				return true
			}
		}
	}
	h.collabHandler.mu.RUnlock()

	// Fallback: check database for active collab session
	// Try with full containerID first, then with prefix variations
	containerIDs := []string{containerID}
	if len(containerID) >= 64 {
		// Also try short ID
		containerIDs = append(containerIDs, containerID[:12])
	}

	for _, cid := range containerIDs {
		session, err := h.collabHandler.store.GetCollabSessionByContainerID(ctx, cid)
		if err != nil {
			log.Printf("[Terminal] HasCollabAccess: DB error checking session for %s: %v", cid, err)
			continue
		}
		if session == nil {
			continue
		}

		// Check if user is a participant in this session
		participants, err := h.collabHandler.store.GetCollabParticipants(ctx, session.ID)
		if err != nil {
			log.Printf("[Terminal] HasCollabAccess: DB error checking participants: %v", err)
			continue
		}

		for _, p := range participants {
			if p.UserID == userID {
				log.Printf("[Terminal] HasCollabAccess: user %s has DB access to container %s (session %s)", userID, cid, session.ID)
				// Restore session to memory for faster future lookups
				h.restoreCollabSession(session, userID)
				return true
			}
		}
	}

	log.Printf("[Terminal] HasCollabAccess: user %s has NO access to container %s", userID, containerID[:min(12, len(containerID))])
	return false
}

// restoreCollabSession restores a collab session from DB to memory
func (h *TerminalHandler) restoreCollabSession(record *storage.CollabSessionRecord, userID string) {
	if h.collabHandler == nil {
		return
	}

	h.collabHandler.mu.Lock()
	defer h.collabHandler.mu.Unlock()

	// Check if already exists
	if _, exists := h.collabHandler.sessions[record.ShareCode]; exists {
		return
	}

	// Create in-memory session
	session := &CollabSession{
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

	// Add the user as a participant
	session.Participants[userID] = &CollabParticipant{
		ID:       userID,
		UserID:   userID,
		Username: "Participant",
		Role:     "viewer",
		Color:    "#3b82f6",
	}

	h.collabHandler.sessions[record.ShareCode] = session
	go session.broadcastLoop()

	log.Printf("[Terminal] Restored collab session %s from DB", record.ShareCode)
}

// HandleWebSocket handles WebSocket connections for terminal access
func (h *TerminalHandler) HandleWebSocket(c *gin.Context) {
	containerIdOrName := c.Param("containerId")
	userID, exists := c.Get("userID")
	if !exists {
		log.Printf("[Terminal] Unauthorized connection attempt for container %s", containerIdOrName)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	log.Printf("[Terminal] Connection request for container %s (user: %s, ID length: %d)", containerIdOrName, userID, len(containerIdOrName))
	reqCtx := c.Request.Context()

	// Check if this is a VM terminal (starts with "vm:")
	if strings.HasPrefix(containerIdOrName, "vm:") {
		h.handleVMWebSocket(c, containerIdOrName, userID.(string))
		return
	}

	// Verify user owns this container (lookup by Docker ID or terminal name)
	containerInfo, ok := h.containerManager.GetContainer(containerIdOrName)
	var dbContainer *storage.ContainerRecord
	var dockerID string

	if !ok {
		log.Printf("[Terminal] Container %s not in manager cache, checking database...", containerIdOrName)
		// Try to find in database - could be DB UUID or Docker ID
		var err error
		lookupStart := time.Now()

		// Add timeout to prevent hanging on slow DB queries
		dbCtx, dbCancel := context.WithTimeout(reqCtx, 5*time.Second)
		defer dbCancel()

		// Check if it looks like a Docker ID (64 hex chars) or DB UUID (36 chars with dashes)
		if len(containerIdOrName) == 64 {
			// Looks like Docker ID - search by Docker ID with ownership check
			log.Printf("[Terminal] Looking up container by Docker ID: %s", containerIdOrName[:12])
			dbContainer, err = h.store.GetContainerByUserAndDockerID(dbCtx, userID.(string), containerIdOrName)
		} else {
			// Looks like DB UUID - search by DB ID
			log.Printf("[Terminal] Looking up container by DB UUID: %s", containerIdOrName)
			dbContainer, err = h.store.GetContainerByID(dbCtx, containerIdOrName)
			// Verify ownership
			if err == nil && dbContainer != nil && dbContainer.UserID != userID.(string) {
				log.Printf("[Terminal] Container %s found but owned by different user (owner: %s, requester: %s)", containerIdOrName, dbContainer.UserID, userID)
				dbContainer = nil // Not owned by this user
			}
		}
		lookupDuration := time.Since(lookupStart)

		// Check for timeout
		if dbCtx.Err() == context.DeadlineExceeded {
			log.Printf("[Terminal] Database lookup timed out for %s after 3s", containerIdOrName)
			err = dbCtx.Err()
		}

		if err != nil {
			log.Printf("[Terminal] Database lookup failed for %s after %v: %v", containerIdOrName, lookupDuration, err)
		} else if dbContainer == nil {
			log.Printf("[Terminal] Container %s not found in database after %v", containerIdOrName, lookupDuration)
		} else {
			log.Printf("[Terminal] Found container in database: %s (Docker ID: %s) after %v", dbContainer.ID, dbContainer.DockerID[:12], lookupDuration)
		}

		if err == nil && dbContainer != nil && dbContainer.DockerID != "" {
			// Found in DB, try getting from manager using the Docker ID
			// This handles single-replica case efficiently (container in cache)
			if info, found := h.containerManager.GetContainer(dbContainer.DockerID); found {
				log.Printf("[Terminal] Container found in manager cache after DB lookup: %s", dbContainer.DockerID[:12])
				containerInfo = info
				ok = true
			} else {
				// Container exists in DB but not in manager cache
				// This is normal for multi-replica setups or after server restart
				// Quick verify it exists in Docker (fast check, no full sync)
				log.Printf("[Terminal] Container not in manager cache (multi-replica or restart), verifying in Docker: %s", dbContainer.DockerID[:12])
				dockerInspectStart := time.Now()
				_, dockerErr := h.containerManager.GetClient().ContainerInspect(reqCtx, dbContainer.DockerID)
				dockerInspectDuration := time.Since(dockerInspectStart)
				if dockerErr == nil {
					// Log slow Docker API calls
					if dockerInspectDuration > 500*time.Millisecond {
						log.Printf("[Terminal] SLOW Docker inspect for %s: %v - Docker daemon may be busy", dbContainer.DockerID[:12], dockerInspectDuration)
					} else {
						log.Printf("[Terminal] Container verified in Docker after %v: %s", dockerInspectDuration, dbContainer.DockerID[:12])
					}
					// Container exists in Docker - we can proceed with DB record
					// Don't need to load into manager cache for terminal connection
					// The DB record has all info we need
					// Set dockerID now so we can use it later even if containerInfo is nil
					dockerID = dbContainer.DockerID
					ok = true // Mark as found so we skip the slow LoadExistingContainers below
				} else {
					log.Printf("[Terminal] Container not found in Docker after %v: %s (error: %v)", dockerInspectDuration, dbContainer.DockerID[:12], dockerErr)
				}
			}
		}

		// Log slow DB lookups
		if lookupDuration > 500*time.Millisecond {
			log.Printf("[Terminal] SLOW database lookup for %s: %v - DB may be under load", containerIdOrName, lookupDuration)
		}
	}

	if !ok {
		log.Printf("[Terminal] Container %s still not found, attempting Docker sync (last resort)...", containerIdOrName)
		// Only do slow Docker sync if we haven't found it in DB
		// For newly created containers, they should be in cache immediately (single-replica)
		// For multi-replica, container should be found via DB lookup above
		// Only do Docker sync as last resort (e.g., server restart, orphaned container)
		// Use a reasonable timeout for Docker sync
		syncStart := time.Now()
		syncCtx, syncCancel := context.WithTimeout(reqCtx, 5*time.Second)
		if err := h.containerManager.LoadExistingContainers(syncCtx); err != nil {
			log.Printf("[Terminal] Failed to sync containers after %v: %v", time.Since(syncStart), err)
		} else {
			syncDuration := time.Since(syncStart)
			if syncDuration > 2*time.Second {
				log.Printf("[Terminal] SLOW Docker sync completed in %v - consider checking Docker daemon performance", syncDuration)
			} else {
				log.Printf("[Terminal] Docker sync completed in %v", syncDuration)
			}
		}
		syncCancel()
		// Try again after sync
		containerInfo, ok = h.containerManager.GetContainer(containerIdOrName)
		if ok {
			log.Printf("[Terminal] Container found in manager cache after Docker sync: %s", containerIdOrName)
		}
	}

	// If still not found, check if this is a collab user trying to access a container
	// In that case, try to verify via Docker directly
	isCollabUser := false
	isOwner := false

	if !ok {
		log.Printf("[Terminal] Container %s not found after all lookups, checking collab access...", containerIdOrName)
		// If we found container in DB but not in manager, try to use Docker ID from DB
		if dbContainer != nil && dbContainer.DockerID != "" {
			// Verify container exists in Docker
			_, err := h.containerManager.GetClient().ContainerInspect(reqCtx, dbContainer.DockerID)
			if err == nil {
				// Container exists in Docker - allow connection using DB record info
				dockerID = dbContainer.DockerID
				isOwner = dbContainer.UserID == userID.(string)
				log.Printf("[Terminal] Container found in DB but not in cache, using Docker ID %s (user: %s)", dockerID[:12], userID)
				// Set containerInfo to nil - we'll use dockerID directly
				containerInfo = nil
			} else {
				// Container in DB but not in Docker - might be stopped/deleted
				log.Printf("[Terminal] Container in DB but not in Docker: %s (user: %s)", containerIdOrName, userID)
				c.JSON(http.StatusNotFound, gin.H{
					"error":           "container not found",
					"code":            "container_not_found",
					"hint":            "Container may have been stopped or removed. Try starting it.",
					"action_required": "start",
				})
				return
			}
		} else {
			// Check if user has collab access to this container ID
			if h.HasCollabAccess(reqCtx, userID.(string), containerIdOrName) {
				// Verify container exists in Docker directly
				dockerContainer, err := h.containerManager.GetClient().ContainerInspect(reqCtx, containerIdOrName)
				if err != nil {
					log.Printf("[Terminal] Collab container not found in Docker: %s (user: %s)", containerIdOrName, userID)
					c.JSON(http.StatusNotFound, gin.H{
						"error":           "container not found",
						"code":            "container_not_found",
						"hint":            "The shared terminal may no longer exist.",
						"action_required": "none",
					})
					return
				}
				// Container exists in Docker, allow collab access
				dockerID = dockerContainer.ID
				isCollabUser = true
				isOwner = false
				log.Printf("[Terminal] Collab user %s accessing container %s via direct Docker lookup", userID, dockerID[:12])
			} else {
				log.Printf("[Terminal] Container not found after all attempts: %s (user: %s, length: %d)", containerIdOrName, userID, len(containerIdOrName))
				// Check if request was cancelled (timeout)
				if reqCtx.Err() != nil {
					log.Printf("[Terminal] Request context cancelled (likely timeout): %v", reqCtx.Err())
				}
				c.JSON(http.StatusNotFound, gin.H{
					"error":           "container not found",
					"code":            "container_not_found",
					"hint":            "Container may need to be recreated. Try starting it.",
					"action_required": "start",
				})
				return
			}
		}
	} else {
		// Container found (either in manager cache or in DB)
		if containerInfo != nil {
			// Found in manager cache
			dockerID = containerInfo.ID
			isOwner = containerInfo.UserID == userID.(string)

			// Verify ownership or collab access
			if !isOwner {
				// Check if user has collab access
				if !h.HasCollabAccess(reqCtx, userID.(string), dockerID) {
					c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
					return
				}
				isCollabUser = true
			}
		} else if dbContainer != nil && dockerID != "" {
			// Found in DB but not in manager cache - use DB record
			isOwner = dbContainer.UserID == userID.(string)
			log.Printf("[Terminal] Using container from DB (not in cache): %s (user: %s)", dockerID[:12], userID)
		}
	}

	// Get image type - prefer from containerInfo, fallback to DB record, then Docker
	var imageType string
	if containerInfo != nil {
		imageType = containerInfo.ImageType
	} else if dbContainer != nil {
		// Extract image type from DB record's image field (format: "ubuntu" or "custom:image:tag")
		imageType = dbContainer.Image
		if strings.HasPrefix(imageType, "custom:") {
			imageType = "custom"
		}
	}

	// Check container status - always verify from DB/Docker for multi-replica support
	// Manager cache might be stale (container created on different replica)
	// Priority: DB status (most accurate) > Docker state > Manager cache
	var containerStatus string
	var isRunning bool

	// First, try to get status from DB (most accurate, reflects actual setup completion)
	if dbContainer != nil {
		containerStatus = dbContainer.Status
		// DB status "configuring" means setup in progress, "running" means ready
		isRunning = (containerStatus == "running" || containerStatus == "configuring")
		log.Printf("[Terminal] Using DB status for %s: %s (multi-replica safe)", dockerID[:12], containerStatus)
	} else if containerInfo != nil {
		// Fallback to manager cache if DB lookup failed
		containerStatus = containerInfo.Status
		isRunning = (containerStatus == "running" || containerStatus == "configuring")
		log.Printf("[Terminal] Using manager cache status for %s: %s", dockerID[:12], containerStatus)
	}

	// Always verify Docker state as final check (for multi-replica: container might exist but not in this instance's cache)
	if dockerID != "" {
		dockerContainer, err := h.containerManager.GetClient().ContainerInspect(reqCtx, dockerID)
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
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "failed to verify container status: " + err.Error(),
				"code":  "docker_error",
			})
			return
		}

		// Docker is source of truth for running state
		isRunning = dockerContainer.State.Running
		// If DB says "configuring" but Docker says running, container is actually ready
		// (setup completed on another replica, DB might not be updated yet)
		if containerStatus == "configuring" && dockerContainer.State.Running {
			// Check if shell setup is complete - if so, treat as "running"
			if dbContainer != nil && h.store != nil {
				quickCtx, quickCancel := context.WithTimeout(reqCtx, 200*time.Millisecond)
				_, _, setupDone, _ := h.store.GetContainerShellMetadata(quickCtx, dbContainer.ID)
				quickCancel()
				if setupDone {
					containerStatus = "running"
					log.Printf("[Terminal] Container %s setup complete, upgrading status from configuring to running", dockerID[:12])
				}
			}
		}

		// Get image type from Docker labels if not already set
		if imageType == "" && dockerContainer.Config != nil && dockerContainer.Config.Labels != nil {
			if imgType, ok := dockerContainer.Config.Labels["rexec.image_type"]; ok {
				imageType = imgType
			}
		}
	}

	// We allow connections during configuring state so users can connect during long role setups
	if !isRunning && containerStatus != "configuring" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           "container is not running",
			"code":            "container_stopped",
			"status":          containerStatus,
			"hint":            "Start the container before connecting to terminal",
			"action_required": "start",
		})
		return
	}

	// Check if terminal is MFA locked (only for owners, not collab users)
	// Collab users inherit the owner's MFA lock but need to verify through a different flow
	if isOwner && dbContainer != nil && dbContainer.MFALocked {
		c.JSON(http.StatusLocked, gin.H{
			"error":           "terminal is MFA protected",
			"code":            "mfa_required",
			"container_id":    dbContainer.ID,
			"hint":            "This terminal is protected with MFA. Enter your authenticator code to access it.",
			"action_required": "mfa_verify",
		})
		return
	}

	// Upgrade to WebSocket with subprotocol support
	// Client sends: Sec-WebSocket-Protocol: rexec.v1, rexec.token.<token>
	// Server should respond with the accepted protocol version
	responseHeader := http.Header{}
	requestedProtocols := c.GetHeader("Sec-WebSocket-Protocol")
	if strings.Contains(requestedProtocols, "rexec.v1") {
		responseHeader.Set("Sec-WebSocket-Protocol", "rexec.v1")
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, responseHeader)
	if err != nil {
		log.Printf("[Terminal] WebSocket upgrade failed for %s (user %s): %v", containerIdOrName, userID, err)
		return
	}
	log.Printf("[Terminal] WebSocket upgraded successfully for %s (user %s)", containerIdOrName, userID)
	// Configure WebSocket for large data handling
	// Allow up to 100MB messages for "vibe coding" (extreme AI contexts/pastes)
	conn.SetReadLimit(100 * 1024 * 1024)
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

	// Get the collab mode to determine behavior
	collabMode := h.getCollabMode(dockerID)

	// If collab user joining:
	// - View mode: Use shared session (mirrored view, input blocked on frontend)
	// - Control mode: Get their own independent terminal session (own tmux session)
	if isCollabUser {
		// Control mode: Each user gets their own independent terminal session
		if collabMode == "control" {
			log.Printf("[Terminal] Control mode collab user %s getting independent session for %s", userID, dockerID[:12])
			// Fall through to regular session creation below
			// Don't return here - let them create their own session
		} else {
			// View mode: Use shared session (mirrored terminal)
			if hasSharedSession && !sharedSession.closed {
				h.joinSharedSession(sharedSession, conn, userID.(string), false)
			} else {
				// No shared session exists. Check if owner is connected in a private session.
				ownerID := userID.(string)
				if containerInfo != nil {
					ownerID = containerInfo.UserID
				} else if dbContainer != nil {
					ownerID = dbContainer.UserID
				}
				ownerSession, ownerConnected := h.GetActiveSession(dockerID, ownerID)

				if ownerConnected && ownerSession != nil {
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
					sharedSession = h.getOrCreateSharedSession(dockerID, ownerID, imageType)

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
	}

	// Owner connecting - check if there's an active collab that needs shared session
	// Only use shared sessions for VIEW mode, not control mode
	if hasSharedSession && !sharedSession.closed && collabMode == "view" {
		// Join existing shared session
		h.joinSharedSession(sharedSession, conn, userID.(string), isOwner)
		return
	}

	// Check if owner is starting while there's an active VIEW mode collab session
	if h.hasActiveCollabSession(dockerID) && collabMode == "view" {
		// Create shared session for view-mode collab
		sharedSession = h.getOrCreateSharedSession(dockerID, userID.(string), imageType)
		h.joinSharedSession(sharedSession, conn, userID.(string), isOwner)
		return
	}

	// Regular session (non-collab or control-mode collab)
	// Support multiple connections via unique client-provided ID
	connectionID := c.Query("id")
	if connectionID == "" {
		// Fallback for old clients or single sessions
		connectionID = "default"
	}

	// Check if this is a new session request (for split panes)
	// newSession=true means create a fresh tmux session instead of resuming main
	forceNewSession := c.Query("newSession") == "true"

	now := time.Now()
	dbSessionID := uuid.New().String()
	session := &TerminalSession{
		UserID:          userID.(string),
		ContainerID:     dockerID,
		DBSessionID:     dbSessionID,
		CreatedAt:       now,
		Conn:            conn,
		Cols:            80,
		Rows:            24,
		Done:            make(chan struct{}),
		ForceNewSession: forceNewSession,
		IsOwner:         isOwner,
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
	go func(sessionID string, createdAt time.Time) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		dbSession := &storage.SessionRecord{
			ID:          sessionID,
			UserID:      userID.(string),
			ContainerID: dockerID,
			CreatedAt:   createdAt,
			LastPingAt:  createdAt,
		}
		if err := h.store.CreateSession(ctx, dbSession); err != nil {
			log.Printf("Failed to create db session: %v", err)
		}
		// Broadcast session created event to admin hub
		if h.adminEventsHub != nil {
			h.adminEventsHub.Broadcast("session_created", dbSession)
		}
	}(dbSessionID, now)

	// Cleanup on exit
	defer func() {
		h.mu.Lock()
		// Only delete if it's still OUR session (race condition protection)
		if currentSession, exists := h.sessions[sessionKey]; exists && currentSession == session {
			delete(h.sessions, sessionKey)
		}
		h.mu.Unlock()

		// Broadcast session deleted event to admin hub
		if h.adminEventsHub != nil && session.DBSessionID != "" {
			h.adminEventsHub.Broadcast("session_deleted", gin.H{"id": session.DBSessionID})
		}

		// Remove from database
		if session.DBSessionID != "" {
			go func(sessionID string) {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				if err := h.store.DeleteSession(ctx, sessionID); err != nil {
					log.Printf("Failed to delete db session: %v", err)
				}
			}(session.DBSessionID)
		}

		session.Close()

		// Split panes create new tmux sessions; clean them up on disconnect to avoid
		// leaking background shells.
		if session.ForceNewSession && strings.HasPrefix(session.TmuxSessionName, "split-") {
			go h.killTmuxSession(session.ContainerID, session.TmuxSessionName)
		}
	}()

	// Send connected message
	session.SendMessage(TerminalMessage{
		Type: "connected",
		Data: "Terminal session established",
	})
	log.Printf("[Terminal] Sent 'connected' message for session %s (user %s), status=%s", session.ContainerID[:12], userID, containerStatus)

	// Refresh container status from DB and update manager cache if stale (multi-replica support)
	// This ensures the status is accurate even if container was created on another replica
	if dbContainer != nil && containerInfo != nil && containerInfo.Status != dbContainer.Status {
		// Status mismatch - DB is source of truth, update manager cache
		h.containerManager.UpdateContainerStatus(dockerID, dbContainer.Status)
		log.Printf("[Terminal] Updated manager cache status for %s: %s -> %s (from DB)", dockerID[:12], containerInfo.Status, dbContainer.Status)
		containerStatus = dbContainer.Status
	}

	// Send current container status to frontend so it can update UI
	// This helps with multi-replica scenarios where status might have changed
	if containerStatus != "" {
		session.SendMessage(TerminalMessage{
			Type: "container_status",
			Data: containerStatus,
		})
	}

	// Kill any orphaned package manager processes from previous sessions
	// Only if NOT configuring (to avoid killing active role setup)
	if containerStatus != "configuring" {
		go h.cleanupOrphanedPackageProcesses(dockerID)
	}

	// Start terminal session with auto-restart on exit

	h.runTerminalSessionWithRestart(session, imageType)
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

	// Start periodic status refresh for multi-replica support
	// Checks DB status every 5 seconds and updates frontend if status changed
	statusCtx, statusCancel := context.WithCancel(context.Background())
	defer statusCancel()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		lastStatus := ""
		for {
			select {
			case <-statusCtx.Done():
				return
			case <-session.Done:
				return
			case <-ticker.C:
				// Quick DB lookup to get current status (multi-replica safe)
				if h.store != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
					// Try to get container by Docker ID
					if dbContainer, err := h.store.GetContainerByDockerID(ctx, session.ContainerID); err == nil && dbContainer != nil {
						currentStatus := dbContainer.Status
						// Update manager cache if status changed
						if info, ok := h.containerManager.GetContainer(session.ContainerID); ok && info.Status != currentStatus {
							h.containerManager.UpdateContainerStatus(session.ContainerID, currentStatus)
							log.Printf("[Terminal] Status refresh: updated %s from %s to %s", session.ContainerID[:12], info.Status, currentStatus)
						}
						// Notify frontend if status changed
						if currentStatus != lastStatus && currentStatus != "" {
							session.SendMessage(TerminalMessage{
								Type: "container_status",
								Data: currentStatus,
							})
							lastStatus = currentStatus
						}
					}
					cancel()
				}
			}
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

	// Check if this is a macOS container
	isMacOS := strings.Contains(strings.ToLower(imageType), "macos") || strings.Contains(strings.ToLower(imageType), "osx")

	// For standard Linux containers, attach to tmux session for persistence
	// This allows users to reconnect and see output that happened while disconnected
	var execConfig container.ExecOptions
	if isMacOS {
		// macOS containers don't use tmux (yet)
		// Use /bin/bash directly for macOS - no detection needed
		shell := "/bin/bash"
		execConfig = container.ExecOptions{
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          true,
			Cmd:          []string{shell, "-l"},
			Env: []string{
				"TERM=xterm-256color",
				"COLORTERM=truecolor",
				"LANG=C.UTF-8",
				"LC_ALL=C.UTF-8",
				"PATH=/root/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			},
		}
	} else {
		var shell string
		var hasTmux bool
		var shellCached bool

		// Check if container is configuring FIRST - this enables true fast path
		// by skipping all slow operations (DB lookup, shell detection, tmux check)
		var isConfiguring bool
		if info, ok := h.containerManager.GetContainer(session.ContainerID); ok {
			isConfiguring = info.Status == "configuring"
		}

		// For configuring containers, check if shell setup is already complete
		// If setup is done, use the proper shell (zsh/bash) for better UX
		// Only use /bin/sh if setup is still in progress
		if isConfiguring && h.store != nil {
			// Quick DB check for shell setup status (with very short timeout to avoid delay)
			// session.ContainerID is Docker ID, need to look up DB UUID first
			dbCtx, dbCancel := context.WithTimeout(ctx, 200*time.Millisecond)
			var cachedShell, dbUUID string
			var cachedTmux, setupDone bool
			// Try to get DB UUID and shell metadata by Docker ID
			if dbContainer, err := h.store.GetContainerByDockerID(dbCtx, session.ContainerID); err == nil && dbContainer != nil {
				dbUUID = dbContainer.ID
				// Now get shell metadata using DB UUID (same context, already has timeout)
				cachedShell, cachedTmux, setupDone, _ = h.store.GetContainerShellMetadata(dbCtx, dbUUID)
			}
			dbCancel()

			if setupDone && cachedShell != "" {
				// Shell setup is complete - use the proper shell even though status is "configuring"
				shell = cachedShell
				hasTmux = cachedTmux
				shellCached = true
				log.Printf("[Terminal] Container configuring but shell setup complete, using %s for %s", shell, session.ContainerID[:12])
			} else {
				// Shell setup not complete or lookup timed out - use fast path
				// Use /bin/sh for maximum compatibility - it exists on all Linux distros
				shell = "/bin/sh"
				hasTmux = false
				shellCached = true
				log.Printf("[Terminal] Container configuring, shell setup in progress, using fast /bin/sh path for %s", session.ContainerID[:12])
			}
		} else if h.store != nil {
			// Try to get cached shell metadata from database (only for non-configuring containers)
			// session.ContainerID is Docker ID, need to look up DB UUID first
			dbCtx, dbCancel := context.WithTimeout(ctx, 500*time.Millisecond)
			if dbContainer, err := h.store.GetContainerByDockerID(dbCtx, session.ContainerID); err == nil && dbContainer != nil {
				cachedShell, cachedTmux, setupDone, err := h.store.GetContainerShellMetadata(dbCtx, dbContainer.ID)
				if err == nil && setupDone && cachedShell != "" {
					shell = cachedShell
					hasTmux = cachedTmux
					shellCached = true
					log.Printf("[Terminal] Using cached shell metadata for %s: shell=%s, tmux=%v", session.ContainerID[:12], shell, hasTmux)
				}
			}
			dbCancel()
		}

		// Fall back to /bin/sh for immediate connection if not cached
		// Do detection in background for future connections
		if !shellCached {
			// Use /bin/sh immediately for fastest connection - it exists on all distros
			shell = "/bin/sh"
			hasTmux = false
			log.Printf("[Terminal] Using fast /bin/sh for immediate connection to %s", session.ContainerID[:12])

			// Detect proper shell in background and cache for next connection
			go func(containerID, imgType string) {
				bgCtx, bgCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer bgCancel()

				detectedShell := h.detectShell(bgCtx, containerID, imgType)
				detectedTmux := h.commandExists(bgCtx, containerID, "tmux")

				// Update caches
				h.mu.Lock()
				h.shellCache[containerID] = detectedShell
				h.tmuxCache[containerID] = detectedTmux
				h.mu.Unlock()

				// Cache to DB for multi-replica scenarios
				if h.store != nil && detectedShell != "" {
					dbCtx, dbCancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer dbCancel()
					if dbContainer, err := h.store.GetContainerByDockerID(dbCtx, containerID); err == nil && dbContainer != nil {
						h.store.UpdateContainerShellMetadata(dbCtx, dbContainer.ID, detectedShell, detectedTmux, true)
					}
				}
				log.Printf("[Terminal] Background shell detection complete for %s: shell=%s, tmux=%v", containerID[:12], detectedShell, detectedTmux)
			}(session.ContainerID, imageType)
		}

		// Check for user preference via label
		if info, ok := h.containerManager.GetContainer(session.ContainerID); ok {
			if val, ok := info.Labels["rexec.use_tmux"]; ok {
				if val == "false" {
					hasTmux = false
					log.Printf("[Terminal] tmux disabled by user preference for %s", session.ContainerID[:12])
				} else if val == "true" && !hasTmux {
					// User explicitly enabled tmux but hasTmux is false (fast path or not cached)
					// Quick check if tmux actually exists in the container
					checkCtx, checkCancel := context.WithTimeout(ctx, 500*time.Millisecond)
					if h.commandExists(checkCtx, session.ContainerID, "tmux") {
						hasTmux = true
						log.Printf("[Terminal] tmux enabled by user preference for %s", session.ContainerID[:12])
					} else {
						log.Printf("[Terminal] tmux requested but not available in container %s", session.ContainerID[:12])
					}
					checkCancel()
				}
			}
		}

		// Determine tmux session name
		// - For owner/single user: use "main" (allows reconnecting to same session)
		// - For control-mode collab users: use unique session per user (independent sessions)
		// - For split panes (ForceNewSession): generate unique session name
		tmuxSessionName := "main"
		collabMode := h.getCollabMode(session.ContainerID)

		if session.ForceNewSession {
			// Split pane - create a completely new tmux session with unique name
			tmuxSessionName = fmt.Sprintf("split-%d", time.Now().UnixNano())
			log.Printf("[Terminal] Split pane: creating new tmux session '%s'", tmuxSessionName)
		} else if collabMode == "control" && session.UserID != "" && !session.IsOwner {
			// Each control-mode collab user gets an independent tmux session.
			tmuxSessionName = tmuxSessionNameForControlUser(session.UserID)
			log.Printf("[Terminal] Control mode: using unique tmux session '%s' for user %s", tmuxSessionName, session.UserID)
		}
		session.TmuxSessionName = tmuxSessionName

		if hasTmux {
			// For split panes (ForceNewSession), always create new session (no -A flag)
			// For main sessions, attach if exists or create (-A flag)
			var tmuxCmd []string
			if session.ForceNewSession {
				// Create new session, don't attach to existing
				tmuxCmd = []string{"tmux", "new-session", "-s", tmuxSessionName, shell}
			} else {
				// Attach if session exists, create if not
				tmuxCmd = []string{"tmux", "new-session", "-A", "-s", tmuxSessionName, shell}
			}

			log.Printf("[Terminal] Using tmux session '%s' for %s in %s", tmuxSessionName,
				map[bool]string{true: "split pane", false: "main terminal"}[session.ForceNewSession],
				session.ContainerID[:12])
			execConfig = container.ExecOptions{
				AttachStdin:  true,
				AttachStdout: true,
				AttachStderr: true,
				Tty:          true,
				Cmd:          tmuxCmd,
				Env: []string{
					"TERM=xterm-256color",
					"COLORTERM=truecolor",
					"LANG=C.UTF-8",
					"LC_ALL=C.UTF-8",
					"HOME=/home/user",
					"PATH=/home/user/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				},
				WorkingDir: "/home/user",
			}
		} else {
			// Direct shell without tmux - tmux is optional, not default
			log.Printf("[Terminal] Using direct shell (no tmux) for %s: %s", session.ContainerID[:12], shell)
			execConfig = container.ExecOptions{
				AttachStdin:  true,
				AttachStdout: true,
				AttachStderr: true,
				Tty:          true,
				Cmd:          []string{shell, "-l"},
				Env: []string{
					"TERM=xterm-256color",
					"COLORTERM=truecolor",
					"LANG=C.UTF-8",
					"LC_ALL=C.UTF-8",
					"HOME=/home/user",
					"PATH=/home/user/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				},
				WorkingDir: "/home/user",
			}
		}
	}

	// Send shell_starting message right before creating exec (no delay)
	session.SendMessage(TerminalMessage{
		Type: "shell_starting",
		Data: "Starting shell...",
	})

	execStartTime := time.Now()
	execResp, err := client.ContainerExecCreate(ctx, session.ContainerID, execConfig)
	if err != nil {
		session.SendError("Failed to create terminal session: " + err.Error())
		return false
	}
	execCreateDuration := time.Since(execStartTime)
	// Check container status for logging
	containerStatusForLog := "unknown"
	if info, ok := h.containerManager.GetContainer(session.ContainerID); ok {
		containerStatusForLog = info.Status
	}
	// Log Docker API latency - warn if slow (Docker daemon may be busy)
	if execCreateDuration > 500*time.Millisecond {
		log.Printf("[Terminal] SLOW Docker exec create for %s: %v (status=%s) - Docker daemon may be busy", session.ContainerID[:12], execCreateDuration, containerStatusForLog)
	} else {
		log.Printf("[Terminal] Exec created for %s in %v (status=%s)", session.ContainerID[:12], execCreateDuration, containerStatusForLog)
	}

	session.ExecID = execResp.ID

	// Attach to exec (this also starts it for Podman compatibility)
	// ContainerExecAttach implicitly starts the exec session
	attachStartTime := time.Now()
	attachResp, err := client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{
		Tty: true,
	})
	if err != nil {
		session.SendError("Failed to attach to terminal: " + err.Error())
		return false
	}
	attachDuration := time.Since(attachStartTime)
	// Log Docker API latency - warn if slow
	if attachDuration > 500*time.Millisecond {
		log.Printf("[Terminal] SLOW Docker exec attach for %s: %v - Docker daemon may be busy", session.ContainerID[:12], attachDuration)
	} else {
		log.Printf("[Terminal] Exec attached for %s in %v", session.ContainerID[:12], attachDuration)
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

	// Send shell_ready message after attach succeeds - terminal is now usable
	session.SendMessage(TerminalMessage{
		Type: "shell_ready",
		Data: "Shell ready",
	})
	log.Printf("[Terminal] Shell ready for %s (total setup: %v)", session.ContainerID[:12], time.Since(execStartTime))

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
	// Check cache first
	h.mu.RLock()
	if shell, ok := h.shellCache[containerID]; ok {
		h.mu.RUnlock()
		return shell
	}
	h.mu.RUnlock()

	var shell string

	// First, check if zsh with oh-my-zsh is set up (best experience)
	if h.isZshSetup(ctx, containerID) {
		// Verify zsh exists
		if h.shellExists(ctx, containerID, "/bin/zsh") {
			shell = "/bin/zsh"
		} else if h.shellExists(ctx, containerID, "/usr/bin/zsh") {
			shell = "/usr/bin/zsh"
		}
	}

	if shell == "" {
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
			shell = "/bin/bash"
		} else {
			for _, s := range shells {
				if h.shellExists(ctx, containerID, s) {
					shell = s
					break
				}
			}
		}
	}

	// Last resort - /bin/sh should always exist
	if shell == "" {
		shell = "/bin/sh"
	}

	// Update cache
	h.mu.Lock()
	h.shellCache[containerID] = shell
	h.mu.Unlock()

	return shell
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

// commandExists checks if a command exists in the container's PATH
func (h *TerminalHandler) commandExists(ctx context.Context, containerID, cmd string) bool {
	client := h.containerManager.GetClient()

	execConfig := container.ExecOptions{
		Cmd:          []string{"/bin/sh", "-c", fmt.Sprintf("command -v %s >/dev/null 2>&1", cmd)},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return false
	}

	attachResp, err := client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return false
	}
	attachResp.Close()

	// Poll exec status
	for i := 0; i < 20; i++ {
		inspect, err := client.ContainerExecInspect(ctx, execResp.ID)
		if err != nil {
			return false
		}
		if !inspect.Running {
			return inspect.ExitCode == 0
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func (h *TerminalHandler) killTmuxSession(containerID, tmuxSessionName string) {
	if containerID == "" || tmuxSessionName == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := h.containerManager.GetClient()
	execConfig := container.ExecOptions{
		// Best-effort cleanup; ignore errors if tmux/session doesn't exist.
		Cmd:          []string{"/bin/sh", "-c", fmt.Sprintf("tmux kill-session -t %s 2>/dev/null || true", tmuxSessionName)},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return
	}

	attachResp, err := client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err == nil {
		attachResp.Close()
	}
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
	for _, session := range h.sessions {
		if session != nil && session.ContainerID == containerID && session.UserID == userID {
			return session, true
		}
	}
	return nil, false
}

// CloseSession closes a terminal session
func (h *TerminalHandler) CloseSession(containerID, userID string) {
	var toClose []*TerminalSession
	h.mu.Lock()
	for key, session := range h.sessions {
		if session != nil && session.ContainerID == containerID && session.UserID == userID {
			toClose = append(toClose, session)
			delete(h.sessions, key)
		}
	}
	h.mu.Unlock()

	for _, session := range toClose {
		session.Close()
	}
}

// CloseAllContainerSessions closes all sessions for a container
func (h *TerminalHandler) CloseAllContainerSessions(containerID string) {
	var toClose []*TerminalSession
	h.mu.Lock()
	for key, session := range h.sessions {
		if session != nil && session.ContainerID == containerID {
			toClose = append(toClose, session)
			delete(h.sessions, key)
		}
	}
	h.mu.Unlock()

	for _, session := range toClose {
		session.Close()
	}
}

// CleanupControlCollab closes terminal WebSockets and tmux sessions for control-mode collaborators.
// The container owner is left untouched.
func (h *TerminalHandler) CleanupControlCollab(containerID, ownerID string, participantUserIDs []string) {
	if containerID == "" || ownerID == "" || strings.HasPrefix(containerID, "agent:") {
		return
	}

	participants := make(map[string]struct{}, len(participantUserIDs))
	for _, id := range participantUserIDs {
		if id != "" {
			participants[id] = struct{}{}
		}
	}

	// Close active WebSocket sessions for participants (excluding owner).
	var toClose []*TerminalSession
	h.mu.Lock()
	for key, session := range h.sessions {
		if session == nil || session.ContainerID != containerID {
			continue
		}
		if session.UserID == ownerID {
			continue
		}
		if _, ok := participants[session.UserID]; !ok {
			continue
		}
		toClose = append(toClose, session)
		delete(h.sessions, key)
	}
	h.mu.Unlock()

	for _, session := range toClose {
		session.CloseWithCode(4003, "collaboration ended")
	}

	// Kill per-user tmux sessions created for control-mode collaborators.
	for userID := range participants {
		if userID == ownerID {
			continue
		}
		go h.killTmuxSession(containerID, tmuxSessionNameForControlUser(userID))
	}
}

func tmuxSessionNameForControlUser(userID string) string {
	userHash := userID
	if len(userHash) > 8 {
		userHash = userHash[:8]
	}
	return "user-" + userHash
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

// getCollabMode returns the collab mode for a container ("view", "control", or "" if no collab)
func (h *TerminalHandler) getCollabMode(containerID string) string {
	if h.collabHandler == nil {
		return ""
	}
	h.collabHandler.mu.RLock()
	defer h.collabHandler.mu.RUnlock()

	for _, session := range h.collabHandler.sessions {
		if session.ContainerID == containerID {
			return session.Mode
		}
	}
	return ""
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

	// Semaphore to limit concurrent DB updates
	// Allows up to 5 concurrent DB updates to prevent overwhelming the DB
	sem := make(chan struct{}, 5)

	for range ticker.C {
		h.mu.RLock()

		// 1. Copy sessions to slice to release lock quickly
		type sessionUpdate struct {
			DBSessionID string
			Session     *TerminalSession
		}
		var sessionsToUpdate []sessionUpdate

		for _, session := range h.sessions {
			if session.DBSessionID != "" {
				sessionsToUpdate = append(sessionsToUpdate, sessionUpdate{
					DBSessionID: session.DBSessionID,
					Session:     session,
				})
			}
		}

		// 2. Ping shared sessions (holding lock is fine as it's fast memory op)
		for _, sharedSession := range h.sharedSessions {
			sharedSession.mu.RLock()
			for _, conn := range sharedSession.Connections {
				// Note: technically unsafe if broadcast is writing simultaneously,
				// but low probability collision on ping. Ideally shared conn should be wrapped.
				conn.WriteJSON(TerminalMessage{Type: "ping"})
			}
			sharedSession.mu.RUnlock()
		}
		h.mu.RUnlock()

		// 3. Process session updates without holding the main lock
		for _, item := range sessionsToUpdate {
			// Send ping to client
			item.Session.SendMessage(TerminalMessage{Type: "ping"})

			// Update database timestamp
			select {
			case sem <- struct{}{}: // Acquire token (non-blocking if full)
				go func(dbID string) {
					defer func() { <-sem }() // Release token

					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					if err := h.store.UpdateSessionLastPing(ctx, dbID); err != nil {
						// Log verbose only on error
					}

					// Broadcast session_updated event
					if h.adminEventsHub != nil {
						updatedSession, err := h.store.GetSessionByID(ctx, dbID)
						if err == nil && updatedSession != nil {
							h.adminEventsHub.Broadcast("session_updated", updatedSession)
						}
					}
				}(item.DBSessionID)
			default:
				// Semaphore full, skip DB update for this tick
				// This acts as a natural rate limiter/load shedder
			}
		}
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
