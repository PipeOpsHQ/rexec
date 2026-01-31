package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rexec/rexec/internal/providers"
)

// VMHandler handles VM-related HTTP requests
type VMHandler struct {
	providerRegistry *providers.Registry
	store            interface{} // Will use storage.PostgresStore when needed
}

// NewVMHandler creates a new VM handler
func NewVMHandler(registry *providers.Registry) *VMHandler {
	return &VMHandler{
		providerRegistry: registry,
	}
}

// List returns all VMs/containers for the authenticated user
// Supports filtering by provider
func (h *VMHandler) List(c *gin.Context) {
	userID := c.GetString("userID")
	providerName := c.Query("provider") // Optional filter: "docker", "firecracker", etc.

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()

	var allTerminals []*providers.TerminalInfo

	// If provider filter specified, only query that provider
	if providerName != "" {
		provider, ok := h.providerRegistry.Get(providerName)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid provider",
				"message": "Provider must be one of: docker, firecracker, agent",
			})
			return
		}

		if !provider.IsAvailable(ctx) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "provider unavailable",
				"message": "The requested provider is not available",
			})
			return
		}

		terminals, err := provider.List(ctx, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list terminals"})
			return
		}
		allTerminals = terminals
	} else {
		// List from all available providers
		availableProviders := h.providerRegistry.GetAvailable(ctx)
		for _, provider := range availableProviders {
			terminals, err := provider.List(ctx, userID)
			if err != nil {
				// Log error but continue with other providers
				continue
			}
			allTerminals = append(allTerminals, terminals...)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"terminals": allTerminals,
		"count":     len(allTerminals),
	})
}

