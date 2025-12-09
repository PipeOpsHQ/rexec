package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rexec/rexec/internal/models"
	"github.com/rexec/rexec/internal/storage"
)

// SnippetHandler handles snippet management API endpoints
type SnippetHandler struct {
	store *storage.PostgresStore
}

// NewSnippetHandler creates a new SnippetHandler
func NewSnippetHandler(store *storage.PostgresStore) *SnippetHandler {
	return &SnippetHandler{
		store: store,
	}
}

// CreateSnippetRequest represents the request to create a snippet
type CreateSnippetRequest struct {
	Name        string `json:"name" binding:"required"`
	Content     string `json:"content" binding:"required"`
	Language    string `json:"language"`
	IsPublic    bool   `json:"is_public"`
	Description string `json:"description"`
}

// UpdateSnippetRequest represents the request to update a snippet
type UpdateSnippetRequest struct {
	Name        string `json:"name"`
	Content     string `json:"content"`
	Language    string `json:"language"`
	IsPublic    *bool  `json:"is_public"`
	Description string `json:"description"`
}

// SnippetResponse represents a snippet in API responses
type SnippetResponse struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id,omitempty"`
	Username    string    `json:"username,omitempty"`
	Name        string    `json:"name"`
	Content     string    `json:"content"`
	Language    string    `json:"language"`
	IsPublic    bool      `json:"is_public"`
	UsageCount  int       `json:"usage_count"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	IsOwner     bool      `json:"is_owner,omitempty"`
}

// CreateSnippet creates a new snippet
// POST /api/snippets
func (h *SnippetHandler) CreateSnippet(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.APIError{Code: http.StatusUnauthorized, Message: "unauthorized"})
		return
	}

	var req CreateSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Code: http.StatusBadRequest, Message: "invalid request: " + err.Error()})
		return
	}

	snippet := &models.Snippet{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        req.Name,
		Content:     req.Content,
		Language:    req.Language,
		IsPublic:    req.IsPublic,
		Description: req.Description,
		CreatedAt:   time.Now(),
	}

	if snippet.Language == "" {
		snippet.Language = "bash"
	}

	if err := h.store.CreateSnippet(c.Request.Context(), snippet); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Code: http.StatusInternalServerError, Message: "failed to save snippet"})
		return
	}

	c.JSON(http.StatusCreated, SnippetResponse{
		ID:          snippet.ID,
		Name:        snippet.Name,
		Content:     snippet.Content,
		Language:    snippet.Language,
		IsPublic:    snippet.IsPublic,
		Description: snippet.Description,
		CreatedAt:   snippet.CreatedAt,
		IsOwner:     true,
	})
}

// ListSnippets lists all snippets for the authenticated user
// GET /api/snippets
func (h *SnippetHandler) ListSnippets(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.APIError{Code: http.StatusUnauthorized, Message: "unauthorized"})
		return
	}

	snippets, err := h.store.GetSnippetsByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Code: http.StatusInternalServerError, Message: "failed to fetch snippets"})
		return
	}

	response := make([]SnippetResponse, 0, len(snippets))
	for _, sn := range snippets {
		response = append(response, SnippetResponse{
			ID:          sn.ID,
			Name:        sn.Name,
			Content:     sn.Content,
			Language:    sn.Language,
			IsPublic:    sn.IsPublic,
			UsageCount:  sn.UsageCount,
			Description: sn.Description,
			CreatedAt:   sn.CreatedAt,
			IsOwner:     true,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"snippets": response,
		"count":    len(response),
	})
}

// UpdateSnippet updates a snippet
// PUT /api/snippets/:id
func (h *SnippetHandler) UpdateSnippet(c *gin.Context) {
	userID := c.GetString("userID")
	snippetID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.APIError{Code: http.StatusUnauthorized, Message: "unauthorized"})
		return
	}

	// Verify ownership
	sn, err := h.store.GetSnippetByID(c.Request.Context(), snippetID)
	if err != nil || sn == nil {
		c.JSON(http.StatusNotFound, models.APIError{Code: http.StatusNotFound, Message: "snippet not found"})
		return
	}
	if sn.UserID != userID {
		c.JSON(http.StatusForbidden, models.APIError{Code: http.StatusForbidden, Message: "access denied"})
		return
	}

	var req UpdateSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Code: http.StatusBadRequest, Message: "invalid request: " + err.Error()})
		return
	}

	// Update fields if provided
	if req.Name != "" {
		sn.Name = req.Name
	}
	if req.Content != "" {
		sn.Content = req.Content
	}
	if req.Language != "" {
		sn.Language = req.Language
	}
	if req.IsPublic != nil {
		sn.IsPublic = *req.IsPublic
	}
	if req.Description != "" {
		sn.Description = req.Description
	}

	if err := h.store.UpdateSnippet(c.Request.Context(), sn); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Code: http.StatusInternalServerError, Message: "failed to update snippet"})
		return
	}

	c.JSON(http.StatusOK, SnippetResponse{
		ID:          sn.ID,
		Name:        sn.Name,
		Content:     sn.Content,
		Language:    sn.Language,
		IsPublic:    sn.IsPublic,
		UsageCount:  sn.UsageCount,
		Description: sn.Description,
		CreatedAt:   sn.CreatedAt,
		IsOwner:     true,
	})
}

// ListPublicSnippets lists all public snippets (marketplace)
// GET /api/snippets/marketplace
func (h *SnippetHandler) ListPublicSnippets(c *gin.Context) {
	userID := c.GetString("userID") // May be empty for unauthenticated users

	language := c.Query("language")
	search := c.Query("search")
	sort := c.DefaultQuery("sort", "popular") // popular, recent, name

	snippets, err := h.store.GetPublicSnippets(c.Request.Context(), language, search, sort)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Code: http.StatusInternalServerError, Message: "failed to fetch public snippets"})
		return
	}

	response := make([]SnippetResponse, 0, len(snippets))
	for _, sn := range snippets {
		response = append(response, SnippetResponse{
			ID:          sn.ID,
			Username:    sn.Username,
			Name:        sn.Name,
			Content:     sn.Content,
			Language:    sn.Language,
			IsPublic:    true,
			UsageCount:  sn.UsageCount,
			Description: sn.Description,
			CreatedAt:   sn.CreatedAt,
			IsOwner:     sn.UserID == userID,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"snippets": response,
		"count":    len(response),
	})
}

// UseSnippet increments the usage count for a public snippet
// POST /api/snippets/:id/use
func (h *SnippetHandler) UseSnippet(c *gin.Context) {
	snippetID := c.Param("id")

	sn, err := h.store.GetSnippetByID(c.Request.Context(), snippetID)
	if err != nil || sn == nil {
		c.JSON(http.StatusNotFound, models.APIError{Code: http.StatusNotFound, Message: "snippet not found"})
		return
	}

	if !sn.IsPublic {
		// Check if user owns the snippet
		userID := c.GetString("userID")
		if sn.UserID != userID {
			c.JSON(http.StatusForbidden, models.APIError{Code: http.StatusForbidden, Message: "access denied"})
			return
		}
	}

	// Increment usage count
	if err := h.store.IncrementSnippetUsage(c.Request.Context(), snippetID); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Code: http.StatusInternalServerError, Message: "failed to update usage"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "usage recorded"})
}

// DeleteSnippet deletes a snippet
// DELETE /api/snippets/:id
func (h *SnippetHandler) DeleteSnippet(c *gin.Context) {
	userID := c.GetString("userID")
	snippetID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.APIError{Code: http.StatusUnauthorized, Message: "unauthorized"})
		return
	}

	// Verify ownership
	sn, err := h.store.GetSnippetByID(c.Request.Context(), snippetID)
	if err != nil || sn == nil {
		c.JSON(http.StatusNotFound, models.APIError{Code: http.StatusNotFound, Message: "snippet not found"})
		return
	}
	if sn.UserID != userID {
		c.JSON(http.StatusForbidden, models.APIError{Code: http.StatusForbidden, Message: "access denied"})
		return
	}

	if err := h.store.DeleteSnippet(c.Request.Context(), snippetID); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Code: http.StatusInternalServerError, Message: "failed to delete snippet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Snippet deleted"})
}
