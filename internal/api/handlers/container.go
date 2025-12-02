package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rexec/rexec/internal/container"
	"github.com/rexec/rexec/internal/models"
	"github.com/rexec/rexec/internal/storage"
)

const (
	// GuestMaxContainerDuration is the maximum time a guest container can exist
	GuestMaxContainerDuration = 1 * time.Hour
)

// Name generation word lists
var (
	adjectives = []string{
		"swift", "bold", "calm", "dark", "eager", "fast", "grand", "happy",
		"idle", "jolly", "keen", "light", "merry", "noble", "proud", "quick",
		"rare", "sharp", "true", "wise", "brave", "cool", "deft", "epic",
		"fair", "glad", "hale", "iron", "jade", "kind", "lean", "mild",
		"neat", "open", "pure", "rich", "safe", "tall", "vast", "warm",
	}
	nouns = []string{
		"ant", "bear", "cat", "dog", "elk", "fox", "goat", "hawk",
		"ibis", "jay", "kite", "lion", "mole", "newt", "owl", "puma",
		"quail", "ram", "seal", "tiger", "urchin", "vole", "wolf", "yak",
		"zebra", "ape", "bat", "cod", "dove", "eel", "frog", "gull",
		"hare", "iguana", "jackal", "koala", "lynx", "moth", "narwhal", "otter",
	}
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// generateContainerName creates a random container name
func generateContainerName() string {
	adj := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	num := rand.Intn(1000)
	return fmt.Sprintf("%s-%s-%d", adj, noun, num)
}

// ContainerHandler handles container-related HTTP requests
type ContainerHandler struct {
	manager   *container.Manager
	store     *storage.PostgresStore
	eventsHub *ContainerEventsHub
}

// NewContainerHandler creates a new container handler
func NewContainerHandler(manager *container.Manager, store *storage.PostgresStore) *ContainerHandler {
	return &ContainerHandler{
		manager: manager,
		store:   store,
	}
}

// SetEventsHub sets the events hub for real-time notifications
func (h *ContainerHandler) SetEventsHub(hub *ContainerEventsHub) {
	h.eventsHub = hub
}

// List returns all containers for the authenticated user
func (h *ContainerHandler) List(c *gin.Context) {
	userID := c.GetString("userID")
	tier := c.GetString("tier")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := context.Background()

	// Get resource limits for user's tier
	limits := models.TierLimits(tier)

	// Get containers from database
	records, err := h.store.GetContainersByUserID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch containers"})
		return
	}

	// Sync status with Docker and build response
	containers := make([]gin.H, 0, len(records))
	for _, record := range records {
		// Check actual Docker status
		status := record.Status
		var idleTime float64

		if info, ok := h.manager.GetContainer(record.DockerID); ok {
			status = info.Status
			idleTime = time.Since(info.LastUsedAt).Seconds()

			// Update DB if status changed
			if status != record.Status {
				h.store.UpdateContainerStatus(ctx, record.ID, status)
			}
		}

		containers = append(containers, gin.H{
			"id":           record.DockerID,
			"db_id":        record.ID,
			"user_id":      record.UserID,
			"name":         record.Name,
			"image":        record.Image,
			"status":       status,
			"created_at":   record.CreatedAt,
			"last_used_at": record.LastUsedAt,
			"idle_seconds": idleTime,
			"resources": gin.H{
				"memory_mb":  limits.MemoryMB,
				"cpu_shares": limits.CPUShares,
				"disk_mb":    limits.DiskMB,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"containers": containers,
		"count":      len(containers),
		"limit":      container.UserContainerLimit(tier),
	})
}

// Create creates a new container for the user
// Uses async creation to avoid Cloudflare timeout - returns immediately with "creating" status
func (h *ContainerHandler) Create(c *gin.Context) {
	userID := c.GetString("userID")
	tier := c.GetString("tier")
	isGuest := c.GetBool("guest")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Note: Container limits are now unified for trial/guest/free users (5 containers)
	// The standard container limit check below will apply to all tiers equally

	var req models.CreateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	// Handle custom image validation
	if req.Image == "custom" {
		if req.CustomImage == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "custom_image is required when image is 'custom'",
			})
			return
		}
		// Validate custom image format
		if !isValidDockerImage(req.CustomImage) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid custom_image format",
			})
			return
		}
	} else {
		// Validate standard image type
		if _, ok := container.SupportedImages[req.Image]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":            "unsupported image type",
				"supported_images": getImageNames(),
			})
			return
		}
	}

	// Auto-generate name if not provided
	containerName := strings.TrimSpace(req.Name)
	if containerName == "" {
		containerName = generateContainerName()
	}

	// Validate container name
	if !isValidContainerName(containerName) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid container name: must be 1-64 characters, alphanumeric and hyphens only",
		})
		return
	}

	// Check container limit - use database count to include orphaned containers
	existingContainers, err := h.store.GetContainersByUserID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check container limit"})
		return
	}
	currentCount := len(existingContainers)
	limit := container.UserContainerLimit(tier)
	if currentCount >= limit {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "container limit reached",
			"current": currentCount,
			"limit":   limit,
			"tier":    tier,
			"message": "Upgrade your plan to create more containers",
		})
		return
	}

	// Check if container name already exists for this user
	for _, record := range existingContainers {
		if record.Name == containerName {
			c.JSON(http.StatusConflict, gin.H{
				"error": "container with this name already exists",
				"name":  containerName,
			})
			return
		}
	}

	// Determine the image name for storage
	imageName := req.Image
	if req.Image == "custom" {
		imageName = "custom:" + req.CustomImage
	}

	// Create a pending record in database first (async creation)
	record := &storage.ContainerRecord{
		ID:         uuid.New().String(),
		UserID:     userID,
		Name:       containerName,
		Image:      imageName,
		Status:     "creating",
		DockerID:   "", // Will be set when container is created
		VolumeName: "rexec-" + userID + "-" + containerName,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}

	if err := h.store.CreateContainer(ctx, record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create container record: " + err.Error()})
		return
	}

	// Prepare container config
	cfg := container.ContainerConfig{
		UserID:        userID,
		ContainerName: containerName,
		ImageType:     req.Image,
		CustomImage:   req.CustomImage,
		Labels: map[string]string{
			"rexec.tier":    tier,
			"rexec.user_id": userID,
		},
	}

	// Mark guest containers with special label for cleanup
	if isGuest || tier == "guest" {
		cfg.Labels["rexec.tier"] = "guest"
		cfg.Labels["rexec.guest"] = "true"
		cfg.Labels["rexec.expires_at"] = time.Now().Add(GuestMaxContainerDuration).Format(time.RFC3339)
	}

	// Apply tier-based resource limits (with trial customization)
	limits := models.ValidateTrialResources(&req, tier)
	cfg.MemoryLimit = limits.MemoryMB * 1024 * 1024 // Convert MB to bytes
	cfg.CPULimit = limits.CPUShares * 1000          // Convert to CPU quota

	// Start async container creation (pull image + create container)
	go h.createContainerAsync(record.ID, cfg, req.Image, req.CustomImage, req.Role, isGuest || tier == "guest")

	// Return immediately with "creating" status
	response := gin.H{
		"id":         record.ID, // Use DB ID as the primary ID until Docker ID is available
		"db_id":      record.ID,
		"user_id":    userID,
		"name":       containerName,
		"image":      imageName,
		"status":     "creating",
		"created_at": record.CreatedAt,
		"async":      true,
		"message":    "Container is being created. This may take a moment if the image needs to be pulled.",
		"resources": gin.H{
			"memory_mb":  limits.MemoryMB,
			"cpu_shares": limits.CPUShares,
			"disk_mb":    limits.DiskMB,
		},
	}

	// Add guest session info
	if isGuest || tier == "guest" {
		response["guest"] = true
		response["expires_at"] = time.Now().Add(GuestMaxContainerDuration).Format(time.RFC3339)
		response["session_limit_seconds"] = int(GuestMaxContainerDuration.Seconds())
	}

	c.JSON(http.StatusAccepted, response)
}

