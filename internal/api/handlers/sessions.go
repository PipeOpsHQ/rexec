package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rexec/rexec/internal/models"
	"github.com/rexec/rexec/internal/storage"
)

// SessionsHandler exposes endpoints to list and revoke authenticated sessions.
type SessionsHandler struct {
	store *storage.PostgresStore
}

func NewSessionsHandler(store *storage.PostgresStore) *SessionsHandler {
	return &SessionsHandler{store: store}
}

// List returns all sessions for the current user.
// GET /api/sessions
func (h *SessionsHandler) List(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessions, err := h.store.ListUserSessions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list sessions"})
		return
	}

	currentID := c.GetString("sessionID")
	resp := make([]gin.H, 0, len(sessions))
	for _, srec := range sessions {
		resp = append(resp, gin.H{
			"id":             srec.ID,
			"created_at":     srec.CreatedAt,
			"last_seen_at":   srec.LastSeenAt,
			"ip_address":     srec.IPAddress,
			"user_agent":     srec.UserAgent,
			"revoked_at":     srec.RevokedAt,
			"revoked_reason": srec.RevokedReason,
			"is_current":     currentID != "" && srec.ID == currentID,
		})
	}

	c.JSON(http.StatusOK, gin.H{"sessions": resp})
}

// Revoke revokes a specific session by ID.
// DELETE /api/sessions/:id
func (h *SessionsHandler) Revoke(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessionID := strings.TrimSpace(c.Param("id"))
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session id is required"})
		return
	}

	srec, err := h.store.GetUserSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke session"})
		return
	}
	if srec == nil || srec.UserID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "revoked_by_user"
	}

	if err := h.store.RevokeUserSession(c.Request.Context(), userID, sessionID, reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke session"})
		return
	}

	// Audit
	_ = h.store.CreateAuditLog(c.Request.Context(), &models.AuditLog{
		ID:        uuid.New().String(),
		UserID:    userID,
		Action:    "session_revoked",
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Details:   sessionID,
		CreatedAt: time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{"revoked": true})
}

// RevokeOthers revokes all other sessions except the current one.
// POST /api/sessions/revoke-others
func (h *SessionsHandler) RevokeOthers(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	currentID := c.GetString("sessionID")
	if currentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "current session not tracked"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "revoked_other_sessions"
	}

	if err := h.store.RevokeOtherUserSessions(c.Request.Context(), userID, currentID, reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke sessions"})
		return
	}

	_ = h.store.CreateAuditLog(c.Request.Context(), &models.AuditLog{
		ID:        uuid.New().String(),
		UserID:    userID,
		Action:    "sessions_revoked_others",
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		CreatedAt: time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{"revoked": true})
}
