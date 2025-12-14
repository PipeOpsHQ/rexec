package container

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
)

func TestManager_EnsureIsolatedNetwork(t *testing.T) {
	// Setup mock client
	mockClient := &MockDockerClient{}

	// Setup Manager
	manager := &Manager{
		client: mockClient,
	}

	// Test Case 1: Network already exists
	t.Run("NetworkExists", func(t *testing.T) {
		mockClient.NetworkInspectFunc = func(ctx context.Context, networkID string, options network.InspectOptions) (network.Inspect, error) {
			if networkID == IsolatedNetworkName {
				return network.Inspect{Name: IsolatedNetworkName, ID: "existing-network-id"}, nil
			}
			return network.Inspect{}, errdefs.NotFound(fmt.Errorf("network not found"))
		}

		// We can call ensureIsolatedNetwork directly on our manager instance since we are in package container.

		err := manager.ensureIsolatedNetwork()
		if err != nil {
			t.Errorf("ensureIsolatedNetwork failed: %v", err)
		}
	})

	// Test Case 2: Network does not exist, create it
	t.Run("CreateNetwork", func(t *testing.T) {
		mockClient.NetworkInspectFunc = func(ctx context.Context, networkID string, options network.InspectOptions) (network.Inspect, error) {
			return network.Inspect{}, errdefs.NotFound(fmt.Errorf("network not found"))
		}

		created := false
		mockClient.NetworkCreateFunc = func(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
			if name != IsolatedNetworkName {
				t.Errorf("Expected network name %s, got %s", IsolatedNetworkName, name)
			}
			created = true
			return network.CreateResponse{ID: "new-network-id"}, nil
		}

		err := manager.ensureIsolatedNetwork()
		if err != nil {
			t.Errorf("ensureIsolatedNetwork failed: %v", err)
		}

		if !created {
			t.Error("Expected network to be created")
		}
	})
}

func TestManager_ExecInContainer(t *testing.T) {
	// Setup mock client
	mockClient := &MockDockerClient{}

	// Setup Manager
	manager := &Manager{
		client:     mockClient,
		containers: make(map[string]*ContainerInfo),
	}

	containerID := "test-container"
	manager.containers[containerID] = &ContainerInfo{
		ID:     containerID,
		Status: "running",
	}

	// Mock ExecCreate
	mockClient.ContainerExecCreateFunc = func(ctx context.Context, container string, config container.ExecOptions) (types.IDResponse, error) {
		if container != containerID {
			t.Errorf("Expected container ID %s, got %s", containerID, container)
		}
		return types.IDResponse{ID: "exec-id"}, nil
	}

	// Mock ExecAttach
	mockClient.ContainerExecAttachFunc = func(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error) {
		if execID != "exec-id" {
			t.Errorf("Expected exec ID exec-id, got %s", execID)
		}
		client, _ := net.Pipe()
		return types.HijackedResponse{
			Conn: client,
		}, nil
	}

	// Test ExecInContainer
	err := manager.ExecInContainer(context.Background(), containerID, []string{"ls"})
	if err != nil {
		t.Errorf("ExecInContainer failed: %v", err)
	}
}