// createContainerAsync handles the actual container creation in the background
func (h *ContainerHandler) createContainerAsync(recordID string, cfg container.ContainerConfig, imageType string, customImage string, role string, isGuest bool) {
	ctx := context.Background()
	userID := cfg.UserID
	tier := cfg.Labels["rexec.tier"]

	// Pull image if needed
	var pullErr error
	if imageType == "custom" {
		pullErr = h.manager.PullCustomImage(ctx, customImage)
	} else {
		pullErr = h.manager.PullImage(ctx, imageType)
	}

	if pullErr != nil {
		// Update record with error status
		h.store.UpdateContainerStatus(ctx, recordID, "error")
		h.store.UpdateContainerError(ctx, recordID, "failed to pull image: "+pullErr.Error())
		// Notify via WebSocket
		if h.eventsHub != nil {
			h.eventsHub.NotifyContainerUpdated(userID, gin.H{
				"id":     recordID,
				"status": "error",
				"error":  "failed to pull image: " + pullErr.Error(),
			})
		}
		return
	}

	// Create the container
	info, err := h.manager.CreateContainer(ctx, cfg)
	if err != nil {
		h.store.UpdateContainerStatus(ctx, recordID, "error")
		h.store.UpdateContainerError(ctx, recordID, "failed to create container: "+err.Error())
		// Notify via WebSocket
		if h.eventsHub != nil {
			h.eventsHub.NotifyContainerUpdated(userID, gin.H{
				"id":     recordID,
				"status": "error",
				"error":  "failed to create container: " + err.Error(),
			})
		}
		return
	}

	// Update the record with Docker container info
	h.store.UpdateContainerDockerID(ctx, recordID, info.ID)
	h.store.UpdateContainerStatus(ctx, recordID, info.Status)

	// Notify via WebSocket that container is ready
	if h.eventsHub != nil {
		// Determine the image name for response
		imageName := imageType
		if imageType == "custom" {
			imageName = "custom:" + customImage
		}
		limits := models.TierLimits(tier)
		h.eventsHub.NotifyContainerCreated(userID, gin.H{
			"id":         info.ID,
			"db_id":      recordID,
			"user_id":    userID,
			"name":       cfg.ContainerName,
			"image":      imageName,
			"status":     info.Status,
			"created_at": info.CreatedAt,
			"ip_address": info.IPAddress,
			"resources": gin.H{
				"memory_mb":  limits.MemoryMB,
				"cpu_shares": limits.CPUShares,
				"disk_mb":    limits.DiskMB,
			},
		})
	}
}

