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

	// Copy headers, stripping sensitive/internal ones so they don't leak into user services.
	for key, values := range c.Request.Header {
		canonicalKey := http.CanonicalHeaderKey(key)
		// Skip hop-by-hop headers
		if canonicalKey == "Connection" || canonicalKey == "Keep-Alive" || canonicalKey == "Proxy-Authenticate" ||
			canonicalKey == "Proxy-Authorization" || canonicalKey == "Te" || canonicalKey == "Trailers" ||
			canonicalKey == "Transfer-Encoding" || canonicalKey == "Upgrade" {
			continue
		}
		// Strip auth/cookies and forwarded/internal headers
		if canonicalKey == "Authorization" || canonicalKey == "Cookie" ||
			canonicalKey == "X-Forwarded-For" || canonicalKey == "X-Forwarded-Proto" ||
			canonicalKey == "X-Forwarded-Host" || canonicalKey == "X-Real-Ip" ||
			strings.HasPrefix(canonicalKey, "X-Rexec-") {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(canonicalKey, value)
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
	portStr := "N/A"
	if port > 0 {
		portStr = fmt.Sprintf("%d", port)
	}

	tipsDisplay := ""
	if port > 0 {
		tipsDisplay = fmt.Sprintf(`
        <div class="tips">
            <h3>Troubleshooting Tips</h3>
            <ul>
                <li>Ensure your app is running inside the container</li>
                <li>Check that it's listening on port %d</li>
                <li>Bind to 0.0.0.0, not localhost</li>
                <li>Check container logs for errors</li>
            </ul>
        </div>`, port)
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - Rexec</title>
    <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            background: #0a0a0a;
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            color: #fafafa;
            padding: 20px;
        }
        .logo {
            margin-bottom: 32px;
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .logo svg { width: 32px; height: 32px; }
        .logo span { font-size: 24px; font-weight: 700; color: #fafafa; }
        .content { max-width: 600px; width: 100%%; text-align: center; }
        .terminal-window {
            background: #111111;
            border: 1px solid #262626;
            border-radius: 8px;
            overflow: hidden;
            margin-bottom: 32px;
            text-align: left;
        }
        .terminal-header {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 10px 14px;
            background: #161616;
            border-bottom: 1px solid #262626;
        }
        .terminal-dots { display: flex; gap: 6px; }
        .dot { width: 10px; height: 10px; border-radius: 50%%; }
        .dot.red { background: #ff5f56; }
        .dot.yellow { background: #ffbd2e; }
        .dot.green { background: #27c93f; }
        .terminal-title {
            font-size: 12px;
            color: #71717a;
            font-family: 'JetBrains Mono', 'Fira Code', monospace;
        }
        .terminal-body {
            padding: 16px;
            font-family: 'JetBrains Mono', 'Fira Code', monospace;
            font-size: 13px;
            line-height: 1.6;
        }
        .line { display: flex; gap: 8px; }
        .prompt { color: #3b82f6; }
        .command { color: #fafafa; }
        .error-line { margin: 8px 0; }
        .error-code { color: #ff6b6b; font-weight: 600; }
        .output { color: #a1a1aa; }
        .cursor { color: #3b82f6; animation: blink 1s step-end infinite; }
        @keyframes blink { 50%% { opacity: 0; } }
        .error-info h1 {
            font-size: 72px;
            font-weight: 700;
            color: #3b82f6;
            margin: 0;
            line-height: 1;
            font-family: 'JetBrains Mono', monospace;
        }
        .error-info h2 {
            font-size: 24px;
            font-weight: 600;
            color: #fafafa;
            margin: 8px 0 16px;
        }
        .error-info p {
            color: #a1a1aa;
            font-size: 14px;
            margin: 0 0 24px;
            line-height: 1.6;
        }
        .tips {
            text-align: left;
            background: #111111;
            border: 1px solid #262626;
            border-radius: 8px;
            padding: 16px;
            margin-bottom: 24px;
        }
        .tips h3 {
            font-size: 12px;
            text-transform: uppercase;
            letter-spacing: 1px;
            color: #71717a;
            margin-bottom: 12px;
        }
        .tips ul { list-style: none; font-size: 13px; color: #a1a1aa; }
        .tips li { padding: 6px 0; display: flex; align-items: center; gap: 8px; }
        .tips li::before { content: "→"; color: #3b82f6; }
        .btn {
            display: inline-flex;
            align-items: center;
            gap: 8px;
            padding: 12px 20px;
            font-size: 14px;
            font-weight: 500;
            border-radius: 6px;
            border: 1px solid #262626;
            cursor: pointer;
            text-decoration: none;
            transition: all 0.15s ease;
            background: transparent;
            color: #fafafa;
            font-family: inherit;
        }
        .btn:hover { background: #161616; border-color: #3b82f6; }
        .btn-primary { background: #3b82f6; color: white; border-color: #3b82f6; }
        .btn-primary:hover { background: #2563eb; transform: translateY(-1px); }
        .actions { display: flex; gap: 12px; justify-content: center; flex-wrap: wrap; margin-top: 24px; }
        .suggestions { margin-top: 48px; padding-top: 32px; border-top: 1px solid #262626; }
        .suggestions h3 {
            font-size: 12px;
            text-transform: uppercase;
            letter-spacing: 1px;
            color: #71717a;
            margin: 0 0 16px;
        }
        .suggestion-links { display: flex; gap: 12px; justify-content: center; flex-wrap: wrap; }
        .suggestion-links a {
            color: #3b82f6;
            text-decoration: none;
            font-size: 14px;
            padding: 6px 12px;
            border: 1px solid #262626;
            border-radius: 6px;
            transition: all 0.15s ease;
        }
        .suggestion-links a:hover { background: #161616; border-color: #3b82f6; }
        @media (max-width: 480px) {
            .error-info h1 { font-size: 56px; }
            .terminal-body { font-size: 11px; }
            .actions { flex-direction: column; }
            .btn { width: 100%%; justify-content: center; }
        }
    </style>
</head>
<body>
    <div class="logo">
        <svg viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
            <rect width="32" height="32" rx="8" fill="#3b82f6"/>
            <path d="M8 10L14 16L8 22" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M16 22H24" stroke="white" stroke-width="2.5" stroke-linecap="round"/>
        </svg>
        <span>rexec</span>
    </div>
    <div class="content">
        <div class="terminal-window">
            <div class="terminal-header">
                <div class="terminal-dots">
                    <span class="dot red"></span>
                    <span class="dot yellow"></span>
                    <span class="dot green"></span>
                </div>
                <span class="terminal-title">rexec@port-forward</span>
            </div>
            <div class="terminal-body">
                <div class="line">
                    <span class="prompt">$</span>
                    <span class="command">curl localhost:%s</span>
                </div>
                <div class="line error-line">
                    <span class="error-code">HTTP/1.1 503 Service Unavailable</span>
                </div>
                <div class="line"><span class="output">{</span></div>
                <div class="line"><span class="output">&nbsp;&nbsp;"error": "%s",</span></div>
                <div class="line"><span class="output">&nbsp;&nbsp;"message": "%s",</span></div>
                <div class="line"><span class="output">&nbsp;&nbsp;"port": "%s"</span></div>
                <div class="line"><span class="output">}</span></div>
                <div class="line">
                    <span class="prompt">$</span>
                    <span class="cursor">_</span>
                </div>
            </div>
        </div>
        
        <div class="error-info">
            <h1>503</h1>
            <h2>%s</h2>
            <p>%s</p>
        </div>
        
        %s
        
        <div class="actions">
            <button onclick="location.reload()" class="btn btn-primary">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M23 4v6h-6M1 20v-6h6"/>
                    <path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/>
                </svg>
                Retry
            </button>
            <a href="/" class="btn">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
                    <polyline points="9 22 9 12 15 12 15 22"/>
                </svg>
                Go to Dashboard
            </a>
        </div>
        
        <div class="suggestions">
            <h3>Quick Links</h3>
            <div class="suggestion-links">
                <a href="/">Dashboard</a>
                <a href="/use-cases">Use Cases</a>
                <a href="/guides">Guides</a>
                <a href="/pricing">Pricing</a>
            </div>
        </div>
    </div>
</body>
</html>`, title, portStr, title, message, portStr, title, message, tipsDisplay)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("X-Rexec-Error", "true") // Signal this is our error page
	c.String(http.StatusOK, html)     // Use 200 to bypass platform error page replacement
}
