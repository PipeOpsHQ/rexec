package handlers

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rexec/rexec/internal/container"
	"github.com/rexec/rexec/internal/models"
	"github.com/rexec/rexec/internal/storage"
)

// PortForwardHandler handles port forwarding API and WebSocket endpoints
type PortForwardHandler struct {
	store            *storage.PostgresStore
	containerManager *container.Manager
	activeForwards   map[string]*ActivePortForward // map[forwardID]ActivePortForward
	mu               sync.Mutex
	upgrader         websocket.Upgrader
}

// ActivePortForward holds state for an active port forward session
type ActivePortForward struct {
	ForwardID   string
	UserID      string
	ContainerID string
	ContainerPort int
	LocalPort   int
	Cancel      context.CancelFunc
}

// NewPortForwardHandler creates a new PortForwardHandler
func NewPortForwardHandler(store *storage.PostgresStore, containerManager *container.Manager) *PortForwardHandler {
	return &PortForwardHandler{
		store:            store,
		containerManager: containerManager,
		activeForwards:   make(map[string]*ActivePortForward),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for now, refine in production
				return true
			},
		},
	}
}

// CreatePortForwardRequest represents the request to create a port forward
type CreatePortForwardRequest struct {
	Name          string `json:"name"`                     // Optional name
	ContainerID   string `json:"container_id" binding:"required"`
	ContainerPort int    `json:"container_port" binding:"required,gt=0,lte=65535"`
	LocalPort     int    `json:"local_port" binding:"required,gt=0,lte=65535"`
}

