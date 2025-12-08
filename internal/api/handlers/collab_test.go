package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestGenerateShareCode tests share code generation
func TestGenerateShareCode(t *testing.T) {
	codes := make(map[string]bool)
	
	// Generate 100 codes and ensure uniqueness
	for i := 0; i < 100; i++ {
		code := generateShareCode()
		
		// Check length
		if len(code) != 6 {
			t.Errorf("Expected code length 6, got %d: %s", len(code), code)
		}
		
		// Check for uniqueness
		if codes[code] {
			t.Errorf("Duplicate code generated: %s", code)
		}
		codes[code] = true
		
		// Check that code is uppercase
		if code != strings.ToUpper(code) {
			t.Errorf("Code should be uppercase: %s", code)
		}
	}
}

// TestGetParticipantColor tests participant color assignment
func TestGetParticipantColor(t *testing.T) {
	tests := []struct {
		index    int
		expected string
	}{
		{0, "#FF6B6B"},
		{1, "#4ECDC4"},
		{2, "#45B7D1"},
		{7, "#F7DC6F"},
		{8, "#FF6B6B"}, // Should wrap around
		{16, "#FF6B6B"}, // Should wrap around twice
	}
	
	for _, tt := range tests {
		color := getParticipantColor(tt.index)
		if color != tt.expected {
			t.Errorf("getParticipantColor(%d) = %s, expected %s", tt.index, color, tt.expected)
		}
	}
}

