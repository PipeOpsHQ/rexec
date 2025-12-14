package container

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
)

func TestCleanupService_CleanupIdleContainers(t *testing.T) {
	// Setup mock client
	mockClient := &MockDockerClient{}

	// Setup Manager
	manager := &Manager{
		client:     mockClient,
		containers: make(map[string]*ContainerInfo),
		userIndex:  make(map[string][]string),
	}

	// Setup CleanupService
	cleanupService := NewCleanupService(manager, 1*time.Hour, 5*time.Minute)

	// Add an idle guest container
	idleGuestID := "idle-guest-container"
	manager.containers[idleGuestID] = &ContainerInfo{
		ID:         idleGuestID,
		UserID:     "guest-user",
		Status:     "running",
		CreatedAt:  time.Now(),                     // Not expired
		LastUsedAt: time.Now().Add(-2 * time.Hour), // Idle for 2 hours
		Labels:     map[string]string{"rexec.tier": "guest"},
	}

	// Add an active guest container
	activeGuestID := "active-guest-container"
	manager.containers[activeGuestID] = &ContainerInfo{
		ID:         activeGuestID,
		UserID:     "guest-user-2",
		Status:     "running",
		CreatedAt:  time.Now(), // Not expired
		LastUsedAt: time.Now(), // Active
		Labels:     map[string]string{"rexec.tier": "guest"},
	}

	// Add an idle registered user container (should NOT be cleaned up)
	idleUserID := "idle-user-container"
	manager.containers[idleUserID] = &ContainerInfo{
		ID:         idleUserID,
		UserID:     "registered-user",
		Status:     "running",
		CreatedAt:  time.Now(),                     // Not expired
		LastUsedAt: time.Now().Add(-2 * time.Hour), // Idle for 2 hours
		Labels:     map[string]string{"rexec.tier": "free"},
	}

	// Mock ContainerStop
	stoppedContainers := make(map[string]bool)
	mockClient.ContainerStopFunc = func(ctx context.Context, id string, options container.StopOptions) error {
		stoppedContainers[id] = true
		return nil
	}

	// Run cleanup
	cleanupService.cleanupIdleContainers()

	// Verify idle guest was stopped
	if !stoppedContainers[idleGuestID] {
		t.Errorf("Expected idle guest container %s to be stopped", idleGuestID)
	}

	// Verify active guest was NOT stopped
	if stoppedContainers[activeGuestID] {
		t.Errorf("Expected active guest container %s NOT to be stopped", activeGuestID)
	}

	// Verify idle registered user was NOT stopped
	if stoppedContainers[idleUserID] {
		t.Errorf("Expected idle registered user container %s NOT to be stopped", idleUserID)
	}
}

func TestCleanupService_CleanupExpiredGuestContainers(t *testing.T) {
	// Setup mock client
	mockClient := &MockDockerClient{}

	// Setup Manager
	manager := &Manager{
		client:     mockClient,
		containers: make(map[string]*ContainerInfo),
		userIndex:  make(map[string][]string),
	}

	// Setup CleanupService
	cleanupService := NewCleanupService(manager, 1*time.Hour, 5*time.Minute)

	// Add an expired guest container
	expiredGuestID := "expired-guest-container"
	manager.containers[expiredGuestID] = &ContainerInfo{
		ID:        expiredGuestID,
		UserID:    "guest-user",
		Status:    "running",
		CreatedAt: time.Now().Add(-GuestMaxSessionDuration - 1*time.Hour), // Expired
		Labels:    map[string]string{"rexec.tier": "guest"},
	}
	manager.userIndex["guest-user"] = []string{expiredGuestID}

	// Add a valid guest container
	validGuestID := "valid-guest-container"
	manager.containers[validGuestID] = &ContainerInfo{
		ID:        validGuestID,
		UserID:    "guest-user-2",
		Status:    "running",
		CreatedAt: time.Now(), // Just created
		Labels:    map[string]string{"rexec.tier": "guest"},
	}

	// Mock ContainerStop
	mockClient.ContainerStopFunc = func(ctx context.Context, id string, options container.StopOptions) error {
		return nil
	}

	// Mock ContainerRemove
	removedContainers := make(map[string]bool)
	mockClient.ContainerRemoveFunc = func(ctx context.Context, id string, options container.RemoveOptions) error {
		removedContainers[id] = true
		return nil
	}

	// Run cleanup
	cleanupService.cleanupExpiredGuestContainers()

	// Verify expired guest was removed
	if !removedContainers[expiredGuestID] {
		t.Errorf("Expected expired guest container %s to be removed", expiredGuestID)
	}

	// Verify valid guest was NOT removed
	if removedContainers[validGuestID] {
		t.Errorf("Expected valid guest container %s NOT to be removed", validGuestID)
	}
}

func TestManager_GetIdleContainers(t *testing.T) {
	manager := &Manager{
		containers: make(map[string]*ContainerInfo),
	}

	// Add idle guest
	manager.containers["idle-guest-container"] = &ContainerInfo{
		ID:         "idle-guest-container",
		Status:     "running",
		LastUsedAt: time.Now().Add(-2 * time.Hour),
		Labels:     map[string]string{"rexec.tier": "guest"},
	}

	// Add active guest
	manager.containers["active-guest-container"] = &ContainerInfo{
		ID:         "active-guest-container",
		Status:     "running",
		LastUsedAt: time.Now(),
		Labels:     map[string]string{"rexec.tier": "guest"},
	}

	// Add idle registered user
	manager.containers["idle-user-container"] = &ContainerInfo{
		ID:         "idle-user-container",
		Status:     "running",
		LastUsedAt: time.Now().Add(-2 * time.Hour),
		Labels:     map[string]string{"rexec.tier": "free"},
	}

	idle := manager.GetIdleContainers(1 * time.Hour)

	if len(idle) != 1 {
		t.Errorf("Expected 1 idle container, got %d", len(idle))
	}
	if len(idle) > 0 && idle[0].ID != "idle-guest-container" {
		t.Errorf("Expected idle-guest-container, got %s", idle[0].ID)
	}
}
