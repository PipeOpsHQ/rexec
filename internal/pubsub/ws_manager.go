// Package pubsub provides WebSocket connection management with Redis pub/sub.
package pubsub

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSConnection wraps a WebSocket connection with metadata
type WSConnection struct {
	Conn       *websocket.Conn
	UserID     string
	ConnID     string
	CreatedAt  time.Time
	LastPingAt time.Time
	mu         sync.Mutex
}

// WSManager manages WebSocket connections and integrates with Redis pub/sub
type WSManager struct {
	hub *Hub
	
	// User connections: userID -> map of connID -> connection
	connections   map[string]map[string]*WSConnection
	connectionsMu sync.RWMutex
	
	// Agent terminal sessions waiting for proxy responses
	// sessionID -> channel to receive output
	proxySessions   map[string]chan []byte
	proxySessionsMu sync.RWMutex
}

// NewWSManager creates a new WebSocket manager
func NewWSManager(hub *Hub) *WSManager {
	mgr := &WSManager{
		hub:           hub,
		connections:   make(map[string]map[string]*WSConnection),
		proxySessions: make(map[string]chan []byte),
	}

	// Subscribe to relevant channels
	if hub != nil {
		hub.Subscribe(ChannelContainerEvents, mgr.handleContainerEvent)
		hub.Subscribe(ChannelAgentEvents, mgr.handleAgentEvent)
		hub.Subscribe(ChannelTerminalProxy, mgr.handleTerminalProxy)
	}

	return mgr
}

// RegisterConnection adds a WebSocket connection
func (m *WSManager) RegisterConnection(userID, connID string, conn *websocket.Conn) *WSConnection {
	wsConn := &WSConnection{
		Conn:       conn,
		UserID:     userID,
		ConnID:     connID,
		CreatedAt:  time.Now(),
		LastPingAt: time.Now(),
	}

	m.connectionsMu.Lock()
	if m.connections[userID] == nil {
		m.connections[userID] = make(map[string]*WSConnection)
	}
	m.connections[userID][connID] = wsConn
	m.connectionsMu.Unlock()

	log.Printf("[WSManager] Registered connection %s for user %s", connID, userID)
	return wsConn
}

// UnregisterConnection removes a WebSocket connection
func (m *WSManager) UnregisterConnection(userID, connID string) {
	m.connectionsMu.Lock()
	if conns, ok := m.connections[userID]; ok {
		delete(conns, connID)
		if len(conns) == 0 {
			delete(m.connections, userID)
		}
	}
	m.connectionsMu.Unlock()

	log.Printf("[WSManager] Unregistered connection %s for user %s", connID, userID)
}

// BroadcastToUser sends a message to all connections for a user (local only)
func (m *WSManager) BroadcastToUser(userID string, message interface{}) {
	m.connectionsMu.RLock()
	conns, ok := m.connections[userID]
	if !ok {
		m.connectionsMu.RUnlock()
		return
	}

	// Copy to avoid holding lock during sends
	connList := make([]*WSConnection, 0, len(conns))
	for _, conn := range conns {
		connList = append(connList, conn)
	}
	m.connectionsMu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("[WSManager] Failed to marshal message: %v", err)
		return
	}

	for _, conn := range connList {
		conn.mu.Lock()
		err := conn.Conn.WriteMessage(websocket.TextMessage, data)
		conn.mu.Unlock()
		if err != nil {
			log.Printf("[WSManager] Failed to send to %s: %v", conn.ConnID, err)
		}
	}
}

// BroadcastToUserGlobal sends a message to all instances serving a user
func (m *WSManager) BroadcastToUserGlobal(userID string, eventType string, data interface{}) {
	// First, broadcast locally
	m.BroadcastToUser(userID, data)

	// Then, publish to Redis for other instances
	if m.hub != nil {
		if err := m.hub.PublishContainerEvent(userID, eventType, data); err != nil {
			log.Printf("[WSManager] Failed to publish event: %v", err)
		}
	}
}

// handleContainerEvent processes container events from Redis
func (m *WSManager) handleContainerEvent(msg Message) {
	var payload struct {
		UserID string      `json:"user_id"`
		Event  string      `json:"event"`
		Data   interface{} `json:"data"`
	}
	
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		log.Printf("[WSManager] Failed to unmarshal container event: %v", err)
		return
	}

	// Broadcast to local connections for this user
	m.BroadcastToUser(payload.UserID, payload.Data)
}

// handleAgentEvent processes agent events from Redis
func (m *WSManager) handleAgentEvent(msg Message) {
	var payload struct {
		UserID string      `json:"user_id"`
		Event  string      `json:"event"`
		Data   interface{} `json:"data"`
	}
	
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		log.Printf("[WSManager] Failed to unmarshal agent event: %v", err)
		return
	}

	// Broadcast to local connections for this user
	m.BroadcastToUser(payload.UserID, payload.Data)
}

// RegisterProxySession creates a channel for receiving proxied terminal output
func (m *WSManager) RegisterProxySession(sessionID string) chan []byte {
	ch := make(chan []byte, 256)
	
	m.proxySessionsMu.Lock()
	m.proxySessions[sessionID] = ch
	m.proxySessionsMu.Unlock()
	
	return ch
}

// UnregisterProxySession removes a proxy session
func (m *WSManager) UnregisterProxySession(sessionID string) {
	m.proxySessionsMu.Lock()
	if ch, ok := m.proxySessions[sessionID]; ok {
		close(ch)
		delete(m.proxySessions, sessionID)
	}
	m.proxySessionsMu.Unlock()
}

// handleTerminalProxy processes terminal proxy messages from Redis
func (m *WSManager) handleTerminalProxy(msg Message) {
	var proxyMsg TerminalProxyMessage
	if err := json.Unmarshal(msg.Payload, &proxyMsg); err != nil {
		log.Printf("[WSManager] Failed to unmarshal terminal proxy: %v", err)
		return
	}

	// Find the session channel and forward data
	m.proxySessionsMu.RLock()
	ch, ok := m.proxySessions[proxyMsg.SessionID]
	m.proxySessionsMu.RUnlock()

	if ok && proxyMsg.Type == "output" {
		select {
		case ch <- proxyMsg.Data:
		default:
			log.Printf("[WSManager] Proxy session %s buffer full", proxyMsg.SessionID)
		}
	}
}

// GetConnectionCount returns the total number of active connections
func (m *WSManager) GetConnectionCount() int {
	m.connectionsMu.RLock()
	defer m.connectionsMu.RUnlock()

	count := 0
	for _, conns := range m.connections {
		count += len(conns)
	}
	return count
}

// GetUserConnectionCount returns the number of connections for a user
func (m *WSManager) GetUserConnectionCount(userID string) int {
	m.connectionsMu.RLock()
	defer m.connectionsMu.RUnlock()

	if conns, ok := m.connections[userID]; ok {
		return len(conns)
	}
	return 0
}

// SendToConnection sends a message to a specific connection
func (m *WSManager) SendToConnection(userID, connID string, message interface{}) error {
	m.connectionsMu.RLock()
	conns, ok := m.connections[userID]
	if !ok {
		m.connectionsMu.RUnlock()
		return nil
	}
	conn, ok := conns[connID]
	m.connectionsMu.RUnlock()

	if !ok {
		return nil
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()
	return conn.Conn.WriteJSON(message)
}