// Get returns a specific container
// Supports lookup by either Docker ID or DB ID (for async container creation)
func (h *ContainerHandler) Get(c *gin.Context) {
	userID := c.GetString("userID")
	containerID := c.Param("id") // Can be Docker ID or DB ID

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := context.Background()

	// Find container in database - check both Docker ID and DB ID
	records, err := h.store.GetContainersByUserID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch containers"})
		return
	}

	var found *storage.ContainerRecord
	for _, record := range records {
		// Match by Docker ID or DB ID
		if record.DockerID == containerID || record.ID == containerID {
			found = record
			break
		}
	}

	if found == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	// Get tier for resource limits
	tier := c.GetString("tier")
	limits := models.TierLimits(tier)

	// If Docker ID is empty, container is still being created
	if found.DockerID == "" {
		c.JSON(http.StatusOK, gin.H{
			"id":           found.ID, // Use DB ID
			"db_id":        found.ID,
			"user_id":      found.UserID,
			"name":         found.Name,
			"image":        found.Image,
			"status":       found.Status, // Will be "creating" or "error"
			"created_at":   found.CreatedAt,
			"last_used_at": found.LastUsedAt,
			"resources": gin.H{
				"memory_mb":  limits.MemoryMB,
				"cpu_shares": limits.CPUShares,
				"disk_mb":    limits.DiskMB,
			},
		})
		return
	}

	// Get live info from Docker
	info, ok := h.manager.GetContainer(found.DockerID)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"id":           found.DockerID,
			"db_id":        found.ID,
			"user_id":      found.UserID,
			"name":         found.Name,
			"image":        found.Image,
			"status":       found.Status,
			"created_at":   found.CreatedAt,
			"last_used_at": found.LastUsedAt,
			"resources": gin.H{
				"memory_mb":  limits.MemoryMB,
				"cpu_shares": limits.CPUShares,
				"disk_mb":    limits.DiskMB,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           info.ID,
		"db_id":        found.ID,
		"user_id":      info.UserID,
		"name":         found.Name,
		"image":        found.Image,
		"status":       info.Status,
		"created_at":   info.CreatedAt,
		"last_used_at": info.LastUsedAt,
		"ip_address":   info.IPAddress,
		"idle_seconds": time.Since(info.LastUsedAt).Seconds(),
		"resources": gin.H{
			"memory_mb":  limits.MemoryMB,
			"cpu_shares": limits.CPUShares,
			"disk_mb":    limits.DiskMB,
		},
	})
}

