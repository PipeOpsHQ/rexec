package handlers

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rexec/rexec/internal/container"
	"github.com/rexec/rexec/internal/storage"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024, // Small read buffer, admin clients shouldn't send much
	WriteBufferSize: 128 * 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, you should validate the origin
		return true
	},
	HandshakeTimeout:  10 * time.Second,
	EnableCompression: true,
}

const (
	wsWriteDeadline = 5 * time.Second
)

// AdminEvent represents a message to be broadcast to admin clients
type AdminEvent struct {
	Type    string      `json:"type"` // e.g., "user_updated", "container_deleted", "session_created"
	Payload interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}

// AdminClient represents a single connected admin WebSocket client
type AdminClient struct {
	conn *websocket.Conn
	send chan AdminEvent
}

// AdminEventsHub manages the WebSocket connections for admin clients
type AdminEventsHub struct {
	clients   map[*AdminClient]bool
	broadcast chan AdminEvent
	register  chan *AdminClient
	unregister chan *AdminClient
	store *storage.PostgresStore
	containerManager *container.Manager
	mu        sync.RWMutex
}

// NewAdminEventsHub creates a new AdminEventsHub
func NewAdminEventsHub(store *storage.PostgresStore, cm *container.Manager) *AdminEventsHub {
	hub := &AdminEventsHub{
		broadcast: make(chan AdminEvent),
		register:  make(chan *AdminClient),
		unregister: make(chan *AdminClient),
		clients:   make(map[*AdminClient]bool),
		store: store,
		containerManager: cm,
	}

	go hub.run()
	return hub
}

// run starts the hub's event loop
func (h *AdminEventsHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case event := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- event:
				default:
					// Client's send buffer is full, drop event and unregister client
					close(client.send)
					h.mu.RUnlock()
					h.mu.Lock()
					delete(h.clients, client)
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// HandleWebSocket upgrades the HTTP connection to a WebSocket and registers the client
func (h *AdminEventsHub) HandleWebSocket(c *gin.Context) {
	// Add subprotocol support
	responseHeader := http.Header{}
	requestedProtocols := c.GetHeader("Sec-WebSocket-Protocol")
	if strings.Contains(requestedProtocols, "rexec.v1") {
		responseHeader.Set("Sec-WebSocket-Protocol", "rexec.v1")
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, responseHeader)
	if err != nil {
		log.Printf("AdminEventsHub: WebSocket upgrade failed: %v", err)
		return
	}

	client := &AdminClient{conn: conn, send: make(chan AdminEvent, 256)}
	h.register <- client

	// Allow client to receive messages
	go client.writePump()

	// Prevent client from sending messages (admin events are server-sent)
	client.readPump(h)
}

// writePump pumps messages from the hub to the WebSocket connection.
func (c *AdminClient) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for event := range c.send {
		c.conn.SetWriteDeadline(time.Now().Add(wsWriteDeadline))
		err := c.conn.WriteJSON(event)
		if err != nil {
			log.Printf("AdminEventsHub: Failed to write event to client: %v", err)
			return
		}
	}
}

// readPump prevents clients from sending messages to the hub.
func (c *AdminClient) readPump(h *AdminEventsHub) {
	defer func() {
		h.unregister <- c
		c.conn.Close()
	}()
	
	// Set a read limit and deadline to prevent malicious clients from holding connections
	c.conn.SetReadLimit(512) // Very small read limit as clients shouldn't send data
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // Expect pings
	for {
		// We don't expect any messages from the client other than pings/pongs implicitly
		// So, just read a message to detect disconnection.
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
	}
}

// Broadcast sends an event to all registered admin clients
func (h *AdminEventsHub) Broadcast(eventType string, payload interface{}) {
	event := AdminEvent{
		Type:    eventType,
		Payload: payload,
		Timestamp: time.Now(),
	}
	select {
	case h.broadcast <- event:
		// Event sent to broadcast channel
	default:
		// Hub's broadcast channel is full, event dropped
		log.Println("AdminEventsHub: Broadcast channel full, event dropped.")
	}
}
