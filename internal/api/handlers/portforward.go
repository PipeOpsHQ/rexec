package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
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
	WebSocketURL  string    `json:"websocket_url"` // URL for WebSocket tunneling (legacy)
	ProxyURL      string    `json:"proxy_url"`     // URL for HTTP proxy access
}

// resolveContainer looks up a container by DB ID or Docker ID
func (h *PortForwardHandler) resolveContainer(ctx context.Context, idOrDockerID string) (*storage.ContainerRecord, error) {
	// Try DB ID first
	record, err := h.store.GetContainerByID(ctx, idOrDockerID)
	if err == nil && record != nil {
		return record, nil
	}
	// Try Docker ID as fallback
	record, err = h.store.GetContainerByDockerID(ctx, idOrDockerID)
	if err == nil && record != nil {
		return record, nil
	}
	return nil, fmt.Errorf("container not found")
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

	// Verify container ownership and status (supports both DB ID and Docker ID)
	containerRecord, err := h.resolveContainer(c.Request.Context(), containerID)
	if err != nil || containerRecord == nil {
		c.JSON(http.StatusNotFound, models.APIError{Code: http.StatusNotFound, Message: "container not found"})
		return
	}
	
	// Update containerID to use DB ID for consistency
	containerID = containerRecord.ID
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
	// Create HTTP proxy URL for direct browser access
	proxyURL := fmt.Sprintf("%s/p/%s/", getHTTPBaseURL(c.Request), pf.ID)

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
		ProxyURL:      proxyURL,
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

	// Verify container ownership (supports both DB ID and Docker ID)
	containerRecord, err := h.resolveContainer(c.Request.Context(), containerID)
	if err != nil || containerRecord == nil {
		c.JSON(http.StatusNotFound, models.APIError{Code: http.StatusNotFound, Message: "container not found"})
		return
	}
	if containerRecord.UserID != userID {
		c.JSON(http.StatusForbidden, models.APIError{Code: http.StatusForbidden, Message: "access denied"})
		return
	}

	// Use DB ID for the query
	forwards, err := h.store.GetPortForwardsByUserIDAndContainerID(c.Request.Context(), userID, containerRecord.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Code: http.StatusInternalServerError, Message: "failed to fetch port forwards"})
		return
	}

	response := make([]PortForwardResponse, 0, len(forwards))
	for _, pf := range forwards {
		wsURL := fmt.Sprintf("%s/ws/port-forward/%s", getWebSocketBaseURL(c.Request), pf.ID)
		proxyURL := fmt.Sprintf("%s/p/%s/", getHTTPBaseURL(c.Request), pf.ID)
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
			ProxyURL:      proxyURL,
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

	// Resolve container (supports both DB ID and Docker ID)
	containerRecord, err := h.resolveContainer(c.Request.Context(), containerID)
	if err != nil || containerRecord == nil {
		c.JSON(http.StatusNotFound, models.APIError{Code: http.StatusNotFound, Message: "container not found"})
		return
	}

	// Verify ownership
	pf, err := h.store.GetPortForwardByID(c.Request.Context(), forwardID)
	if err != nil || pf == nil {
		c.JSON(http.StatusNotFound, models.APIError{Code: http.StatusNotFound, Message: "port forward not found"})
		return
	}
	// Ensure the forward belongs to the authenticated user AND the correct container (use DB ID)
	if pf.UserID != userID || pf.ContainerID != containerRecord.ID {
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
	dockerID := containerRecord.DockerID
	if dockerID == "" {
		log.Printf("Container %s has no Docker ID for port forward %s", pf.ContainerID, forwardID)
		wsConn.WriteMessage(websocket.TextMessage, []byte("Error: Container not available"))
		return
	}
	inspect, err := dockerClient.ContainerInspect(c.Request.Context(), dockerID)
	if err != nil {
		log.Printf("Failed to inspect container %s (docker: %s) for port forward %s: %v", pf.ContainerID, dockerID, forwardID, err)
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

// Helper to get HTTP base URL for port forward access URLs
func getHTTPBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.HasPrefix(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

// HandleHTTPProxy handles HTTP requests to proxied container ports
// GET/POST/etc /p/:forwardId/*path
func (h *PortForwardHandler) HandleHTTPProxy(c *gin.Context) {
	forwardID := c.Param("forwardId")
	proxyPath := c.Param("path")
	if proxyPath == "" {
		proxyPath = "/"
	}

	// Retrieve port forward from DB (no auth required for public access)
	pf, err := h.store.GetPortForwardByID(c.Request.Context(), forwardID)
	if err != nil || pf == nil || !pf.IsActive {
		h.renderPortForwardError(c, "Port Forward Not Found", "This port forward link is invalid or has been deactivated.", 0)
		return
	}

	// Verify container status
	containerRecord, err := h.store.GetContainerByID(c.Request.Context(), pf.ContainerID)
	if err != nil || containerRecord == nil || containerRecord.Status != string(models.StatusRunning) {
		h.renderPortForwardError(c, "Container Not Running", "The container associated with this port forward is not currently running. Please start the container and try again.", pf.ContainerPort)
		return
	}

	// Get container IP - use Docker ID, not DB ID
	dockerClient := h.containerManager.GetClient()
	dockerID := containerRecord.DockerID
	if dockerID == "" {
		log.Printf("Container %s has no Docker ID for proxy %s", pf.ContainerID, forwardID)
		h.renderPortForwardError(c, "Container Unavailable", "The container is not properly initialized. Please try restarting it.", pf.ContainerPort)
		return
	}
	inspect, err := dockerClient.ContainerInspect(c.Request.Context(), dockerID)
	if err != nil {
		log.Printf("Failed to inspect container %s (docker: %s) for proxy %s: %v", pf.ContainerID, dockerID, forwardID, err)
		h.renderPortForwardError(c, "Connection Error", "Failed to connect to the container. Please try again later.", pf.ContainerPort)
		return
	}

	ipAddress := inspect.NetworkSettings.IPAddress
	if ipAddress == "" {
		for _, network := range inspect.NetworkSettings.Networks {
			if network.IPAddress != "" {
				ipAddress = network.IPAddress
				break
			}
		}
	}

	if ipAddress == "" {
		h.renderPortForwardError(c, "Network Error", "The container has no network address. Please try restarting it.", pf.ContainerPort)
		return
	}

	// Build target URL
	targetURL := fmt.Sprintf("http://%s:%d%s", ipAddress, pf.ContainerPort, proxyPath)
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	// Create proxy request
	proxyReq, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create proxy request"})
		return
	}

	// Copy headers
	for key, values := range c.Request.Header {
		// Skip hop-by-hop headers
		if key == "Connection" || key == "Keep-Alive" || key == "Proxy-Authenticate" ||
			key == "Proxy-Authorization" || key == "Te" || key == "Trailers" ||
			key == "Transfer-Encoding" || key == "Upgrade" {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Add forwarding headers
	proxyReq.Header.Set("X-Forwarded-For", c.ClientIP())
	proxyReq.Header.Set("X-Forwarded-Proto", "https")
	proxyReq.Header.Set("X-Forwarded-Host", c.Request.Host)
	proxyReq.Header.Set("X-Real-IP", c.ClientIP())

	// Execute request
	client := &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects, let client handle them
		},
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Printf("Proxy request failed for %s: %v", forwardID, err)
		h.renderPortForwardError(c, "Service Unavailable", "The service at this port is not responding. Make sure your application is running and listening on the correct port.", pf.ContainerPort)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// Set status code
	c.Status(resp.StatusCode)

	// Stream response body
	io.Copy(c.Writer, resp.Body)
}

// renderPortForwardError renders a branded HTML error page for port forwarding
func (h *PortForwardHandler) renderPortForwardError(c *gin.Context, title, message string, port int) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - Rexec</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #0a0a0f 0%%, #1a1a2e 50%%, #16213e 100%%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #fff;
        }
        .container {
            text-align: center;
            padding: 2rem;
            max-width: 500px;
        }
        .logo {
            width: 80px;
            height: 80px;
            margin: 0 auto 1.5rem;
            background: linear-gradient(135deg, #6366f1 0%%, #8b5cf6 100%%);
            border-radius: 16px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 2rem;
            font-weight: bold;
        }
        h1 {
            font-size: 1.75rem;
            margin-bottom: 1rem;
            background: linear-gradient(135deg, #6366f1, #8b5cf6);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }
        p {
            color: #94a3b8;
            line-height: 1.6;
            margin-bottom: 1.5rem;
        }
        .port-info {
            background: rgba(99, 102, 241, 0.1);
            border: 1px solid rgba(99, 102, 241, 0.3);
            border-radius: 8px;
            padding: 1rem;
            margin-bottom: 1.5rem;
        }
        .port-info code {
            background: rgba(99, 102, 241, 0.2);
            padding: 0.25rem 0.5rem;
            border-radius: 4px;
            font-family: 'JetBrains Mono', monospace;
            color: #a5b4fc;
        }
        .tips {
            text-align: left;
            background: rgba(255, 255, 255, 0.05);
            border-radius: 8px;
            padding: 1rem;
        }
        .tips h3 {
            font-size: 0.875rem;
            color: #6366f1;
            margin-bottom: 0.5rem;
        }
        .tips ul {
            list-style: none;
            font-size: 0.875rem;
            color: #94a3b8;
        }
        .tips li {
            padding: 0.25rem 0;
        }
        .tips li::before {
            content: "→";
            color: #6366f1;
            margin-right: 0.5rem;
        }
        a {
            color: #6366f1;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">R×</div>
        <h1>%s</h1>
        <p>%s</p>
        <div class="port-info">
            Target Port: <code>%d</code>
        </div>
        <div class="tips">
            <h3>Troubleshooting Tips</h3>
            <ul>
                <li>Ensure your app is running inside the container</li>
                <li>Check that it's listening on port %d</li>
                <li>Bind to 0.0.0.0, not localhost</li>
                <li>Check container logs for errors</li>
            </ul>
        </div>
        <p style="margin-top: 1.5rem; font-size: 0.875rem;">
            <a href="/">← Back to Rexec</a>
        </p>
    </div>
</body>
</html>`, title, title, message, port, port)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("X-Rexec-Error", "true") // Signal this is our error page
	c.String(http.StatusOK, html) // Use 200 to bypass platform error page replacement
}