// Delete soft-deletes a container (stops it but doesn't remove)
func (h *ContainerHandler) Delete(c *gin.Context) {
	userID := c.GetString("userID")
	dockerID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := context.Background()

	// Verify ownership
	records, err := h.store.GetContainersByUserID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify container ownership"})
		return
	}

	var found *storage.ContainerRecord
	for _, record := range records {
		// Match by Docker ID or DB ID (for containers that failed during creation)
		if record.DockerID == dockerID || record.ID == dockerID {
			found = record
			break
		}
	}

	if found == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	// Stop and remove the container from Docker (only if it has a Docker ID)
	if found.DockerID != "" {
		if err := h.manager.StopContainer(ctx, found.DockerID); err != nil {
			// Log but continue - container might already be stopped
			log.Printf("Warning: failed to stop container %s: %v", found.DockerID, err)
		}

		// Remove from manager's tracking (so it doesn't count toward limits)
		if err := h.manager.RemoveContainer(ctx, found.DockerID); err != nil {
			// Log but continue - container might already be removed from Docker
			log.Printf("Warning: failed to remove container %s from Docker: %v", found.DockerID, err)
		}
	}

	// Soft delete in database (sets deleted_at timestamp)
	if err := h.store.DeleteContainer(ctx, found.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete container record"})
		return
	}

	// Notify via WebSocket - send both IDs so frontend can match
	if h.eventsHub != nil {
		// Use the ID that was passed in (could be docker ID or db ID)
		h.eventsHub.NotifyContainerDeleted(userID, dockerID)
		// Also notify with db_id if different
		if found.ID != dockerID && found.DockerID != dockerID {
			h.eventsHub.NotifyContainerDeleted(userID, found.ID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "container deleted",
		"id":      dockerID,
		"db_id":   found.ID,
		"name":    found.Name,
	})
}

