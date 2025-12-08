package handlers

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestGenerateRecordingToken tests token generation
func TestGenerateRecordingToken(t *testing.T) {
	tokens := make(map[string]bool)
	
	// Generate 100 tokens and check uniqueness
	for i := 0; i < 100; i++ {
		token := generateRecordingToken()
		
		// Should be URL-safe (no = padding)
		if strings.Contains(token, "=") {
			t.Errorf("Token should not contain padding: %s", token)
		}
		
		// Check length (16 bytes base64 = ~22 chars without padding)
		if len(token) < 20 {
			t.Errorf("Token too short: %d chars", len(token))
		}
		
		// Check uniqueness
		if tokens[token] {
			t.Errorf("Duplicate token: %s", token)
		}
		tokens[token] = true
	}
}

// TestFormatDuration tests duration formatting
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{5 * time.Second, "5s"},
		{65 * time.Second, "1m 5s"},
		{3661 * time.Second, "1h 1m 1s"},
		{0, "0s"},
		{59 * time.Second, "59s"},
		{60 * time.Second, "1m 0s"},
		{3600 * time.Second, "1h 0m 0s"},
	}
	
	for _, tt := range tests {
		result := formatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", tt.duration, result, tt.expected)
		}
	}
}

// TestRecordingEvent tests event structure
func TestRecordingEvent(t *testing.T) {
	event := RecordingEvent{
		Time: 1234,
		Type: "o",
		Data: "Hello, World!",
	}
	
	// Test JSON serialization
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}
	
	// Verify field names are short
	if !bytes.Contains(data, []byte(`"t":`)) {
		t.Error("Expected short field name 't' for time")
	}
	if !bytes.Contains(data, []byte(`"e":`)) {
		t.Error("Expected short field name 'e' for type")
	}
	if !bytes.Contains(data, []byte(`"d":`)) {
		t.Error("Expected short field name 'd' for data")
	}
	
	// Test resize event
	resizeEvent := RecordingEvent{
		Time: 5678,
		Type: "r",
		Data: "",
		Cols: 120,
		Rows: 30,
	}
	
	resizeData, err := json.Marshal(resizeEvent)
	if err != nil {
		t.Fatalf("Failed to marshal resize event: %v", err)
	}
	
	if !bytes.Contains(resizeData, []byte(`"c":120`)) {
		t.Error("Expected cols in resize event")
	}
	if !bytes.Contains(resizeData, []byte(`"r":30`)) {
		t.Error("Expected rows in resize event")
	}
}

// TestRecordingMetadata tests metadata structure
func TestRecordingMetadata(t *testing.T) {
	meta := RecordingMetadata{
		Version:   2,
		Width:     120,
		Height:    30,
		Timestamp: time.Now(),
		Duration:  60.5,
		Title:     "Test Recording",
	}
	
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Failed to marshal metadata: %v", err)
	}
	
	// Verify required fields
	if !bytes.Contains(data, []byte(`"version":2`)) {
		t.Error("Expected version field")
	}
	if !bytes.Contains(data, []byte(`"width":120`)) {
		t.Error("Expected width field")
	}
	if !bytes.Contains(data, []byte(`"height":30`)) {
		t.Error("Expected height field")
	}
	if !bytes.Contains(data, []byte(`"duration":60.5`)) {
		t.Error("Expected duration field")
	}
}

// TestActiveRecording tests active recording state
func TestActiveRecording(t *testing.T) {
	recording := &ActiveRecording{
		ID:          "test-rec-1",
		ContainerID: "container-123",
		UserID:      "user-456",
		Title:       "My Recording",
		StartedAt:   time.Now(),
		Events:      make([]RecordingEvent, 0),
	}
	
	// Test adding events
	recording.mu.Lock()
	recording.Events = append(recording.Events, RecordingEvent{
		Time: 0,
		Type: "o",
		Data: "First output",
	})
	recording.Events = append(recording.Events, RecordingEvent{
		Time: 100,
		Type: "o",
		Data: "Second output",
	})
	recording.mu.Unlock()
	
	recording.mu.Lock()
	eventCount := len(recording.Events)
	recording.mu.Unlock()
	
	if eventCount != 2 {
		t.Errorf("Expected 2 events, got %d", eventCount)
	}
}

