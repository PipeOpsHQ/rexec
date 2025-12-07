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
	Name     string `json:"name" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Language string `json:"language"`
}

// SnippetResponse represents a snippet in API responses
type SnippetResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	Language  string    `json:"language"`
	CreatedAt time.Time `json:"created_at"`
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
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      req.Name,
		Content:   req.Content,
		Language:  req.Language,
		CreatedAt: time.Now(),
	}

	if snippet.Language == "" {
		snippet.Language = "bash"
	}

	if err := h.store.CreateSnippet(c.Request.Context(), snippet); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Code: http.StatusInternalServerError, Message: "failed to save snippet"})
		return
	}

	c.JSON(http.StatusCreated, SnippetResponse{
		ID:        snippet.ID,
		Name:      snippet.Name,
		Content:   snippet.Content,
		Language:  snippet.Language,
		CreatedAt: snippet.CreatedAt,
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
			ID:        sn.ID,
			Name:      sn.Name,
			Content:   sn.Content,
			Language:  sn.Language,
			CreatedAt: sn.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"snippets": response,
		"count":    len(response),
	})
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