// Start starts a stopped container
// If the container was removed from Docker but exists in the database,
// it will be automatically recreated with the same configuration
func (h *ContainerHandler) Start(c *gin.Context) {
	userID := c.GetString("userID")
	dockerID := c.Param("id")
	tier := c.GetString("tier")
	isGuest := c.GetBool("guest")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := context.Background()

	// Guest users cannot restart stopped containers - they expire after 1 hour
	if isGuest || tier == "guest" {
		// Check if container has exceeded guest time limit
		if info, ok := h.manager.GetContainer(dockerID); ok {
			if time.Since(info.CreatedAt) > GuestMaxContainerDuration {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "guest session expired",
					"message": "Guest containers are limited to 1 hour. Sign in with PipeOps for unlimited sessions.",
					"upgrade": true,
				})
				return
			}
		}
	}

	// Verify ownership
	records, err := h.store.GetContainersByUserID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify container ownership"})
		return
	}

	var found *storage.ContainerRecord
	for _, record := range records {
		if record.DockerID == dockerID {
			found = record
			break
		}
	}

	if found == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	// Check if container actually exists in Docker
	containerExistsInDocker := h.manager.DockerContainerExists(ctx, dockerID)

	if !containerExistsInDocker {
		// Container was removed from Docker but exists in database
		// Recreate it with the same configuration
		recreateCfg := container.RecreateContainerConfig{
			UserID:        userID,
			ContainerName: found.Name,
			Image:         found.Image,
			OldDockerID:   dockerID,
			Tier:          tier,
		}

		newInfo, err := h.manager.RecreateContainer(ctx, recreateCfg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to recreate container: " + err.Error(),
				"message": "Container was removed from server and could not be recreated",
			})
			return
		}

		// Update database with new Docker ID
		h.store.UpdateContainerDockerID(ctx, found.ID, newInfo.ID)
		h.store.UpdateContainerStatus(ctx, found.ID, "running")

		c.JSON(http.StatusOK, gin.H{
			"message":     "container recreated and started",
			"id":          newInfo.ID,
			"old_id":      dockerID,
			"name":        found.Name,
			"status":      "running",
			"recreated":   true,
			"volume_kept": true, // Volume data is preserved if it still exists
		})
		return
	}

	// Container exists in Docker, start it normally
	if err := h.manager.StartContainer(ctx, dockerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start container: " + err.Error()})
		return
	}

	// Update status in database
	h.store.UpdateContainerStatus(ctx, found.ID, "running")

	// Notify via WebSocket with full container data
	if h.eventsHub != nil {
		limits := models.TierLimits(tier)
		h.eventsHub.NotifyContainerStarted(userID, gin.H{
			"id":       dockerID,
			"db_id":    found.ID,
			"name":     found.Name,
			"image":    found.Image,
			"status":   "running",
			"resources": gin.H{
				"memory_mb":  limits.MemoryMB,
				"cpu_shares": limits.CPUShares,
				"disk_mb":    limits.DiskMB,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "container started",
		"id":      dockerID,
		"name":    found.Name,
		"status":  "running",
	})
}

// Stop stops a running container
func (h *ContainerHandler) Stop(c *gin.Context) {
	userID := c.GetString("userID")
	dockerID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := context.Background()

	// Verify ownership
	records, err := h.store.GetContainersByUserID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify container ownership"})
		return
	}

	var found *storage.ContainerRecord
	for _, record := range records {
		if record.DockerID == dockerID {
			found = record
			break
		}
	}

	if found == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if err := h.manager.StopContainer(ctx, dockerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stop container: " + err.Error()})
		return
	}

	// Update status in database
	h.store.UpdateContainerStatus(ctx, found.ID, "stopped")

	// Notify via WebSocket with full container data
	tier := c.GetString("tier")
	if h.eventsHub != nil {
		limits := models.TierLimits(tier)
		h.eventsHub.NotifyContainerStopped(userID, gin.H{
			"id":       dockerID,
			"db_id":    found.ID,
			"name":     found.Name,
			"image":    found.Image,
			"status":   "stopped",
			"resources": gin.H{
				"memory_mb":  limits.MemoryMB,
				"cpu_shares": limits.CPUShares,
				"disk_mb":    limits.DiskMB,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "container stopped",
		"id":      dockerID,
		"name":    found.Name,
		"status":  "stopped",
	})
}

// CreateWithProgress creates a container with SSE progress streaming
func (h *ContainerHandler) CreateWithProgress(c *gin.Context) {
	userID := c.GetString("userID")
	tier := c.GetString("tier")
	isGuest := c.GetBool("guest")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.CreateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Quick Docker connectivity check before starting SSE stream
	// This prevents 502 errors from proxy timeouts when Docker is unavailable
	checkCtx, checkCancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, dockerErr := h.manager.GetClient().Ping(checkCtx)
	checkCancel()

	if dockerErr != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Docker daemon is not available",
			"message": "Please try again in a moment or contact support if the issue persists",
			"detail":  dockerErr.Error(),
		})
		return
	}

	// Set SSE headers - include multiple anti-buffering headers for various proxies
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")           // nginx
	c.Header("X-Content-Type-Options", "nosniff") // Prevents content sniffing
	c.Header("Transfer-Encoding", "chunked")      // Force chunked encoding

	// Flush headers immediately
	c.Writer.WriteHeader(200)
	c.Writer.Flush()

	// Helper to send SSE events with padding to bypass proxy buffering
	sendEvent := func(event container.ProgressEvent) {
		data, _ := json.Marshal(event)
		// Add padding comment to ensure minimum chunk size (some proxies buffer small chunks)
		padding := ": padding " + strings.Repeat(".", 256) + "\n"
		c.Writer.Write([]byte(padding))
		c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
		c.Writer.Flush()
	}

	ctx := context.Background()

	// Send initial comment to establish connection (helps with proxy buffering)
	c.Writer.Write([]byte(": stream connected\n\n"))
	c.Writer.Flush()

	// Stage 1: Validating
	sendEvent(container.ProgressEvent{
		Stage:    "validating",
		Message:  "Validating request...",
		Progress: 5,
	})

	// Note: Container limits now unified for all trial users (5 containers)

	// Handle custom image validation
	if req.Image == "custom" {
		if req.CustomImage == "" {
			sendEvent(container.ProgressEvent{
				Stage:    "validating",
				Error:    "custom_image is required when image is 'custom'",
				Complete: true,
			})
			return
		}
		if !isValidDockerImage(req.CustomImage) {
			sendEvent(container.ProgressEvent{
				Stage:    "validating",
				Error:    "invalid custom_image format",
				Complete: true,
			})
			return
		}
	} else {
		if _, ok := container.SupportedImages[req.Image]; !ok {
			sendEvent(container.ProgressEvent{
				Stage:    "validating",
				Error:    "unsupported image type",
				Complete: true,
			})
			return
		}
	}

	// Auto-generate name if not provided
	containerName := strings.TrimSpace(req.Name)
	if containerName == "" {
		containerName = generateContainerName()
	}

	if !isValidContainerName(containerName) {
		sendEvent(container.ProgressEvent{
			Stage:    "validating",
			Error:    "invalid container name: must be 1-64 characters, alphanumeric and hyphens only",
			Complete: true,
		})
		return
	}

	// Check container limit - use database count to include orphaned containers
	existingRecords, err := h.store.GetContainersByUserID(ctx, userID)
	if err != nil {
		sendEvent(container.ProgressEvent{
			Stage:    "validating",
			Error:    "failed to check container limit",
			Complete: true,
		})
		return
	}
	currentCount := len(existingRecords)
	limit := container.UserContainerLimit(tier)
	if currentCount >= limit {
		sendEvent(container.ProgressEvent{
			Stage:    "validating",
			Error:    fmt.Sprintf("container limit reached (%d/%d)", currentCount, limit),
			Complete: true,
		})
		return
	}

	// Check duplicate name
	for _, record := range existingRecords {
		if record.Name == containerName {
			sendEvent(container.ProgressEvent{
				Stage:    "validating",
				Error:    "container with this name already exists",
				Complete: true,
			})
			return
		}
	}

	sendEvent(container.ProgressEvent{
		Stage:    "validating",
		Message:  "Validation complete",
		Progress: 10,
	})

	// Stage 2: Check if image exists locally
	isCustom := req.Image == "custom"
	imageExists, imageName := h.manager.CheckImageExists(ctx, req.Image, isCustom, req.CustomImage)

	if imageExists {
		sendEvent(container.ProgressEvent{
			Stage:    "pulling",
			Message:  "Image already available locally",
			Progress: 100,
			Detail:   imageName,
		})
	} else {
		// Stage 2: Pull image with progress
		progressCh := make(chan container.ProgressEvent, 100)
		pullErrCh := make(chan error, 1)

		go func() {
			var err error
			if isCustom {
				err = h.manager.PullCustomImageWithProgress(ctx, req.CustomImage, progressCh)
			} else {
				err = h.manager.PullImageWithProgress(ctx, req.Image, progressCh)
			}
			pullErrCh <- err
			close(progressCh)
		}()

		// Stream pull progress
		for event := range progressCh {
			sendEvent(event)
		}

		if err := <-pullErrCh; err != nil {
			sendEvent(container.ProgressEvent{
				Stage:    "pulling",
				Error:    "Failed to pull image: " + err.Error(),
				Complete: true,
			})
			return
		}
	}

	// Stage 3: Creating container
	sendEvent(container.ProgressEvent{
		Stage:    "creating",
		Message:  "Creating container...",
		Progress: 60,
		Detail:   containerName,
	})

	// Create container config
	cfg := container.ContainerConfig{
		UserID:        userID,
		ContainerName: containerName,
		ImageType:     req.Image,
		CustomImage:   req.CustomImage,
		Labels: map[string]string{
			"rexec.tier":    tier,
			"rexec.user_id": userID,
		},
	}

	if isGuest || tier == "guest" {
		cfg.Labels["rexec.tier"] = "guest"
		cfg.Labels["rexec.guest"] = "true"
		cfg.Labels["rexec.expires_at"] = time.Now().Add(GuestMaxContainerDuration).Format(time.RFC3339)
	}

	limits := models.ValidateTrialResources(&req, tier)
	cfg.MemoryLimit = limits.MemoryMB * 1024 * 1024
	cfg.CPULimit = limits.CPUShares * 1000

	info, err := h.manager.CreateContainer(ctx, cfg)
	if err != nil {
		sendEvent(container.ProgressEvent{
			Stage:    "creating",
			Error:    "Failed to create container: " + err.Error(),
			Complete: true,
		})
		return
	}

	sendEvent(container.ProgressEvent{
		Stage:    "creating",
		Message:  "Container created",
		Progress: 80,
	})

	// Stage 4: Starting
	sendEvent(container.ProgressEvent{
		Stage:    "starting",
		Message:  "Starting container...",
		Progress: 85,
	})

	// Determine the image name for storage
	storedImageName := req.Image
	if req.Image == "custom" {
		storedImageName = "custom:" + req.CustomImage
	}

	// Store in database
	record := &storage.ContainerRecord{
		ID:         uuid.New().String(),
		UserID:     userID,
		Name:       containerName,
		Image:      storedImageName,
		Status:     info.Status,
		DockerID:   info.ID,
		VolumeName: "rexec-" + userID + "-" + containerName,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}

	if err := h.store.CreateContainer(ctx, record); err != nil {
		// Container was created but DB failed - still continue
		sendEvent(container.ProgressEvent{
			Stage:    "starting",
			Message:  "Warning: container created but database save failed",
			Progress: 90,
			Detail:   "Container is usable but may not persist after restart",
		})
	}

	sendEvent(container.ProgressEvent{
		Stage:    "starting",
		Message:  "Container started successfully",
		Progress: 90,
	})

	// Stage 5: Configuring shell (install oh-my-zsh)
	sendEvent(container.ProgressEvent{
		Stage:    "configuring",
		Message:  "Setting up enhanced shell environment...",
		Progress: 92,
		Detail:   "Installing zsh and oh-my-zsh",
	})

	// Run shell setup in background - don't block container creation if it fails
	shellResult, shellErr := container.SetupEnhancedShell(ctx, h.manager.GetClient(), info.ID)
	if shellErr != nil {
		// Log error but continue - shell setup is optional
		sendEvent(container.ProgressEvent{
			Stage:    "configuring",
			Message:  "Shell setup skipped (will use default shell)",
			Progress: 95,
			Detail:   shellErr.Error(),
		})
	} else if !shellResult.Success {
		sendEvent(container.ProgressEvent{
			Stage:    "configuring",
			Message:  "Shell setup incomplete (will use default shell)",
			Progress: 95,
			Detail:   shellResult.Message,
		})
	} else {
		sendEvent(container.ProgressEvent{
			Stage:    "configuring",
			Message:  "Enhanced shell configured successfully",
			Progress: 95,
			Detail:   "zsh with oh-my-zsh ready",
		})
	}

	// Stage 6: Setup Role (if specified)
	if req.Role != "" && req.Role != "standard" {
		sendEvent(container.ProgressEvent{
			Stage:    "configuring",
			Message:  fmt.Sprintf("Setting up %s environment...", req.Role),
			Progress: 96,
			Detail:   "Installing role-specific tools",
		})

		roleResult, roleErr := container.SetupRole(ctx, h.manager.GetClient(), info.ID, req.Role)
		if roleErr != nil {
			sendEvent(container.ProgressEvent{
				Stage:    "configuring",
				Message:  fmt.Sprintf("Role setup failed: %v", roleErr),
				Progress: 97,
				Detail:   roleErr.Error(),
			})
		} else if !roleResult.Success {
			sendEvent(container.ProgressEvent{
				Stage:    "configuring",
				Message:  fmt.Sprintf("Role setup incomplete: %s", roleResult.Message),
				Progress: 97,
				Detail:   roleResult.Output,
			})
		} else {
			sendEvent(container.ProgressEvent{
				Stage:    "configuring",
				Message:  fmt.Sprintf("Role %s configured successfully", req.Role),
				Progress: 98,
				Detail:   "Tools installed",
			})
		}
	}

	// Stage 6: Ready
	response := map[string]interface{}{
		"id":         info.ID,
		"db_id":      record.ID,
		"user_id":    info.UserID,
		"name":       containerName,
		"image":      storedImageName,
		"status":     info.Status,
		"created_at": info.CreatedAt,
		"ip_address": info.IPAddress,
		"resources": map[string]interface{}{
			"memory_mb":  limits.MemoryMB,
			"cpu_shares": limits.CPUShares,
			"disk_mb":    limits.DiskMB,
		},
	}

	if isGuest || tier == "guest" {
		response["guest"] = true
		response["expires_at"] = time.Now().Add(GuestMaxContainerDuration).Format(time.RFC3339)
		response["session_limit_seconds"] = int(GuestMaxContainerDuration.Seconds())
	}

	// Notify via WebSocket
	if h.eventsHub != nil {
		limits := models.TierLimits(tier)
		h.eventsHub.NotifyContainerCreated(userID, gin.H{
			"id":         info.ID,
			"db_id":      record.ID,
			"user_id":    info.UserID,
			"name":       containerName,
			"image":      storedImageName,
			"status":     info.Status,
			"created_at": info.CreatedAt,
			"ip_address": info.IPAddress,
			"resources": gin.H{
				"memory_mb":  limits.MemoryMB,
				"cpu_shares": limits.CPUShares,
				"disk_mb":    limits.DiskMB,
			},
		})
	}

	responseJSON, _ := json.Marshal(response)

	sendEvent(container.ProgressEvent{
		Stage:       "ready",
		Message:     "Terminal ready!",
		Progress:    100,
		Complete:    true,
		ContainerID: info.ID,
		Detail:      string(responseJSON),
	})
}

