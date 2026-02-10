package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rexec/rexec/internal/container"
	"github.com/rexec/rexec/internal/models"
	"github.com/rexec/rexec/internal/pubsub"
	"github.com/rexec/rexec/internal/storage"
)

// ContainerEvent represents a container state change event
type ContainerEvent struct {
	Type      string      `json:"type"`      // "created", "started", "stopped", "deleted", "updated"
	Container interface{} `json:"container"` // Container data
	Timestamp time.Time   `json:"timestamp"`
}

// SafeConn wraps websocket.Conn with a mutex for thread-safe writes
type SafeConn struct {
	Conn *websocket.Conn
	Mu   sync.Mutex
}

// WriteMessage sends a message thread-safely
func (sc *SafeConn) WriteMessage(messageType int, data []byte) error {
	sc.Mu.Lock()
	defer sc.Mu.Unlock()
	sc.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return sc.Conn.WriteMessage(messageType, data)
}

// ContainerEventsHub manages WebSocket connections for container events
type ContainerEventsHub struct {
	manager *container.Manager
	store   *storage.PostgresStore

	// Map of userID -> set of connections
	connections map[string]map[*websocket.Conn]*SafeConn
	mu          sync.RWMutex

	// Upgrader for WebSocket
	upgrader websocket.Upgrader

	// Redis Pub/Sub for horizontal scaling
	pubsubHub *pubsub.Hub

	// Agent handler for getting online agents (set after creation to avoid circular deps)
	agentHandler interface {
		GetOnlineAgentsForUser(userID string) []gin.H
	}

	// Container handler for getting shared terminals (set after creation to avoid circular deps)
	containerHandler interface {
		GetSharedTerminalsForUser(ctx context.Context, userID string) []gin.H
	}
}

