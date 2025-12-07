package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rexec/rexec/internal/models"
	"github.com/rexec/rexec/internal/storage"
)

// AdminHandler handles API requests related to admin functionalities.
type AdminHandler struct {
	store *storage.PostgresStore
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(store *storage.PostgresStore) *AdminHandler {
	return &AdminHandler{store: store}
}

// ListUsers returns all users in the system.
func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, err := h.store.GetAllUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
    
    // Define a response struct that includes container count
    type AdminUserResponse struct {
        models.User
        ContainerCount int `json:"containerCount"`
    }
    
    response := make([]AdminUserResponse, len(users))
    for i, user := range users {
        // Get container count for each user (can be optimized with a single query later)
        containers, _ := h.store.GetContainersByUserID(c.Request.Context(), user.ID)
        count := 0
        if containers != nil {
            count = len(containers)
        }
        response[i] = AdminUserResponse{
            User: *user, // Dereference the pointer
            ContainerCount: count,
        }
    }

	c.JSON(http.StatusOK, response)
}

// ListContainers returns all containers in the system with owner information.
func (h *AdminHandler) ListContainers(c *gin.Context) {
	containers, err := h.store.GetAllContainersAdmin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch containers"})
		return
	}
	c.JSON(http.StatusOK, containers)
}

// ListTerminals returns all active terminal sessions.
func (h *AdminHandler) ListTerminals(c *gin.Context) {
	terminals, err := h.store.GetAllSessionsAdmin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch terminals"})
		return
	}
	c.JSON(http.StatusOK, terminals)
}

// DeleteUser deletes a user by ID.
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if err := h.store.DeleteUser(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "User deleted successfully"})
}

// DeleteContainer deletes a container by ID.
func (h *AdminHandler) DeleteContainer(c *gin.Context) {
	containerID := c.Param("id")
    // Assumes h.store.DeleteContainer method exists and handles necessary cleanup
	if err := h.store.DeleteContainer(c.Request.Context(), containerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete container"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Container deleted successfully"})
}