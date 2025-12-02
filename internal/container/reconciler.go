package container

import (
	"context"
	"log"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/rexec/rexec/internal/storage"
)

// ContainerStore defines the storage interface needed by the reconciler
type ContainerStore interface {
	GetAllContainers(ctx context.Context) ([]*storage.ContainerRecord, error)
	UpdateContainerStatus(ctx context.Context, id, status string) error
	DeleteContainer(ctx context.Context, id string) error
}

// ReconcilerService syncs database state with actual Docker container state
type ReconcilerService struct {
	manager       *Manager
	store         ContainerStore
	dockerClient  *client.Client
	checkInterval time.Duration
	stopChan      chan struct{}
}

// NewReconcilerService creates a new reconciler service
func NewReconcilerService(manager *Manager, store ContainerStore, checkInterval time.Duration) *ReconcilerService {
	return &ReconcilerService{
		manager:       manager,
		store:         store,
		dockerClient:  manager.client,
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
	}
}

// Start begins the reconciler service
func (r *ReconcilerService) Start() {
	// Run once immediately on startup
	r.reconcile()

	go r.run()
	log.Printf("🔄 Reconciler service started (check interval: %v)", r.checkInterval)
}

// Stop stops the reconciler service
func (r *ReconcilerService) Stop() {
	close(r.stopChan)
	log.Println("🔄 Reconciler service stopped")
}

// run is the main loop for the reconciler service
func (r *ReconcilerService) run() {
	ticker := time.NewTicker(r.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.reconcile()
		case <-r.stopChan:
			return
		}
	}
}

// reconcile checks all containers in the database and syncs their status with Docker
func (r *ReconcilerService) reconcile() {
	ctx := context.Background()

	// Get all containers from database
	dbContainers, err := r.store.GetAllContainers(ctx)
	if err != nil {
		log.Printf("🔄 Reconciler: failed to get containers from database: %v", err)
		return
	}

	if len(dbContainers) == 0 {
		return
	}

	// Get all Docker containers (including stopped ones)
	dockerContainers, err := r.dockerClient.ContainerList(ctx, container.ListOptions{
		All: true,
	})
	if err != nil {
		log.Printf("🔄 Reconciler: failed to list Docker containers: %v", err)
		return
	}

	// Build a map of Docker container IDs to their state
	dockerState := make(map[string]string)
	for _, dc := range dockerContainers {
		dockerState[dc.ID] = dc.State
	}

	var reconciled, removed, updated int

	for _, dbContainer := range dbContainers {
		if dbContainer.DockerID == "" {
			// Container was never created in Docker (failed during creation)
			// Mark as error and soft-delete
			if dbContainer.Status != "error" && dbContainer.Status != "deleted" {
				if err := r.store.UpdateContainerStatus(ctx, dbContainer.ID, "error"); err != nil {
					log.Printf("🔄 Reconciler: failed to update status for %s: %v", dbContainer.ID, err)
				}
				// Soft delete containers that never got a Docker ID
				if err := r.store.DeleteContainer(ctx, dbContainer.ID); err != nil {
					log.Printf("🔄 Reconciler: failed to soft-delete orphaned container %s: %v", dbContainer.ID, err)
				} else {
					removed++
				}
			}
			continue
		}

		dockerStatus, exists := dockerState[dbContainer.DockerID]

		if !exists {
			// Container exists in DB but not in Docker - it was removed externally
			// Soft-delete from database and remove from manager
			if err := r.store.DeleteContainer(ctx, dbContainer.ID); err != nil {
				log.Printf("🔄 Reconciler: failed to soft-delete missing container %s: %v", dbContainer.ID, err)
			} else {
				removed++
			}
			// Remove from manager's tracking
			r.manager.RemoveFromTracking(dbContainer.DockerID)
			continue
		}

		// Map Docker state to our status
		newStatus := mapDockerState(dockerStatus)

		// Update if status differs
		if dbContainer.Status != newStatus {
			if err := r.store.UpdateContainerStatus(ctx, dbContainer.ID, newStatus); err != nil {
				log.Printf("🔄 Reconciler: failed to update status for %s: %v", dbContainer.ID, err)
			} else {
				updated++
			}

			// Also update in-memory state
			r.manager.UpdateContainerStatus(dbContainer.DockerID, newStatus)
		}

		reconciled++
	}

	if removed > 0 || updated > 0 {
		log.Printf("🔄 Reconciler: checked %d containers, removed %d orphaned, updated %d statuses",
			reconciled+removed, removed, updated)
	}
}

// mapDockerState converts Docker container state to our status string
func mapDockerState(dockerState string) string {
	switch dockerState {
	case "running":
		return "running"
	case "paused":
		return "paused"
	case "exited", "dead":
		return "stopped"
	case "created", "restarting":
		return "starting"
	default:
		return "unknown"
	}
}