// NewContainerEventsHub creates a new container events hub
func NewContainerEventsHub(manager *container.Manager, store *storage.PostgresStore) *ContainerEventsHub {
	hub := &ContainerEventsHub{
		manager:     manager,
		store:       store,
		connections: make(map[string]map[*websocket.Conn]*SafeConn),
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

// SetAgentHandler sets the agent handler for getting online agents
func (h *ContainerEventsHub) SetAgentHandler(ah interface {
	GetOnlineAgentsForUser(userID string) []gin.H
}) {
	h.agentHandler = ah
}

// SetContainerHandler sets the container handler for getting shared terminals
func (h *ContainerEventsHub) SetContainerHandler(ch interface {
	GetSharedTerminalsForUser(ctx context.Context, userID string) []gin.H
}) {
	h.containerHandler = ch
}

// SetPubSubHub sets the redis hub for horizontal scaling
func (h *ContainerEventsHub) SetPubSubHub(hub *pubsub.Hub) {
	h.pubsubHub = hub
	if h.pubsubHub != nil {
		h.pubsubHub.Subscribe(pubsub.ChannelContainerEvents, h.handleContainerEventMessage)
		h.pubsubHub.Subscribe(pubsub.ChannelAgentEvents, h.handleAgentEventMessage)
	}
}

// handleContainerEventMessage handles container events from other instances
func (h *ContainerEventsHub) handleContainerEventMessage(msg pubsub.Message) {
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		log.Printf("[ContainerEvents] Failed to unmarshal pubsub message: %v", err)
		return
	}

	userID, _ := payload["user_id"].(string)
	eventType, _ := payload["event"].(string)
	data := payload["data"]

	if userID != "" && eventType != "" {
		// Broadcast to local connections for this user
		h.broadcastLocal(userID, ContainerEvent{
			Type:      eventType,
			Container: data,
			Timestamp: msg.Timestamp,
		})
	}
}

// handleAgentEventMessage handles agent events from other instances
func (h *ContainerEventsHub) handleAgentEventMessage(msg pubsub.Message) {
	// Re-use container event logic as the structure is similar
	h.handleContainerEventMessage(msg)
}

// HandleWebSocket handles WebSocket connections for container events
func (h *ContainerEventsHub) HandleWebSocket(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tier := c.GetString("tier")

	// Upgrade to WebSocket with subprotocol support
	responseHeader := http.Header{}
	requestedProtocols := c.GetHeader("Sec-WebSocket-Protocol")
	if strings.Contains(requestedProtocols, "rexec.v1") {
		responseHeader.Set("Sec-WebSocket-Protocol", "rexec.v1")
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, responseHeader)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Register connection
	sc := h.registerConnection(userID, conn)

	// Send initial container list
	h.sendContainerList(sc, userID, tier)

	// Handle connection lifecycle
	go h.handleConnection(sc, userID, tier)
}

// registerConnection adds a connection to the hub and returns the safe wrapper
func (h *ContainerEventsHub) registerConnection(userID string, conn *websocket.Conn) *SafeConn {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.connections[userID] == nil {
		h.connections[userID] = make(map[*websocket.Conn]*SafeConn)
	}

	sc := &SafeConn{Conn: conn}
	h.connections[userID][conn] = sc

	log.Printf("[ContainerEvents] User %s connected (total connections: %d)", userID, len(h.connections[userID]))
	return sc
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
func (h *ContainerEventsHub) handleConnection(sc *SafeConn, userID, tier string) {
	defer h.unregisterConnection(userID, sc.Conn)

	// Set up ping/pong for connection health - use longer timeout for stability
	sc.Conn.SetReadDeadline(time.Now().Add(180 * time.Second)) // 3 minutes
	sc.Conn.SetPongHandler(func(string) error {
		sc.Conn.SetReadDeadline(time.Now().Add(180 * time.Second))
		return nil
	})

	// Ping ticker - send pings to keep connection alive
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Channel to signal when read loop exits
	done := make(chan struct{})

	// Read messages (ping from client, pong responses, and keepalive)
	go func() {
		defer close(done)
		for {
			_, message, err := sc.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[ContainerEvents] Read error: %v", err)
				}
				return
			}

			// Reset read deadline on any message
			sc.Conn.SetReadDeadline(time.Now().Add(180 * time.Second))

			// Try to parse as JSON to handle client pings
			var msg struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(message, &msg); err == nil {
				if msg.Type == "ping" {
					// Respond with pong
					pongMsg, _ := json.Marshal(map[string]string{"type": "pong"})
					// Use SafeConn for writing
					if err := sc.WriteMessage(websocket.TextMessage, pongMsg); err != nil {
						return
					}
				}
			}
		}
	}()

	// Send pings periodically
	for {
		select {
		case <-done:
			return
		case <-pingTicker.C:
			// Use SafeConn for writing
			if err := sc.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sendContainerList sends the full container list to a connection
func (h *ContainerEventsHub) sendContainerList(sc *SafeConn, userID, tier string) {
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

		// Use DockerID as primary ID, fallback to DB ID for error state containers
		containerID := record.DockerID
		if containerID == "" {
			containerID = record.ID
		}
		containers = append(containers, gin.H{
			"id":           containerID,
			"db_id":        record.ID,
			"user_id":      record.UserID,
			"name":         record.Name,
			"image":        record.Image,
			"role":         record.Role,
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

	// Include online agents in the list
	if h.agentHandler != nil {
		onlineAgents := h.agentHandler.GetOnlineAgentsForUser(userID)
		// Prepend agents to containers (agents first)
		containers = append(onlineAgents, containers...)
	}

	// Include all shared terminals (both containers and agents from collab sessions)
	if h.containerHandler != nil {
		ctx := context.Background()
		sharedTerminals := h.containerHandler.GetSharedTerminalsForUser(ctx, userID)
		containers = append(containers, sharedTerminals...)
	}

	// Sort unified list by created_at descending (newest first)
	sort.Slice(containers, func(i, j int) bool {
		t1 := containers[i]["created_at"].(time.Time)
		t2 := containers[j]["created_at"].(time.Time)
		return t1.After(t2)
	})

	event := ContainerEvent{
		Type: "list",
		Container: gin.H{
			"containers": containers,
			"count":      len(containers),
			"limit":      container.UserContainerLimit(tier),
		},
		Timestamp: time.Now(),
	}

	h.sendToConnection(sc, event)
}

// sendToConnection sends an event to a specific connection
func (h *ContainerEventsHub) sendToConnection(sc *SafeConn, event ContainerEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[ContainerEvents] Failed to marshal event: %v", err)
		return
	}

	if err := sc.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("[ContainerEvents] Failed to send event: %v", err)
	}
}

// broadcastLocal sends an event to all LOCAL connections for a user
func (h *ContainerEventsHub) broadcastLocal(userID string, event ContainerEvent) {
	h.mu.RLock()
	conns, ok := h.connections[userID]
	if !ok {
		h.mu.RUnlock()
		// No local connections, but might have remote ones, so don't log as error/skip
		return
	}

	// Copy connections to avoid holding lock during send
	connList := make([]*SafeConn, 0, len(conns))
	for _, sc := range conns {
		connList = append(connList, sc)
	}
	h.mu.RUnlock()

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[ContainerEvents] Failed to marshal event: %v", err)
		return
	}

	// Track dead connections to clean up
	var deadConns []*websocket.Conn

	for _, sc := range connList {
		// Use SafeConn.WriteMessage for thread safety
		if err := sc.WriteMessage(websocket.TextMessage, data); err != nil {
			// Connection is dead, mark for cleanup (don't spam logs for broken pipe)
			deadConns = append(deadConns, sc.Conn)
		}
	}

	// Clean up dead connections
	if len(deadConns) > 0 {
		h.mu.Lock()
		if userConns, ok := h.connections[userID]; ok {
			for _, deadConn := range deadConns {
				delete(userConns, deadConn)
				deadConn.Close()
			}
			if len(userConns) == 0 {
				delete(h.connections, userID)
			}
		}
		h.mu.Unlock()
		// Log once for all dead connections
		log.Printf("[ContainerEvents] Cleaned up %d dead connection(s) for user %s", len(deadConns), userID)
	}
}

// BroadcastToUser sends an event to all connections for a user (local and remote)
func (h *ContainerEventsHub) BroadcastToUser(userID string, event ContainerEvent) {
	// 1. Send to local connections
	h.broadcastLocal(userID, event)

	// 2. Publish to Redis for remote connections
	if h.pubsubHub != nil {
		// Determine channel based on event type (agent events go to agent channel)
		channel := pubsub.ChannelContainerEvents
		if event.Type == "agent_connected" || event.Type == "agent_disconnected" || event.Type == "agent_stats" {
			channel = pubsub.ChannelAgentEvents
		}

		// Payload matches what PublishContainerEvent expects
		payload := map[string]interface{}{
			"user_id": userID,
			"event":   event.Type,
			"data":    event.Container,
		}

		// Use raw Publish to control message structure matching handleContainerEventMessage expectations
		h.pubsubHub.Publish(channel, event.Type, payload)
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
func (h *ContainerEventsHub) NotifyContainerDeleted(userID string, containerID string, dbID ...string) {
	data := gin.H{"id": containerID}
	// Include db_id if provided
	if len(dbID) > 0 && dbID[0] != "" {
		data["db_id"] = dbID[0]
	}
	h.BroadcastToUser(userID, ContainerEvent{
		Type:      "deleted",
		Container: data,
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

// NotifyContainerProgress notifies a user of container creation progress
func (h *ContainerEventsHub) NotifyContainerProgress(userID string, progressData interface{}) {
	h.BroadcastToUser(userID, ContainerEvent{
		Type:      "progress",
		Container: progressData,
		Timestamp: time.Now(),
	})
}

// NotifyAgentConnected notifies a user that an agent connected
func (h *ContainerEventsHub) NotifyAgentConnected(userID string, agentData interface{}) {
	h.BroadcastToUser(userID, ContainerEvent{
		Type:      "agent_connected",
		Container: agentData,
		Timestamp: time.Now(),
	})
}

// NotifyAgentDisconnected notifies a user that an agent disconnected
func (h *ContainerEventsHub) NotifyAgentDisconnected(userID string, agentID string) {
	h.BroadcastToUser(userID, ContainerEvent{
		Type:      "agent_disconnected",
		Container: gin.H{"id": "agent:" + agentID},
		Timestamp: time.Now(),
	})
}

// NotifyAgentStatsUpdated notifies a user that an agent's stats have been updated
func (h *ContainerEventsHub) NotifyAgentStatsUpdated(userID string, agentID string, stats interface{}) {
	h.BroadcastToUser(userID, ContainerEvent{
		Type: "agent_stats",
		Container: gin.H{
			"id":    "agent:" + agentID,
			"stats": stats,
		},
		Timestamp: time.Now(),
	})
}