// PortForwardResponse represents a port forward in API responses
type PortForwardResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	ContainerID   string    `json:"container_id"`
	ContainerPort int       `json:"container_port"`
	LocalPort     int       `json:"local_port"`
	Protocol      string    `json:"protocol"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	WebSocketURL  string    `json:"websocket_url"` // URL for the client to connect to
}

// CreatePortForward creates a new port forward
// POST /api/containers/:id/port-forwards
func (h *PortForwardHandler) CreatePortForward(c *gin.Context) {
	userID := c.GetString("userID")
	containerID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.APIError{Code: http.StatusUnauthorized, Message: "unauthorized"})
		return
	}

	var req CreatePortForwardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Code: http.StatusBadRequest, Message: "invalid request: " + err.Error()})
		return
	}

	if req.ContainerID != containerID {
		c.JSON(http.StatusBadRequest, models.APIError{Code: http.StatusBadRequest, Message: "container ID mismatch"})
		return
	}

	// Verify container ownership and status
	containerRecord, err := h.store.GetContainerByID(c.Request.Context(), containerID)
	if err != nil || containerRecord == nil {
		c.JSON(http.StatusNotFound, models.APIError{Code: http.StatusNotFound, Message: "container not found"})
		return
	}
	if containerRecord.UserID != userID {
		c.JSON(http.StatusForbidden, models.APIError{Code: http.StatusForbidden, Message: "access denied"})
		return
	}
	if containerRecord.Status != string(models.StatusRunning) {
		c.JSON(http.StatusBadRequest, models.APIError{Code: http.StatusBadRequest, Message: "container is not running"})
		return
	}

	// Check for port conflicts on the local client side (this assumes a single client)
	// In a real multi-client scenario, this would be more complex (e.g. dynamic client ports)
	h.mu.Lock()
	for _, af := range h.activeForwards {
		if af.UserID == userID && af.LocalPort == req.LocalPort {
			h.mu.Unlock()
			c.JSON(http.StatusConflict, models.APIError{Code: http.StatusConflict, Message: fmt.Sprintf("Local port %d is already in use by forward %s", req.LocalPort, af.ForwardID)})
			return
		}
	}
	h.mu.Unlock()

	// Create port forward record in DB
	pf := &models.PortForward{
		ID:          uuid.New().String(),
		UserID:      userID,
		ContainerID: containerID,
		Name:        req.Name,
		ContainerPort: req.ContainerPort,
		LocalPort:   req.LocalPort,
		Protocol:    "tcp", // Only TCP for now
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	if err := h.store.CreatePortForward(c.Request.Context(), pf); err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "unique_port_forward") || strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, models.APIError{Code: http.StatusConflict, Message: "A similar port forward already exists for this container/ports"})
			return
		}
		c.JSON(http.StatusInternalServerError, models.APIError{Code: http.StatusInternalServerError, Message: "failed to save port forward: " + err.Error()})
		return
	}

	// Create WebSocket URL for client to connect to
	wsURL := fmt.Sprintf("%s/ws/port-forward/%s", getWebSocketBaseURL(c.Request), pf.ID)

	response := PortForwardResponse{
		ID:            pf.ID,
		Name:          pf.Name,
		ContainerID:   pf.ContainerID,
		ContainerPort: pf.ContainerPort,
		LocalPort:     pf.LocalPort,
		Protocol:      pf.Protocol,
		IsActive:      pf.IsActive,
		CreatedAt:     pf.CreatedAt,
		WebSocketURL:  wsURL,
	}

	c.JSON(http.StatusCreated, response)
}

// ListPortForwards lists active port forwards for a container
// GET /api/containers/:id/port-forwards
func (h *PortForwardHandler) ListPortForwards(c *gin.Context) {
	userID := c.GetString("userID")
	containerID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.APIError{Code: http.StatusUnauthorized, Message: "unauthorized"})
		return
	}

	// Verify container ownership
	containerRecord, err := h.store.GetContainerByID(c.Request.Context(), containerID)
	if err != nil || containerRecord == nil {
		c.JSON(http.StatusNotFound, models.APIError{Code: http.StatusNotFound, Message: "container not found"})
		return
	}
	if containerRecord.UserID != userID {
		c.JSON(http.StatusForbidden, models.APIError{Code: http.StatusForbidden, Message: "access denied"})
		return
	}

	forwards, err := h.store.GetPortForwardsByUserIDAndContainerID(c.Request.Context(), userID, containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Code: http.StatusInternalServerError, Message: "failed to fetch port forwards"})
		return
	}

	response := make([]PortForwardResponse, 0, len(forwards))
	for _, pf := range forwards {
		wsURL := fmt.Sprintf("%s/ws/port-forward/%s", getWebSocketBaseURL(c.Request), pf.ID)
		response = append(response, PortForwardResponse{
			ID:            pf.ID,
			Name:          pf.Name,
			ContainerID:   pf.ContainerID,
			ContainerPort: pf.ContainerPort,
			LocalPort:     pf.LocalPort,
			Protocol:      pf.Protocol,
			IsActive:      pf.IsActive,
			CreatedAt:     pf.CreatedAt,
			WebSocketURL:  wsURL,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"forwards": response,
		"count":    len(response),
	})
}

// DeletePortForward deletes an active port forward
// DELETE /api/containers/:id/port-forwards/:forwardId
func (h *PortForwardHandler) DeletePortForward(c *gin.Context) {
	userID := c.GetString("userID")
	containerID := c.Param("id")
	forwardID := c.Param("forwardId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.APIError{Code: http.StatusUnauthorized, Message: "unauthorized"})
		return
	}

	// Verify ownership
	pf, err := h.store.GetPortForwardByID(c.Request.Context(), forwardID)
	if err != nil || pf == nil {
		c.JSON(http.StatusNotFound, models.APIError{Code: http.StatusNotFound, Message: "port forward not found"})
		return
	}
	// Ensure the forward belongs to the authenticated user AND the correct container
	if pf.UserID != userID || pf.ContainerID != containerID {
		c.JSON(http.StatusForbidden, models.APIError{Code: http.StatusForbidden, Message: "access denied"})
		return
	}

	// Mark as inactive in DB
	if err := h.store.DeletePortForward(c.Request.Context(), forwardID); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Code: http.StatusInternalServerError, Message: "failed to delete port forward: " + err.Error()})
		return
	}

	// Remove from active forwards map and cancel context
	h.mu.Lock()
	if activeFwd, ok := h.activeForwards[forwardID]; ok {
		activeFwd.Cancel() // Signal the WebSocket handler to close
		delete(h.activeForwards, forwardID)
	}
	h.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"message": "Port forward deleted"})
}

// HandlePortForwardWebSocket establishes a WebSocket connection for port tunneling
// GET /ws/port-forward/:forwardId
func (h *PortForwardHandler) HandlePortForwardWebSocket(c *gin.Context) {
	userID := c.GetString("userID")
	forwardID := c.Param("forwardId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.APIError{Code: http.StatusUnauthorized, Message: "unauthorized"})
		return
	}

	// Retrieve port forward from DB
	pf, err := h.store.GetPortForwardByID(c.Request.Context(), forwardID)
	if err != nil || pf == nil || !pf.IsActive {
		c.JSON(http.StatusNotFound, models.APIError{Code: http.StatusNotFound, Message: "active port forward not found"})
		return
	}

	// Verify ownership
	if pf.UserID != userID {
		c.JSON(http.StatusForbidden, models.APIError{Code: http.StatusForbidden, Message: "access denied"})
		return
	}

	// Verify container status
	containerRecord, err := h.store.GetContainerByID(c.Request.Context(), pf.ContainerID)
	if err != nil || containerRecord == nil || containerRecord.Status != string(models.StatusRunning) {
		c.JSON(http.StatusBadRequest, models.APIError{Code: http.StatusBadRequest, Message: "target container not running"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	wsConn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket for port forward %s: %v", forwardID, err)
		return
	}
	defer wsConn.Close()

	// Establish TCP connection to the container's internal port
	// We need the Docker host or container IP
	dockerClient := h.containerManager.GetClient()
	inspect, err := dockerClient.ContainerInspect(c.Request.Context(), pf.ContainerID)
	if err != nil {
		log.Printf("Failed to inspect container %s for port forward %s: %v", pf.ContainerID, forwardID, err)
		return
	}

	ipAddress := inspect.NetworkSettings.IPAddress
	if ipAddress == "" {
		// Fallback for Podman/Docker networks - try finding the first attached network's IP
		for _, network := range inspect.NetworkSettings.Networks {
			if network.IPAddress != "" {
				ipAddress = network.IPAddress
				break
			}
		}
	}

	if ipAddress == "" {
		log.Printf("Container %s has no IP address for port forward %s", pf.ContainerID, forwardID)
		wsConn.WriteMessage(websocket.TextMessage, []byte("Error: Container has no IP address"))
		return
	}

	containerAddr := fmt.Sprintf("%s:%d", ipAddress, pf.ContainerPort)
	tcpConn, err := net.DialTimeout("tcp", containerAddr, 5*time.Second)
	if err != nil {
		log.Printf("Failed to connect to container %s port %d for forward %s: %v", pf.ContainerID, pf.ContainerPort, forwardID, err)
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: Could not connect to container port %d: %v", pf.ContainerPort, err)))
		return
	}
	defer tcpConn.Close()

	log.Printf("Established port forward %s: WS client -> TCP %s", forwardID, containerAddr)

	// Add to active forwards map
	ctx, cancel := context.WithCancel(context.Background())
	h.mu.Lock()
	h.activeForwards[forwardID] = &ActivePortForward{
		ForwardID:   forwardID,
		UserID:      userID,
		ContainerID: pf.ContainerID,
		ContainerPort: pf.ContainerPort,
		LocalPort:   pf.LocalPort,
		Cancel:      cancel,
	}
	h.mu.Unlock()

	defer func() {
		// Clean up on exit
		log.Printf("Closing port forward %s", forwardID)
		h.mu.Lock()
		delete(h.activeForwards, forwardID)
		h.mu.Unlock()
		// Also mark as inactive in DB
		h.store.DeletePortForward(context.Background(), forwardID) // Soft delete
	}()

	// Bidirectional proxying
	var wg sync.WaitGroup
	wg.Add(2)

	// WebSocket to TCP
	go func() {
		defer wg.Done()
		defer tcpConn.Close()
		defer wsConn.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				mt, message, err := wsConn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Printf("WebSocket read error for port forward %s: %v", forwardID, err)
					}
					return
				}
				if mt == websocket.BinaryMessage {
					if _, err := tcpConn.Write(message); err != nil {
						log.Printf("TCP write error for port forward %s: %v", forwardID, err)
						return
					}
				} else if mt == websocket.TextMessage {
					// Log text messages but typically forward binary
					log.Printf("WS text message for port forward %s: %s", forwardID, message)
				}
			}
		}
	}()

	// TCP to WebSocket
	go func() {
		defer wg.Done()
		defer tcpConn.Close()
		defer wsConn.Close()
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := tcpConn.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("TCP read error for port forward %s: %v", forwardID, err)
					}
					return
				}
				if n > 0 {
					if err := wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
						log.Printf("WebSocket write error for port forward %s: %v", forwardID, err)
						return
					}
				}
			}
		}
	}()

	wg.Wait() // Wait for both goroutines to finish
}

// Helper to get WebSocket base URL
func getWebSocketBaseURL(r *http.Request) string {
	scheme := "ws"
	if r.TLS != nil || strings.HasPrefix(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "wss"
	}
	// Check for custom WebSocket host if needed, otherwise use request host
	wsHost := os.Getenv("REXEC_WS_HOST")
	if wsHost != "" {
		return fmt.Sprintf("%s://%s", scheme, wsHost)
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