// TestEventTypes tests all valid event types
func TestEventTypes(t *testing.T) {
	validTypes := map[string]string{
		"o": "output",
		"i": "input",
		"r": "resize",
	}
	
	for typeCode, description := range validTypes {
		event := RecordingEvent{
			Time: 0,
			Type: typeCode,
			Data: "test",
		}
		
		if event.Type != typeCode {
			t.Errorf("Event type %s (%s) not set correctly", typeCode, description)
		}
	}
}

// TestAsciicastFormat tests the asciicast v2 format conversion
func TestAsciicastFormat(t *testing.T) {
	// Simulate what convertToAsciicast produces
	recording := &ActiveRecording{
		ID:        "test",
		Title:     "Test",
		StartedAt: time.Now(),
		Events: []RecordingEvent{
			{Time: 0, Type: "o", Data: "Hello"},
			{Time: 100, Type: "o", Data: "World"},
		},
	}
	
	var buf bytes.Buffer
	
	// Write header
	header := RecordingMetadata{
		Version:   2,
		Width:     120,
		Height:    30,
		Timestamp: recording.StartedAt,
		Duration:  0.1,
		Title:     recording.Title,
	}
	
	headerJSON, _ := json.Marshal(header)
	buf.Write(headerJSON)
	buf.WriteByte('\n')
	
	// Write events
	for _, event := range recording.Events {
		timeInSeconds := float64(event.Time) / 1000.0
		eventData := []interface{}{timeInSeconds, event.Type, event.Data}
		eventJSON, _ := json.Marshal(eventData)
		buf.Write(eventJSON)
		buf.WriteByte('\n')
	}
	
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	// Should have header + 2 events = 3 lines
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
	
	// First line should be valid JSON header
	var parsedHeader map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &parsedHeader); err != nil {
		t.Errorf("Failed to parse header: %v", err)
	}
	
	if parsedHeader["version"].(float64) != 2 {
		t.Error("Header version should be 2")
	}
	
	// Event lines should be arrays
	var event1 []interface{}
	if err := json.Unmarshal([]byte(lines[1]), &event1); err != nil {
		t.Errorf("Failed to parse event 1: %v", err)
	}
	
	if len(event1) != 3 {
		t.Errorf("Event should have 3 elements, got %d", len(event1))
	}
}

// TestRecordingHandler_IsRecording tests the IsRecording check
func TestRecordingHandler_IsRecording(t *testing.T) {
	handler := &RecordingHandler{
		recordings: make(map[string]*ActiveRecording),
	}
	
	// Should return false for non-existent container
	if handler.IsRecording("nonexistent") {
		t.Error("Should return false for non-existent container")
	}
	
	// Add a recording
	handler.mu.Lock()
	handler.recordings["container-1"] = &ActiveRecording{
		ID:          "rec-1",
		ContainerID: "container-1",
	}
	handler.mu.Unlock()
	
	// Should return true now
	if !handler.IsRecording("container-1") {
		t.Error("Should return true for recording container")
	}
	
	// Different container should still be false
	if handler.IsRecording("container-2") {
		t.Error("Should return false for different container")
	}
}

// TestConcurrentEventAddition tests thread safety of adding events
func TestConcurrentEventAddition(t *testing.T) {
	recording := &ActiveRecording{
		Events: make([]RecordingEvent, 0),
	}
	
	done := make(chan bool, 10)
	
	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(idx int) {
			recording.mu.Lock()
			recording.Events = append(recording.Events, RecordingEvent{
				Time: int64(idx * 100),
				Type: "o",
				Data: "test",
			})
			recording.mu.Unlock()
			done <- true
		}(i)
	}
	
	// Wait for all
	for i := 0; i < 10; i++ {
		<-done
	}
	
	recording.mu.Lock()
	count := len(recording.Events)
	recording.mu.Unlock()
	
	if count != 10 {
		t.Errorf("Expected 10 events, got %d", count)
	}
}

// BenchmarkGenerateRecordingToken benchmarks token generation
func BenchmarkGenerateRecordingToken(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateRecordingToken()
	}
}

// BenchmarkFormatDuration benchmarks duration formatting
func BenchmarkFormatDuration(b *testing.B) {
	d := 3661 * time.Second
	for i := 0; i < b.N; i++ {
		formatDuration(d)
	}
}

// BenchmarkEventSerialization benchmarks event JSON encoding
func BenchmarkEventSerialization(b *testing.B) {
	event := RecordingEvent{
		Time: 1234567890,
		Type: "o",
		Data: "This is some terminal output that needs to be recorded",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(event)
	}
}
