// Package pubsub provides Redis-based pub/sub for cross-instance WebSocket communication.
// This enables horizontal scaling by allowing WebSocket events to be broadcast across
// all server instances.
package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Message represents a pub/sub message
type Message struct {
	Type       string          `json:"type"`       // Message type (e.g., "container_event", "agent_connected")
	Channel    string          `json:"channel"`    // Target channel (e.g., "user:<id>", "agent:<id>")
	InstanceID string          `json:"instance_id"` // Source instance ID
	Payload    json.RawMessage `json:"payload"`    // Message payload
	Timestamp  time.Time       `json:"timestamp"`
}

// AgentLocationMessage represents an agent's connection location
type AgentLocationMessage struct {
	AgentID     string `json:"agent_id"`
	InstanceID  string `json:"instance_id"`
	UserID      string `json:"user_id"`
	Status      string `json:"status"` // "connected", "disconnected"
	Name        string `json:"name,omitempty"`
	OS          string `json:"os,omitempty"`
	Arch        string `json:"arch,omitempty"`
	ConnectedAt time.Time `json:"connected_at,omitempty"`
}

// TerminalProxyMessage represents terminal I/O to be proxied
type TerminalProxyMessage struct {
	SessionID  string `json:"session_id"`
	AgentID    string `json:"agent_id"`
	Type       string `json:"type"`    // "input", "output", "resize", "close"
	Data       []byte `json:"data"`
	Cols       int    `json:"cols,omitempty"`
	Rows       int    `json:"rows,omitempty"`
}

// Hub manages Redis pub/sub connections and message routing
type Hub struct {
	client     *redis.Client
	instanceID string
	handlers   map[string][]MessageHandler
	handlersMu sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	
	// Connection health tracking
	connected     bool
	connectedMu   sync.RWMutex
	reconnectCh   chan struct{}
	
	// Agent location cache (agent_id -> instance_id)
	agentLocations   map[string]string
	agentLocationsMu sync.RWMutex
}

// MessageHandler is a function that handles incoming messages
type MessageHandler func(msg Message)

// Channels for pub/sub
const (
	ChannelContainerEvents = "rexec:container_events"
	ChannelAgentEvents     = "rexec:agent_events"
	ChannelAgentLocations  = "rexec:agent_locations"
	ChannelTerminalProxy   = "rexec:terminal_proxy"
	ChannelAdminEvents     = "rexec:admin_events"
)

// Redis keys for agent locations
const (
	KeyAgentLocation = "rexec:agent:location:" // + agent_id -> instance_id
	KeyAgentTTL      = 60 * time.Second        // Agent location TTL
)

// NewHub creates a new Redis pub/sub hub
func NewHub() (*Hub, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, errors.New("REDIS_URL not configured")
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	// Generate unique instance ID
	hostname, _ := os.Hostname()
	instanceID := hostname + "-" + generateShortID()

	hubCtx, hubCancel := context.WithCancel(context.Background())

	hub := &Hub{
		client:         client,
		instanceID:     instanceID,
		handlers:       make(map[string][]MessageHandler),
		ctx:            hubCtx,
		cancel:         hubCancel,
		connected:      true,
		reconnectCh:    make(chan struct{}, 1),
		agentLocations: make(map[string]string),
	}

	log.Printf("[PubSub] Connected to Redis, instance ID: %s", instanceID)

	return hub, nil
}

// Start begins listening on all channels
func (h *Hub) Start() {
	channels := []string{
		ChannelContainerEvents,
		ChannelAgentEvents,
		ChannelAgentLocations,
		ChannelTerminalProxy,
		ChannelAdminEvents,
	}

	h.wg.Add(1)
	go h.subscribeLoop(channels)

	// Start heartbeat/health check
	h.wg.Add(1)
	go h.healthCheckLoop()
}

// Stop shuts down the hub
func (h *Hub) Stop() {
	h.cancel()
	h.wg.Wait()
	h.client.Close()
	log.Printf("[PubSub] Hub stopped")
}

