package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rexec/rexec/internal/container"
	"github.com/rexec/rexec/internal/models"
	"github.com/rexec/rexec/internal/storage"
)

// ContainerEvent represents a container state change event
type ContainerEvent struct {
	Type      string      `json:"type"`      // "created", "started", "stopped", "deleted", "updated"
	Container interface{} `json:"container"` // Container data
	Timestamp time.Time   `json:"timestamp"`
}

// ContainerEventsHub manages WebSocket connections for container events
type ContainerEventsHub struct {
	manager *container.Manager
	store   *storage.PostgresStore

	// Map of userID -> set of connections
	connections map[string]map[*websocket.Conn]bool
	mu          sync.RWMutex

	// Upgrader for WebSocket
	upgrader websocket.Upgrader
}

// NewContainerEventsHub creates a new container events hub
func NewContainerEventsHub(manager *container.Manager, store *storage.PostgresStore) *ContainerEventsHub {
	hub := &ContainerEventsHub{
		manager:     manager,
		store:       store,
		connections: make(map[string]map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	return hub
}

// HandleWebSocket handles WebSocket connections for container events
func (h *ContainerEventsHub) HandleWebSocket(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tier := c.GetString("tier")

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Register connection
	h.registerConnection(userID, conn)

	// Send initial container list
	h.sendContainerList(conn, userID, tier)

	// Handle connection lifecycle
	go h.handleConnection(conn, userID, tier)
}

// registerConnection adds a connection to the hub
func (h *ContainerEventsHub) registerConnection(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.connections[userID] == nil {
		h.connections[userID] = make(map[*websocket.Conn]bool)
	}
	h.connections[userID][conn] = true

	log.Printf("[ContainerEvents] User %s connected (total connections: %d)", userID, len(h.connections[userID]))
}

// unregisterConnection removes a connection from the hub
func (h *ContainerEventsHub) unregisterConnection(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.connections[userID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.connections, userID)
		}
	}

	conn.Close()
	log.Printf("[ContainerEvents] User %s disconnected", userID)
}

// handleConnection manages a WebSocket connection
func (h *ContainerEventsHub) handleConnection(conn *websocket.Conn, userID, tier string) {
	defer h.unregisterConnection(userID, conn)

	// Set up ping/pong for connection health
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Ping ticker
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Read messages (mainly for pong responses and keepalive)
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[ContainerEvents] Read error: %v", err)
				}
				return
			}
		}
	}()

	// Send pings periodically
	for {
		select {
		case <-pingTicker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sendContainerList sends the full container list to a connection
func (h *ContainerEventsHub) sendContainerList(conn *websocket.Conn, userID, tier string) {
	ctx := context.Background()
	records, err := h.store.GetContainersByUserID(ctx, userID)
	if err != nil {
		log.Printf("[ContainerEvents] Failed to get containers: %v", err)
		return
	}

	containers := make([]gin.H, 0, len(records))

	for _, record := range records {
		status := record.Status
		var idleTime float64

		if info, ok := h.manager.GetContainer(record.DockerID); ok {
			status = info.Status
			idleTime = time.Since(info.LastUsedAt).Seconds()
		} else if status == "running" && !record.LastUsedAt.IsZero() {
			// Calculate idle time from database for running containers not in memory
			idleTime = time.Since(record.LastUsedAt).Seconds()
		}

		// Use stored resources from database (with fallback to tier limits for old containers)
		memoryMB := record.MemoryMB
		cpuShares := record.CPUShares
		diskMB := record.DiskMB
		if memoryMB == 0 {
			limits := models.TierLimits(tier)
			memoryMB = limits.MemoryMB
			cpuShares = limits.CPUShares
			diskMB = limits.DiskMB
		}

		containers = append(containers, gin.H{
			"id":           record.DockerID,
			"db_id":        record.ID,
			"user_id":      record.UserID,
			"name":         record.Name,
			"image":        record.Image,
			"status":       status,
			"created_at":   record.CreatedAt,
			"last_used_at": record.LastUsedAt,
			"idle_seconds": idleTime,
			"resources": gin.H{
				"memory_mb":  memoryMB,
				"cpu_shares": cpuShares,
				"disk_mb":    diskMB,
			},
		})
	}

	event := ContainerEvent{
		Type: "list",
		Container: gin.H{
			"containers": containers,
			"count":      len(containers),
			"limit":      container.UserContainerLimit(tier),
		},
		Timestamp: time.Now(),
	}

	h.sendToConnection(conn, event)
}

// sendToConnection sends an event to a specific connection
func (h *ContainerEventsHub) sendToConnection(conn *websocket.Conn, event ContainerEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[ContainerEvents] Failed to marshal event: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("[ContainerEvents] Failed to send event: %v", err)
	}
}

// BroadcastToUser sends an event to all connections for a user
func (h *ContainerEventsHub) BroadcastToUser(userID string, event ContainerEvent) {
	h.mu.RLock()
	conns, ok := h.connections[userID]
	if !ok {
		h.mu.RUnlock()
		log.Printf("[ContainerEvents] No connections for user %s, skipping broadcast of %s event", userID, event.Type)
		return
	}

	// Copy connections to avoid holding lock during send
	connList := make([]*websocket.Conn, 0, len(conns))
	for conn := range conns {
		connList = append(connList, conn)
	}
	h.mu.RUnlock()

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[ContainerEvents] Failed to marshal event: %v", err)
		return
	}

	log.Printf("[ContainerEvents] Broadcasting %s event to %d connections for user %s", event.Type, len(connList), userID)

	for _, conn := range connList {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("[ContainerEvents] Failed to broadcast to user %s: %v", userID, err)
		}
	}
}

// NotifyContainerCreated notifies a user that a container was created
func (h *ContainerEventsHub) NotifyContainerCreated(userID string, containerData interface{}) {
	h.BroadcastToUser(userID, ContainerEvent{
		Type:      "created",
		Container: containerData,
		Timestamp: time.Now(),
	})
}

// NotifyContainerUpdated notifies a user that a container was updated
func (h *ContainerEventsHub) NotifyContainerUpdated(userID string, containerData interface{}) {
	h.BroadcastToUser(userID, ContainerEvent{
		Type:      "updated",
		Container: containerData,
		Timestamp: time.Now(),
	})
}

// NotifyContainerDeleted notifies a user that a container was deleted
func (h *ContainerEventsHub) NotifyContainerDeleted(userID string, containerID string) {
	h.BroadcastToUser(userID, ContainerEvent{
		Type:      "deleted",
		Container: gin.H{"id": containerID},
		Timestamp: time.Now(),
	})
}

// NotifyContainerStarted notifies a user that a container started
func (h *ContainerEventsHub) NotifyContainerStarted(userID string, containerData interface{}) {
	h.BroadcastToUser(userID, ContainerEvent{
		Type:      "started",
		Container: containerData,
		Timestamp: time.Now(),
	})
}

// NotifyContainerStopped notifies a user that a container stopped
func (h *ContainerEventsHub) NotifyContainerStopped(userID string, containerData interface{}) {
	h.BroadcastToUser(userID, ContainerEvent{
		Type:      "stopped",
		Container: containerData,
		Timestamp: time.Now(),
	})
}