// SetupShell installs and configures enhanced shell (zsh + oh-my-zsh) in a container
func (h *ContainerHandler) SetupShell(c *gin.Context) {
	userID := c.GetString("userID")
	dockerID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Verify container exists and belongs to user
	records, err := h.store.GetContainersByUserID(context.Background(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch containers"})
		return
	}

	var found *storage.ContainerRecord
	for i := range records {
		if records[i].DockerID == dockerID {
			found = records[i]
			break
		}
	}

	if found == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	// Check if already set up
	client := h.manager.GetClient()
	ctx := context.Background()

	if container.IsShellSetupComplete(ctx, client, dockerID) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Shell already configured",
			"status":  "complete",
		})
		return
	}

	// Run shell setup
	result, err := container.SetupEnhancedShell(ctx, client, dockerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to setup shell",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": result.Success,
		"message": result.Message,
		"output":  result.Output,
	})
}

// GetShellStatus checks if enhanced shell is set up in a container
func (h *ContainerHandler) GetShellStatus(c *gin.Context) {
	userID := c.GetString("userID")
	dockerID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Verify container exists and belongs to user
	records, err := h.store.GetContainersByUserID(context.Background(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch containers"})
		return
	}

	var found *storage.ContainerRecord
	for i := range records {
		if records[i].DockerID == dockerID {
			found = records[i]
			break
		}
	}

	if found == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	client := h.manager.GetClient()
	ctx := context.Background()

	isSetup := container.IsShellSetupComplete(ctx, client, dockerID)
	shell := container.GetContainerShell(ctx, client, dockerID)

	c.JSON(http.StatusOK, gin.H{
		"setup_complete": isSetup,
		"current_shell":  shell,
		"enhanced":       isSetup && (shell == "/bin/zsh" || shell == "/usr/bin/zsh"),
	})
}

// ListImages returns available container images
func (h *ContainerHandler) ListImages(c *gin.Context) {
	showAll := c.Query("all") == "true"

	var images []container.ImageMetadata
	if showAll {
		images = container.GetImageMetadata()
	} else {
		images = container.GetPopularImages()
	}

	c.JSON(http.StatusOK, gin.H{
		"images":     images,
		"categories": container.GetImagesByCategory(),
		"popular":    container.GetPopularImages(),
	})
}

// Stats returns container statistics (admin endpoint)
func (h *ContainerHandler) Stats(c *gin.Context) {
	stats := h.manager.GetContainerStats()
	c.JSON(http.StatusOK, gin.H{
		"total":    stats.Total,
		"running":  stats.Running,
		"stopped":  stats.Stopped,
		"by_user":  stats.ByUser,
		"by_image": stats.ByImage,
	})
}

// Helper functions

// isValidContainerName validates container name format
func isValidContainerName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	for i, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
		// Can't start with hyphen or underscore
		if i == 0 && (c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// isValidDockerImage validates Docker image name format
func isValidDockerImage(image string) bool {
	if len(image) == 0 || len(image) > 256 {
		return false
	}
	// Basic validation - must contain at least one character and optionally a tag
	// Format: [registry/]image[:tag]
	parts := strings.Split(image, ":")
	if len(parts) > 2 {
		return false
	}
	imageName := parts[0]
	if len(imageName) == 0 {
		return false
	}
	// Check for invalid characters
	for _, c := range imageName {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '/') {
			return false
		}
	}
	return true
}

// getImageNames returns a list of supported image names
func getImageNames() []string {
	names := make([]string, 0, len(container.SupportedImages))
	for name := range container.SupportedImages {
		names = append(names, name)
	}
	return names
}
