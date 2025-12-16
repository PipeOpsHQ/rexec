package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/rexec/rexec/internal/models"
	"github.com/rexec/rexec/internal/storage"
)

// SecurityHandler handles server-enforced screen lock endpoints.
type SecurityHandler struct {
	store     *storage.PostgresStore
	jwtSecret []byte
}

// NewSecurityHandler creates a new SecurityHandler.
func NewSecurityHandler(store *storage.PostgresStore, jwtSecret []byte) *SecurityHandler {
	return &SecurityHandler{store: store, jwtSecret: jwtSecret}
}

// GetScreenLock returns the current screen lock settings for the user.
// GET /api/security
func (h *SecurityHandler) GetScreenLock(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	hash, enabled, lockAfter, lockSince, err := h.store.GetUserScreenLock(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch security settings"})
		return
	}

	// Get user to check single_session_mode
	user, err := h.store.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user settings"})
		return
	}

	resp := gin.H{
		"enabled":             enabled && hash != "",
		"lock_after_minutes":  lockAfter,
		"single_session_mode": user.SingleSessionMode,
	}
	if lockSince != nil {
		resp["lock_required_since"] = lockSince
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateSettings updates non-secret screen lock settings (currently lock_after_minutes).
// PATCH /api/security
func (h *SecurityHandler) UpdateSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		LockAfterMinutes int `json:"lock_after_minutes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.LockAfterMinutes <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lock_after_minutes must be a positive integer"})
		return
	}

	hash, enabled, _, _, err := h.store.GetUserScreenLock(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}
	if !enabled || hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "screen lock not enabled"})
		return
	}

	if err := h.store.SetScreenLockPasscode(c.Request.Context(), userID, hash, req.LockAfterMinutes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":            true,
		"lock_after_minutes": req.LockAfterMinutes,
	})
}

// SetPasscode sets or changes the screen lock passcode.
// POST /api/security/passcode
func (h *SecurityHandler) SetPasscode(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		NewPasscode     string `json:"new_passcode" binding:"required"`
		CurrentPasscode string `json:"current_passcode"`
		LockAfter       *int   `json:"lock_after_minutes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new_passcode is required"})
		return
	}

	req.NewPasscode = strings.TrimSpace(req.NewPasscode)
	if len(req.NewPasscode) < 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passcode must be at least 4 characters"})
		return
	}

	// If a passcode already exists, verify current first.
	existingHash, enabled, existingAfter, _, err := h.store.GetUserScreenLock(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify current passcode"})
		return
	}
	if enabled && existingHash != "" {
		if strings.TrimSpace(req.CurrentPasscode) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "current_passcode is required to change passcode"})
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(existingHash), []byte(req.CurrentPasscode)) != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "current passcode is incorrect"})
			return
		}
	}

	afterMinutes := existingAfter
	if req.LockAfter != nil && *req.LockAfter > 0 {
		afterMinutes = *req.LockAfter
	}
	if afterMinutes <= 0 {
		afterMinutes = 5
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPasscode), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set passcode"})
		return
	}

	if err := h.store.SetScreenLockPasscode(c.Request.Context(), userID, string(hashed), afterMinutes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save passcode"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":            true,
		"lock_after_minutes": afterMinutes,
	})
}

// RemovePasscode disables screen lock.
// DELETE /api/security/passcode
func (h *SecurityHandler) RemovePasscode(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		CurrentPasscode string `json:"current_passcode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "current_passcode is required"})
		return
	}

	existingHash, enabled, _, _, err := h.store.GetUserScreenLock(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify passcode"})
		return
	}
	if !enabled || existingHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "screen lock is not enabled"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(existingHash), []byte(req.CurrentPasscode)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "passcode is incorrect"})
		return
	}

	if err := h.store.RemoveScreenLockPasscode(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable passcode"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"enabled": false})
}

// Lock marks the account as locked from now on.
// POST /api/security/lock
func (h *SecurityHandler) Lock(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	hash, enabled, _, _, err := h.store.GetUserScreenLock(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to lock"})
		return
	}
	if !enabled || hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "screen lock not enabled"})
		return
	}

	now := time.Now()
	if err := h.store.SetLockRequiredSince(c.Request.Context(), userID, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to lock"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"locked": true, "lock_required_since": now})
}

// Unlock verifies passcode and returns a fresh JWT.
// POST /api/security/unlock
func (h *SecurityHandler) Unlock(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Passcode string `json:"passcode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passcode is required"})
		return
	}

	existingHash, enabled, _, _, err := h.store.GetUserScreenLock(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify passcode"})
		return
	}
	if !enabled || existingHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "screen lock not enabled"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(existingHash), []byte(req.Passcode)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect passcode"})
		return
	}

	user, err := h.store.GetUserByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	sessionID := c.GetString("sessionID")
	if sessionID == "" {
		sid := uuid.New().String()
		if err := h.store.CreateUserSession(c.Request.Context(), &models.UserSession{
			ID:         sid,
			UserID:     user.ID,
			IPAddress:  c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			CreatedAt:  time.Now(),
			LastSeenAt: time.Now(),
		}); err != nil {
			log.Printf("failed to create session record: %v", err)
			sid = ""
		}
		sessionID = sid
	}

	token, err := h.generateToken(user, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":                  user.ID,
			"email":               user.Email,
			"username":            user.Username,
			"tier":                user.Tier,
			"subscription_active": user.SubscriptionActive,
			"is_admin":            user.IsAdmin,
			"mfa_enabled":         user.MFAEnabled,
			"allowed_ips":         user.AllowedIPs,
		},
	})
}

func (h *SecurityHandler) generateToken(user *models.User, sessionID string) (string, error) {
	expiry := 24 * time.Hour
	isGuest := user.Tier == "guest"
	if isGuest {
		expiry = GuestSessionDuration
	}

	claims := jwt.MapClaims{
		"user_id":             user.ID,
		"email":               user.Email,
		"username":            user.Username,
		"tier":                user.Tier,
		"subscription_active": user.SubscriptionActive,
		"guest":               isGuest,
		"exp":                 time.Now().Add(expiry).Unix(),
		"iat":                 time.Now().Unix(),
	}
	if sessionID != "" {
		claims["sid"] = sessionID
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

// SetSingleSessionMode enables or disables single session mode.
// When enabled, logging in from a new device automatically revokes all other sessions.
// POST /api/security/single-session
func (h *SecurityHandler) SetSingleSessionMode(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.store.SetSingleSessionMode(c.Request.Context(), userID, req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update setting"})
		return
	}

	// If enabling single session mode, also revoke all other sessions now
	if req.Enabled {
		sessionID := c.GetString("sessionID")
		if sessionID != "" {
			if err := h.store.RevokeOtherUserSessions(c.Request.Context(), userID, sessionID, "single_session_mode enabled"); err != nil {
				log.Printf("[Security] Failed to revoke other sessions: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"single_session_mode": req.Enabled,
		"message":             "Single session mode updated successfully",
	})
}