// subscribeLoop handles Redis subscription with automatic reconnection
func (h *Hub) subscribeLoop(channels []string) {
	defer h.wg.Done()

	for {
		select {
		case <-h.ctx.Done():
			return
		default:
		}

		pubsub := h.client.Subscribe(h.ctx, channels...)
		
		// Wait for subscription confirmation
		_, err := pubsub.Receive(h.ctx)
		if err != nil {
			log.Printf("[PubSub] Subscription failed: %v, retrying in 5s", err)
			h.setConnected(false)
			time.Sleep(5 * time.Second)
			continue
		}

		h.setConnected(true)
		log.Printf("[PubSub] Subscribed to channels: %v", channels)

		// Process messages
		ch := pubsub.Channel()
		for {
			select {
			case <-h.ctx.Done():
				pubsub.Close()
				return
			case msg, ok := <-ch:
				if !ok {
					log.Printf("[PubSub] Channel closed, reconnecting...")
					h.setConnected(false)
					break
				}
				h.handleMessage(msg)
			}
		}

		pubsub.Close()
		time.Sleep(1 * time.Second) // Brief pause before reconnection
	}
}

// handleMessage processes incoming Redis messages
func (h *Hub) handleMessage(redisMsg *redis.Message) {
	var msg Message
	if err := json.Unmarshal([]byte(redisMsg.Payload), &msg); err != nil {
		log.Printf("[PubSub] Failed to unmarshal message: %v", err)
		return
	}

	// Skip messages from our own instance (except for specific types)
	if msg.InstanceID == h.instanceID && msg.Type != "agent_location" {
		return
	}

	// Call registered handlers
	h.handlersMu.RLock()
	handlers := h.handlers[redisMsg.Channel]
	h.handlersMu.RUnlock()

	for _, handler := range handlers {
		handler(msg)
	}
}

// Subscribe adds a handler for a specific channel
func (h *Hub) Subscribe(channel string, handler MessageHandler) {
	h.handlersMu.Lock()
	h.handlers[channel] = append(h.handlers[channel], handler)
	h.handlersMu.Unlock()
}

// Publish sends a message to a channel
func (h *Hub) Publish(channel string, msgType string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[PubSub] Failed to marshal payload: %v", err)
		return err
	}

	msg := Message{
		Type:       msgType,
		Channel:    channel,
		InstanceID: h.instanceID,
		Payload:    payloadBytes,
		Timestamp:  time.Now(),
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[PubSub] Failed to marshal message: %v", err)
		return err
	}

	// Use a fresh context for publishing to avoid issues with canceled hub context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	result := h.client.Publish(ctx, channel, msgBytes)
	if err := result.Err(); err != nil {
		log.Printf("[PubSub] Failed to publish to %s: %v", channel, err)
		return err
	}
	
	// Log successful publish for debugging
	log.Printf("[PubSub] Published to %s: type=%s, receivers=%d", channel, msgType, result.Val())
	return nil
}

// PublishContainerEvent publishes a container event for a user
func (h *Hub) PublishContainerEvent(userID string, eventType string, data interface{}) error {
	payload := map[string]interface{}{
		"user_id": userID,
		"event":   eventType,
		"data":    data,
	}
	return h.Publish(ChannelContainerEvents, eventType, payload)
}

// PublishAgentEvent publishes an agent-related event
func (h *Hub) PublishAgentEvent(userID string, eventType string, data interface{}) error {
	payload := map[string]interface{}{
		"user_id": userID,
		"event":   eventType,
		"data":    data,
	}
	return h.Publish(ChannelAgentEvents, eventType, payload)
}

// RegisterAgentLocation registers which instance an agent is connected to
func (h *Hub) RegisterAgentLocation(agentID, userID, name, os, arch string, connectedAt time.Time) error {
	ctx := context.Background()
	
	// Store in Redis with TTL
	key := KeyAgentLocation + agentID
	if err := h.client.Set(ctx, key, h.instanceID, KeyAgentTTL).Err(); err != nil {
		return err
	}

	// Update local cache
	h.agentLocationsMu.Lock()
	h.agentLocations[agentID] = h.instanceID
	h.agentLocationsMu.Unlock()

	// Broadcast location update
	msg := AgentLocationMessage{
		AgentID:     agentID,
		InstanceID:  h.instanceID,
		UserID:      userID,
		Status:      "connected",
		Name:        name,
		OS:          os,
		Arch:        arch,
		ConnectedAt: connectedAt,
	}
	return h.Publish(ChannelAgentLocations, "agent_location", msg)
}

