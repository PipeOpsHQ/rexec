package container

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/rexec/rexec/internal/storage"
)

// MockContainerStore implements ContainerStore for testing
type MockContainerStore struct {
	GetAllContainersFunc      func(ctx context.Context) ([]*storage.ContainerRecord, error)
	UpdateContainerStatusFunc func(ctx context.Context, id, status string) error
	DeleteContainerFunc       func(ctx context.Context, id string) error
}

func (m *MockContainerStore) GetAllContainers(ctx context.Context) ([]*storage.ContainerRecord, error) {
	if m.GetAllContainersFunc != nil {
		return m.GetAllContainersFunc(ctx)
	}
	return []*storage.ContainerRecord{}, nil
}

func (m *MockContainerStore) UpdateContainerStatus(ctx context.Context, id, status string) error {
	if m.UpdateContainerStatusFunc != nil {
		return m.UpdateContainerStatusFunc(ctx, id, status)
	}
	return nil
}

func (m *MockContainerStore) DeleteContainer(ctx context.Context, id string) error {
	if m.DeleteContainerFunc != nil {
		return m.DeleteContainerFunc(ctx, id)
	}
	return nil
}

func TestReconcilerService_Reconcile(t *testing.T) {
	// Setup mock client
	mockClient := &MockDockerClient{}

	// Setup mock store
	mockStore := &MockContainerStore{}

	// Setup Manager
	manager := &Manager{
		client:     mockClient,
		containers: make(map[string]*ContainerInfo),
	}

	// Setup Reconciler
	reconciler := NewReconcilerService(manager, mockStore, 1*time.Hour)
	// Override dockerClient in reconciler to use mock
	reconciler.dockerClient = mockClient

	// Test Case 1: Container in DB but not in Docker (should be removed)
	t.Run("RemoveOrphanedContainer", func(t *testing.T) {
		// Mock DB returning one container
		mockStore.GetAllContainersFunc = func(ctx context.Context) ([]*storage.ContainerRecord, error) {
			return []*storage.ContainerRecord{
				{
					ID:         "db-id-1",
					DockerID:   "docker-id-1",
					Status:     "running",
					LastUsedAt: time.Now(),
				},
			}, nil
		}

		// Mock Docker returning NO containers
		mockClient.ContainerListFunc = func(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
			return []types.Container{}, nil
		}

		// Mock DeleteContainer
		deletedID := ""
		mockStore.DeleteContainerFunc = func(ctx context.Context, id string) error {
			deletedID = id
			return nil
		}

		// Run reconcile
		reconciler.reconcile()

		if deletedID != "db-id-1" {
			t.Errorf("Expected container db-id-1 to be deleted, got %s", deletedID)
		}
	})

	// Test Case 2: Container status mismatch (DB: running, Docker: exited)
	t.Run("UpdateContainerStatus", func(t *testing.T) {
		// Mock DB returning one container
		mockStore.GetAllContainersFunc = func(ctx context.Context) ([]*storage.ContainerRecord, error) {
			return []*storage.ContainerRecord{
				{
					ID:         "db-id-2",
					DockerID:   "docker-id-2",
					Status:     "running",
					LastUsedAt: time.Now(),
				},
			}, nil
		}

		// Mock Docker returning exited container
		mockClient.ContainerListFunc = func(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
			return []types.Container{
				{
					ID:    "docker-id-2",
					State: "exited",
				},
			}, nil
		}

		// Mock UpdateContainerStatus
		updatedID := ""
		updatedStatus := ""
		mockStore.UpdateContainerStatusFunc = func(ctx context.Context, id, status string) error {
			updatedID = id
			updatedStatus = status
			return nil
		}

		// Run reconcile
		reconciler.reconcile()

		if updatedID != "db-id-2" {
			t.Errorf("Expected container db-id-2 to be updated, got %s", updatedID)
		}
		if updatedStatus != "stopped" {
			t.Errorf("Expected status 'stopped', got %s", updatedStatus)
		}
	})

	// Test Case 3: Container stuck in starting state for too long
	t.Run("StuckContainer", func(t *testing.T) {
		// Mock DB returning one stuck container
		mockStore.GetAllContainersFunc = func(ctx context.Context) ([]*storage.ContainerRecord, error) {
			return []*storage.ContainerRecord{
				{
					ID:         "db-id-3",
					DockerID:   "docker-id-3-long-enough",
					Status:     "starting",
					LastUsedAt: time.Now().Add(-10 * time.Minute), // Stuck for 10 mins
				},
			}, nil
		}

		// Mock Docker returning created container (still starting)
		mockClient.ContainerListFunc = func(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
			return []types.Container{
				{
					ID:    "docker-id-3-long-enough",
					State: "created",
				},
			}, nil
		}

		// Mock ContainerStop
		stoppedID := ""
		mockClient.ContainerStopFunc = func(ctx context.Context, id string, options container.StopOptions) error {
			stoppedID = id
			return nil
		}

		// Mock UpdateContainerStatus
		updatedStatus := ""
		mockStore.UpdateContainerStatusFunc = func(ctx context.Context, id, status string) error {
			if id == "db-id-3" {
				updatedStatus = status
			}
			return nil
		}

		// Run reconcile
		reconciler.reconcile()

		if stoppedID != "docker-id-3-long-enough" {
			t.Errorf("Expected container docker-id-3-long-enough to be stopped, got %s", stoppedID)
		}
		if updatedStatus != "stopped" {
			t.Errorf("Expected status 'stopped', got %s", updatedStatus)
		}
	})
}

func TestMapDockerState(t *testing.T) {
	tests := []struct {
		dockerState string
		want        string
	}{
		{"running", "running"},
		{"paused", "paused"},
		{"exited", "stopped"},
		{"dead", "stopped"},
		{"created", "starting"},
		{"restarting", "starting"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		if got := mapDockerState(tt.dockerState); got != tt.want {
			t.Errorf("mapDockerState(%s) = %s, want %s", tt.dockerState, got, tt.want)
		}
	}
}
