package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rexec/rexec/internal/storage"
)

// TokenHandler handles API token operations
type TokenHandler struct {
	store *storage.PostgresStore
}

// NewTokenHandler creates a new token handler
func NewTokenHandler(store *storage.PostgresStore) *TokenHandler {
	return &TokenHandler{store: store}
}

// CreateTokenRequest represents a request to create a new API token
type CreateTokenRequest struct {
	Name      string   `json:"name" binding:"required"`
	Scopes    []string `json:"scopes"`
	ExpiresIn *int     `json:"expires_in"` // Days until expiration (optional)
}

// CreateToken creates a new API token
func (h *TokenHandler) CreateToken(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	// Calculate expiration if provided
	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		exp := time.Now().AddDate(0, 0, *req.ExpiresIn)
		expiresAt = &exp
	}

	// Default scopes
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"read", "write"}
	}

	token, plainToken, err := h.store.GenerateAPIToken(c.Request.Context(), userID, req.Name, scopes, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
		return
	}

	// Return the token with the plain text (only shown once!)
	c.JSON(http.StatusCreated, gin.H{
		"token":        plainToken,
		"id":           token.ID,
		"name":         token.Name,
		"token_prefix": token.TokenPrefix,
		"scopes":       token.Scopes,
		"expires_at":   token.ExpiresAt,
		"created_at":   token.CreatedAt,
		"message":      "Save this token now. You won't be able to see it again!",
	})
}

// ListTokens returns all tokens for the current user
func (h *TokenHandler) ListTokens(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tokens, err := h.store.GetAPITokensByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

// RevokeToken revokes an API token
func (h *TokenHandler) RevokeToken(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tokenID := c.Param("id")
	if tokenID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token id required"})
		return
	}

	err := h.store.RevokeAPIToken(c.Request.Context(), userID, tokenID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "token revoked"})
}

// DeleteToken permanently deletes an API token
func (h *TokenHandler) DeleteToken(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tokenID := c.Param("id")
	if tokenID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token id required"})
		return
	}

	err := h.store.DeleteAPIToken(c.Request.Context(), userID, tokenID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "token deleted"})
}

// ValidateToken validates an API token (used by middleware or CLI)
func (h *TokenHandler) ValidateToken(c *gin.Context) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
		return
	}

	apiToken, err := h.store.ValidateAPIToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Get user info
	user, err := h.store.GetUserByID(c.Request.Context(), apiToken.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":    true,
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"tier":     user.Tier,
		"scopes":   apiToken.Scopes,
	})
}