// Create creates a new VM or container
func (h *VMHandler) Create(c *gin.Context) {
	userID := c.GetString("userID")
	tier := c.GetString("tier")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Name        string            `json:"name"`
		Image       string            `json:"image" binding:"required"`
		CustomImage string            `json:"custom_image,omitempty"`
		Role        string            `json:"role,omitempty"`
		Provider    string            `json:"provider"` // "docker" (default) or "firecracker"
		MemoryMB    int64             `json:"memory_mb,omitempty"`
		CPUShares   int64             `json:"cpu_shares,omitempty"`
		DiskMB      int64             `json:"disk_mb,omitempty"`
		UserData    string            `json:"user_data,omitempty"` // Cloud-init for VMs
		Labels      map[string]string `json:"labels,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to docker if not specified
	providerName := req.Provider
	if providerName == "" {
		providerName = "docker"
	}

	provider, ok := h.providerRegistry.Get(providerName)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid provider",
			"message": "Provider must be one of: docker, firecracker",
		})
		return
	}

	if !provider.IsAvailable(c.Request.Context()) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "provider unavailable",
			"message": "The requested provider is not available on this server",
		})
		return
	}

	// Set defaults based on tier
	if req.MemoryMB == 0 {
		req.MemoryMB = 512 // Default
	}
	if req.CPUShares == 0 {
		req.CPUShares = 500 // 0.5 CPU
	}
	if req.DiskMB == 0 {
		req.DiskMB = 2048 // 2GB
	}

	cfg := providers.CreateConfig{
		UserID:      userID,
		Name:        req.Name,
		Image:       req.Image,
		CustomImage: req.CustomImage,
		Role:        req.Role,
		MemoryMB:    req.MemoryMB,
		CPUShares:   req.CPUShares,
		DiskMB:      req.DiskMB,
		UserData:    req.UserData,
		Labels:      req.Labels,
	}

	// Add tier to labels
	if cfg.Labels == nil {
		cfg.Labels = make(map[string]string)
	}
	cfg.Labels["rexec.tier"] = tier

	terminal, err := provider.Create(c.Request.Context(), cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to create terminal",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, terminal)
}

// Get returns a specific VM/container
func (h *VMHandler) Get(c *gin.Context) {
	userID := c.GetString("userID")
	terminalID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()

	// Determine provider from ID format or try all providers
	// Format: "vm:{id}" or "{docker-id}" or "agent:{id}"
	var provider providers.Provider
	var actualID string

	if strings.HasPrefix(terminalID, "vm:") {
		actualID = strings.TrimPrefix(terminalID, "vm:")
		p, ok := h.providerRegistry.Get("firecracker")
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "firecracker provider not available"})
			return
		}
		provider = p
	} else if strings.HasPrefix(terminalID, "agent:") {
		actualID = strings.TrimPrefix(terminalID, "agent:")
		p, ok := h.providerRegistry.Get("agent")
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent provider not available"})
			return
		}
		provider = p
	} else {
		// Try docker first (most common)
		actualID = terminalID
		p, ok := h.providerRegistry.Get("docker")
		if ok && p.IsAvailable(ctx) {
			terminal, err := p.Get(ctx, actualID)
			if err == nil && terminal.UserID == userID {
				c.JSON(http.StatusOK, terminal)
				return
			}
		}
		// Try firecracker
		p, ok = h.providerRegistry.Get("firecracker")
		if ok && p.IsAvailable(ctx) {
			terminal, err := p.Get(ctx, actualID)
			if err == nil && terminal.UserID == userID {
				c.JSON(http.StatusOK, terminal)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "terminal not found"})
		return
	}

	terminal, err := provider.Get(ctx, actualID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "terminal not found"})
		return
	}

	// Verify ownership
	if terminal.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
		return
	}

	c.JSON(http.StatusOK, terminal)
}

// Delete deletes a VM/container
func (h *VMHandler) Delete(c *gin.Context) {
	userID := c.GetString("userID")
	terminalID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()

	// Determine provider and ID (same logic as Get)
	var provider providers.Provider
	var actualID string

	if strings.HasPrefix(terminalID, "vm:") {
		actualID = strings.TrimPrefix(terminalID, "vm:")
		p, ok := h.providerRegistry.Get("firecracker")
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "firecracker provider not available"})
			return
		}
		provider = p
	} else if strings.HasPrefix(terminalID, "agent:") {
		actualID = strings.TrimPrefix(terminalID, "agent:")
		p, ok := h.providerRegistry.Get("agent")
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent provider not available"})
			return
		}
		provider = p
	} else {
		// Try docker first
		actualID = terminalID
		p, ok := h.providerRegistry.Get("docker")
		if ok && p.IsAvailable(ctx) {
			terminal, err := p.Get(ctx, actualID)
			if err == nil && terminal.UserID == userID {
				if err := p.Delete(ctx, actualID); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"message": "deleted", "id": terminalID})
				return
			}
		}
		// Try firecracker
		p, ok = h.providerRegistry.Get("firecracker")
		if ok && p.IsAvailable(ctx) {
			terminal, err := p.Get(ctx, actualID)
			if err == nil && terminal.UserID == userID {
				if err := p.Delete(ctx, actualID); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"message": "deleted", "id": terminalID})
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "terminal not found"})
		return
	}

	// Verify ownership before deletion
	terminal, err := provider.Get(ctx, actualID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "terminal not found"})
		return
	}

	if terminal.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
		return
	}

	if err := provider.Delete(ctx, actualID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted", "id": terminalID})
}

// Start starts a stopped VM/container
func (h *VMHandler) Start(c *gin.Context) {
	userID := c.GetString("userID")
	terminalID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()

	// Similar provider resolution logic as Delete
	// TODO: Extract to helper function
	var provider providers.Provider
	var actualID string

	if strings.HasPrefix(terminalID, "vm:") {
		actualID = strings.TrimPrefix(terminalID, "vm:")
		p, ok := h.providerRegistry.Get("firecracker")
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "firecracker provider not available"})
			return
		}
		provider = p
	} else {
		actualID = terminalID
		p, ok := h.providerRegistry.Get("docker")
		if !ok || !p.IsAvailable(ctx) {
			c.JSON(http.StatusNotFound, gin.H{"error": "provider not available"})
			return
		}
		provider = p
	}

	// Verify ownership
	terminal, err := provider.Get(ctx, actualID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "terminal not found"})
		return
	}

	if terminal.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
		return
	}

	if err := provider.Start(ctx, actualID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "started", "id": terminalID})
}

// Stop stops a running VM/container
func (h *VMHandler) Stop(c *gin.Context) {
	userID := c.GetString("userID")
	terminalID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()

	// Similar provider resolution logic
	var provider providers.Provider
	var actualID string

	if strings.HasPrefix(terminalID, "vm:") {
		actualID = strings.TrimPrefix(terminalID, "vm:")
		p, ok := h.providerRegistry.Get("firecracker")
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "firecracker provider not available"})
			return
		}
		provider = p
	} else {
		actualID = terminalID
		p, ok := h.providerRegistry.Get("docker")
		if !ok || !p.IsAvailable(ctx) {
			c.JSON(http.StatusNotFound, gin.H{"error": "provider not available"})
			return
		}
		provider = p
	}

	// Verify ownership
	terminal, err := provider.Get(ctx, actualID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "terminal not found"})
		return
	}

	if terminal.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
		return
	}

	if err := provider.Stop(ctx, actualID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "stopped", "id": terminalID})
}

// ListProviders returns available providers
func (h *VMHandler) ListProviders(c *gin.Context) {
	ctx := c.Request.Context()
	available := h.providerRegistry.GetAvailable(ctx)

	providers := make([]gin.H, len(available))
	for i, p := range available {
		providers[i] = gin.H{
			"name":      p.Name(),
			"available": true,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"count":     len(providers),
	})
}
