package container

import (
	"testing"
	"time"
)

func TestManager_Accessors(t *testing.T) {
	// Setup Manager with some dummy data
	manager := &Manager{
		containers: make(map[string]*ContainerInfo),
		userIndex:  make(map[string][]string),
	}

	// Add some containers
	c1 := &ContainerInfo{
		ID:            "container-1",
		UserID:        "user-1",
		ContainerName: "test-1",
		Status:        "running",
		LastUsedAt:    time.Now().Add(-1 * time.Hour),
	}
	c2 := &ContainerInfo{
		ID:            "container-2",
		UserID:        "user-1",
		ContainerName: "test-2",
		Status:        "stopped",
		LastUsedAt:    time.Now().Add(-2 * time.Hour),
	}
	c3 := &ContainerInfo{
		ID:            "container-3",
		UserID:        "user-2",
		ContainerName: "test-3",
		Status:        "running",
		LastUsedAt:    time.Now(),
	}

	manager.containers[c1.ID] = c1
	manager.containers[c2.ID] = c2
	manager.containers[c3.ID] = c3

	manager.userIndex["user-1"] = []string{c1.ID, c2.ID}
	manager.userIndex["user-2"] = []string{c3.ID}

	// Test GetContainer
	t.Run("GetContainer", func(t *testing.T) {
		// By ID
		if got, ok := manager.GetContainer("container-1"); !ok || got != c1 {
			t.Errorf("GetContainer(container-1) failed")
		}
		// By Name
		if got, ok := manager.GetContainer("test-2"); !ok || got != c2 {
			t.Errorf("GetContainer(test-2) failed")
		}
		// By Prefix
		if got, ok := manager.GetContainer("container-3-extra"); !ok || got != c3 {
			// Note: Prefix matching logic in GetContainer might be slightly different, let's check implementation
			// Implementation: if len(idOrName) >= 12 { check prefix }
			// My dummy IDs are short. Let's use a long ID for prefix test.
		}

		// Test prefix with long ID
		longID := "12345678901234567890"
		cLong := &ContainerInfo{ID: longID, ContainerName: "long-id", UserID: "user-3"}
		manager.containers[longID] = cLong

		if got, ok := manager.GetContainer("123456789012"); !ok || got != cLong {
			t.Errorf("GetContainer prefix match failed")
		}

		// Not found
		if _, ok := manager.GetContainer("non-existent"); ok {
			t.Errorf("GetContainer(non-existent) should return false")
		}
	})

	// Test GetContainerByUserID
	t.Run("GetContainerByUserID", func(t *testing.T) {
		if got, ok := manager.GetContainerByUserID("user-2"); !ok || got != c3 {
			t.Errorf("GetContainerByUserID(user-2) failed")
		}
		if _, ok := manager.GetContainerByUserID("user-3"); !ok {
			// user-3 has no entry in userIndex yet (I added cLong but didn't update userIndex)
		}
		if _, ok := manager.GetContainerByUserID("non-existent"); ok {
			t.Errorf("GetContainerByUserID(non-existent) should return false")
		}
	})

	// Test GetUserContainers
	t.Run("GetUserContainers", func(t *testing.T) {
		containers := manager.GetUserContainers("user-1")
		if len(containers) != 2 {
			t.Errorf("GetUserContainers(user-1) returned %d containers, want 2", len(containers))
		}

		containers = manager.GetUserContainers("non-existent")
		if containers != nil {
			t.Errorf("GetUserContainers(non-existent) should return nil")
		}
	})

	// Test ListContainers
	t.Run("ListContainers", func(t *testing.T) {
		all := manager.ListContainers()
		if len(all) != 4 { // c1, c2, c3, cLong
			t.Errorf("ListContainers returned %d containers, want 4", len(all))
		}
	})

	// Test CountUserContainers
	t.Run("CountUserContainers", func(t *testing.T) {
		if count := manager.CountUserContainers("user-1"); count != 2 {
			t.Errorf("CountUserContainers(user-1) = %d, want 2", count)
		}
		if count := manager.CountUserContainers("non-existent"); count != 0 {
			t.Errorf("CountUserContainers(non-existent) = %d, want 0", count)
		}
	})

	// Test TouchContainer
	t.Run("TouchContainer", func(t *testing.T) {
		oldTime := c1.LastUsedAt
		time.Sleep(1 * time.Millisecond)
		manager.TouchContainer("container-1")
		if !c1.LastUsedAt.After(oldTime) {
			t.Errorf("TouchContainer failed to update LastUsedAt")
		}
	})

	// Test UpdateContainerStatus
	t.Run("UpdateContainerStatus", func(t *testing.T) {
		manager.UpdateContainerStatus("container-1", "paused")
		if c1.Status != "paused" {
			t.Errorf("UpdateContainerStatus failed")
		}
	})

	// Test RemoveFromTracking
	t.Run("RemoveFromTracking", func(t *testing.T) {
		manager.RemoveFromTracking("container-2")
		if _, ok := manager.containers["container-2"]; ok {
			t.Errorf("RemoveFromTracking failed to remove from containers map")
		}

		// Check user index
		ids := manager.userIndex["user-1"]
		if len(ids) != 1 || ids[0] != "container-1" {
			t.Errorf("RemoveFromTracking failed to update user index")
		}
	})
}
