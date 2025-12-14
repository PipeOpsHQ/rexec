package container

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
)

func TestManager_StreamContainerStats(t *testing.T) {
	// Setup mock client
	mockClient := &MockDockerClient{}

	// Setup Manager
	manager := &Manager{
		client:             mockClient,
		containers:         make(map[string]*ContainerInfo),
		activeStatsStreams: make(map[string]*StatsBroadcaster),
	}

	containerID := "test-container"
	manager.containers[containerID] = &ContainerInfo{
		ID:     containerID,
		Status: "running",
	}

	// Mock ContainerStats
	// We need to return a ReadCloser that simulates stats JSON stream
	mockClient.ContainerStatsFunc = func(ctx context.Context, containerID string, stream bool) (container.StatsResponseReader, error) {
		// Create a pipe to simulate streaming
		r, w := io.Pipe()

		go func() {
			// Write one stats object
			statsJSON := `{"read":"2023-01-01T00:00:00Z","memory_stats":{"usage":1024,"limit":2048},"cpu_stats":{"cpu_usage":{"total_usage":100}},"precpu_stats":{"cpu_usage":{"total_usage":50}}}`
			w.Write([]byte(statsJSON + "\n"))
			time.Sleep(100 * time.Millisecond)
			w.Close()
		}()

		return container.StatsResponseReader{
			Body: r,
		}, nil
	}

	// Test StreamContainerStats
	// This function returns a channel that receives stats
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	statsCh := make(chan ContainerResourceStats)
	go func() {
		err := manager.StreamContainerStats(ctx, containerID, statsCh)
		if err != nil {
			t.Errorf("StreamContainerStats failed: %v", err)
		}
	}()

	// Read from channel
	select {
	case stats := <-statsCh:
		if stats.Memory != 1024 {
			t.Errorf("Expected memory usage 1024, got %f", stats.Memory)
		}
	case <-ctx.Done():
		t.Fatal("Timeout waiting for stats")
	}
}