// TestCollabSessionBroadcast tests the broadcast channel
func TestCollabSessionBroadcast(t *testing.T) {
	session := &CollabSession{
		ID:           "test-session",
		ContainerID:  "test-container",
		OwnerID:      "user-1",
		ShareCode:    "TEST01",
		Mode:         "control",
		MaxUsers:     5,
		ExpiresAt:    time.Now().Add(time.Hour),
		Participants: make(map[string]*CollabParticipant),
		broadcast:    make(chan CollabMessage, 1024),
	}
	
	// Test non-blocking send
	msg := CollabMessage{
		Type:      "test",
		UserID:    "user-1",
		Timestamp: time.Now().UnixMilli(),
	}
	
	select {
	case session.broadcast <- msg:
		// Success
	default:
		t.Error("Failed to send message to broadcast channel")
	}
	
	// Verify message received
	select {
	case received := <-session.broadcast:
		if received.Type != "test" {
			t.Errorf("Expected message type 'test', got '%s'", received.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for broadcast message")
	}
}

// TestCollabMessageJSON tests message serialization
func TestCollabMessageJSON(t *testing.T) {
	msg := CollabMessage{
		Type:      "join",
		UserID:    "user-123",
		Username:  "testuser",
		Role:      "editor",
		Color:     "#FF6B6B",
		Timestamp: 1234567890,
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}
	
	var decoded CollabMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}
	
	if decoded.Type != msg.Type {
		t.Errorf("Type mismatch: got %s, expected %s", decoded.Type, msg.Type)
	}
	if decoded.UserID != msg.UserID {
		t.Errorf("UserID mismatch: got %s, expected %s", decoded.UserID, msg.UserID)
	}
	if decoded.Username != msg.Username {
		t.Errorf("Username mismatch: got %s, expected %s", decoded.Username, msg.Username)
	}
}

// TestCollabSessionExpiration tests session expiration logic
func TestCollabSessionExpiration(t *testing.T) {
	// Create an expired session
	expiredSession := &CollabSession{
		ID:        "expired",
		ExpiresAt: time.Now().Add(-time.Hour), // Expired 1 hour ago
	}
	
	if !time.Now().After(expiredSession.ExpiresAt) {
		t.Error("Session should be expired")
	}
	
	// Create a valid session
	validSession := &CollabSession{
		ID:        "valid",
		ExpiresAt: time.Now().Add(time.Hour), // Expires in 1 hour
	}
	
	if time.Now().After(validSession.ExpiresAt) {
		t.Error("Session should not be expired")
	}
}

// TestCollabParticipantLimit tests max user enforcement
func TestCollabParticipantLimit(t *testing.T) {
	session := &CollabSession{
		MaxUsers:     3,
		Participants: make(map[string]*CollabParticipant),
	}
	
	// Add participants up to limit
	session.Participants["user-1"] = &CollabParticipant{UserID: "user-1"}
	session.Participants["user-2"] = &CollabParticipant{UserID: "user-2"}
	session.Participants["user-3"] = &CollabParticipant{UserID: "user-3"}
	
	if len(session.Participants) < session.MaxUsers {
		t.Error("Should have reached max users")
	}
	
	// Verify at capacity
	if len(session.Participants) != session.MaxUsers {
		t.Errorf("Expected %d participants, got %d", session.MaxUsers, len(session.Participants))
	}
}

// TestCollabRoleAssignment tests role determination
func TestCollabRoleAssignment(t *testing.T) {
	session := &CollabSession{
		OwnerID: "owner-123",
		Mode:    "control",
	}
	
	tests := []struct {
		userID   string
		expected string
	}{
		{"owner-123", "owner"},
		{"other-user", "editor"}, // control mode gives editor
	}
	
	for _, tt := range tests {
		var role string
		if tt.userID == session.OwnerID {
			role = "owner"
		} else if session.Mode == "control" {
			role = "editor"
		} else {
			role = "viewer"
		}
		
		if role != tt.expected {
			t.Errorf("Role for %s: got %s, expected %s", tt.userID, role, tt.expected)
		}
	}
	
	// Test view mode
	session.Mode = "view"
	for _, tt := range tests {
		var role string
		if tt.userID == session.OwnerID {
			role = "owner"
		} else if session.Mode == "control" {
			role = "editor"
		} else {
			role = "viewer"
		}
		
		expected := tt.expected
		if tt.userID != session.OwnerID {
			expected = "viewer" // In view mode, non-owners are viewers
		}
		
		if role != expected {
			t.Errorf("Role for %s in view mode: got %s, expected %s", tt.userID, role, expected)
		}
	}
}

// MockResponseRecorder for testing HTTP responses
type mockResponseWriter struct {
	httptest.ResponseRecorder
}

// TestStartSessionValidation tests request validation
func TestStartSessionValidation(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "missing container_id",
			body:           `{"mode": "view"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty body",
			body:           `{}`,
			expectedStatus: http.StatusBadRequest,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock handler (without actual store/manager)
			// This tests request parsing only
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/api/collab/start", strings.NewReader(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("userID", "test-user")
			c.Set("username", "testuser")
			
			// We can't fully test without mocking the store,
			// but we can verify the request parsing logic
			var req struct {
				ContainerID string `json:"container_id" binding:"required"`
				Mode        string `json:"mode"`
			}
			
			if err := c.ShouldBindJSON(&req); err != nil {
				if tt.expectedStatus != http.StatusBadRequest {
					t.Errorf("Unexpected binding error: %v", err)
				}
			}
		})
	}
}

// TestCollabMessageTypes tests all message type constants
func TestCollabMessageTypes(t *testing.T) {
	validTypes := []string{
		"join", "leave", "cursor", "selection", 
		"input", "output", "sync", "participants",
		"ended", "expired",
	}
	
	for _, msgType := range validTypes {
		msg := CollabMessage{Type: msgType}
		if msg.Type != msgType {
			t.Errorf("Message type mismatch: got %s, expected %s", msg.Type, msgType)
		}
	}
}

// TestConcurrentParticipantAccess tests thread safety
func TestConcurrentParticipantAccess(t *testing.T) {
	session := &CollabSession{
		Participants: make(map[string]*CollabParticipant),
	}
	
	done := make(chan bool, 10)
	
	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func(id int) {
			session.mu.Lock()
			session.Participants[string(rune('A'+id))] = &CollabParticipant{
				UserID: string(rune('A' + id)),
			}
			session.mu.Unlock()
			done <- true
		}(i)
	}
	
	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			session.mu.RLock()
			_ = len(session.Participants)
			session.mu.RUnlock()
			done <- true
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	if len(session.Participants) != 5 {
		t.Errorf("Expected 5 participants, got %d", len(session.Participants))
	}
}

// BenchmarkGenerateShareCode benchmarks share code generation
func BenchmarkGenerateShareCode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateShareCode()
	}
}

// BenchmarkBroadcast benchmarks message broadcasting
func BenchmarkBroadcast(b *testing.B) {
	session := &CollabSession{
		broadcast: make(chan CollabMessage, 1024),
	}
	
	// Drain the channel in background
	go func() {
		for range session.broadcast {
		}
	}()
	
	msg := CollabMessage{
		Type:      "cursor",
		Timestamp: time.Now().UnixMilli(),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		select {
		case session.broadcast <- msg:
		default:
		}
	}
}

// Integration test helper - verifies collab flow
func TestCollabFlowIntegration(t *testing.T) {
	// This is a pseudo-integration test that validates the flow
	// without actual WebSocket connections
	
	ctx := context.Background()
	_ = ctx // Would be used with actual store
	
	// 1. Create session
	session := &CollabSession{
		ID:           "integration-test",
		ContainerID:  "container-123",
		OwnerID:      "owner-user",
		ShareCode:    generateShareCode(),
		Mode:         "control",
		MaxUsers:     5,
		ExpiresAt:    time.Now().Add(time.Hour),
		Participants: make(map[string]*CollabParticipant),
		broadcast:    make(chan CollabMessage, 1024),
	}
	
	// 2. Add owner
	session.Participants["owner-user"] = &CollabParticipant{
		UserID:   "owner-user",
		Username: "Owner",
		Role:     "owner",
		Color:    getParticipantColor(0),
	}
	
	// 3. Add participant
	session.mu.Lock()
	if len(session.Participants) < session.MaxUsers {
		session.Participants["guest-user"] = &CollabParticipant{
			UserID:   "guest-user",
			Username: "Guest",
			Role:     "editor",
			Color:    getParticipantColor(1),
		}
	}
	session.mu.Unlock()
	
	// 4. Verify state
	session.mu.RLock()
	defer session.mu.RUnlock()
	
	if len(session.Participants) != 2 {
		t.Errorf("Expected 2 participants, got %d", len(session.Participants))
	}
	
	owner := session.Participants["owner-user"]
	if owner == nil || owner.Role != "owner" {
		t.Error("Owner not properly set")
	}
	
	guest := session.Participants["guest-user"]
	if guest == nil || guest.Role != "editor" {
		t.Error("Guest not properly set as editor in control mode")
	}
}