// UnregisterAgentLocation removes an agent's location
func (h *Hub) UnregisterAgentLocation(agentID, userID string) error {
	ctx := context.Background()
	
	// Remove from Redis
	key := KeyAgentLocation + agentID
	if err := h.client.Del(ctx, key).Err(); err != nil {
		log.Printf("[PubSub] Failed to delete agent location: %v", err)
	}

	// Update local cache
	h.agentLocationsMu.Lock()
	delete(h.agentLocations, agentID)
	h.agentLocationsMu.Unlock()

	// Broadcast disconnection
	msg := AgentLocationMessage{
		AgentID:    agentID,
		InstanceID: h.instanceID,
		UserID:     userID,
		Status:     "disconnected",
	}
	return h.Publish(ChannelAgentLocations, "agent_location", msg)
}

// GetAgentLocation returns which instance an agent is connected to
func (h *Hub) GetAgentLocation(agentID string) (string, bool) {
	// Check local cache first
	h.agentLocationsMu.RLock()
	instanceID, ok := h.agentLocations[agentID]
	h.agentLocationsMu.RUnlock()
	if ok {
		return instanceID, true
	}

	// Check Redis
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := KeyAgentLocation + agentID
	instanceID, err := h.client.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}

	// Update local cache
	h.agentLocationsMu.Lock()
	h.agentLocations[agentID] = instanceID
	h.agentLocationsMu.Unlock()

	return instanceID, true
}

// IsAgentLocal checks if an agent is connected to this instance
func (h *Hub) IsAgentLocal(agentID string) bool {
	instanceID, ok := h.GetAgentLocation(agentID)
	return ok && instanceID == h.instanceID
}

// ProxyTerminalData sends terminal data to the instance hosting an agent
func (h *Hub) ProxyTerminalData(agentID, sessionID, msgType string, data []byte, cols, rows int) error {
	msg := TerminalProxyMessage{
		SessionID: sessionID,
		AgentID:   agentID,
		Type:      msgType,
		Data:      data,
		Cols:      cols,
		Rows:      rows,
	}
	return h.Publish(ChannelTerminalProxy, "terminal_proxy", msg)
}

// RefreshAgentLocation refreshes the TTL for an agent's location
func (h *Hub) RefreshAgentLocation(agentID string) error {
	ctx := context.Background()
	key := KeyAgentLocation + agentID
	return h.client.Expire(ctx, key, KeyAgentTTL).Err()
}

// healthCheckLoop periodically checks Redis connection health
func (h *Hub) healthCheckLoop() {
	defer h.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := h.client.Ping(ctx).Err()
			cancel()

			if err != nil {
				log.Printf("[PubSub] Health check failed: %v", err)
				h.setConnected(false)
			} else {
				h.setConnected(true)
			}
		}
	}
}

// setConnected updates the connection status
func (h *Hub) setConnected(connected bool) {
	h.connectedMu.Lock()
	defer h.connectedMu.Unlock()
	
	if h.connected != connected {
		h.connected = connected
		if connected {
			log.Printf("[PubSub] Connection restored")
		} else {
			log.Printf("[PubSub] Connection lost")
		}
	}
}

// IsConnected returns the current connection status
func (h *Hub) IsConnected() bool {
	h.connectedMu.RLock()
	defer h.connectedMu.RUnlock()
	return h.connected
}

// InstanceID returns this instance's unique ID
func (h *Hub) InstanceID() string {
	return h.instanceID
}

// generateShortID generates a short random ID
func generateShortID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(1 * time.Nanosecond)
	}
	return string(b)
}
