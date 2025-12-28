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

	"github.com/rexec/rexec/internal/auth"
	"github.com/rexec/rexec/internal/models"
	"github.com/rexec/rexec/internal/storage"
)

// SecurityHandler handles server-enforced screen lock endpoints.
type SecurityHandler struct {
	store      *storage.PostgresStore
	jwtSecret  []byte
	mfaService *auth.MFAService
}

// NewSecurityHandler creates a new SecurityHandler.
func NewSecurityHandler(store *storage.PostgresStore, jwtSecret []byte) *SecurityHandler {
	return &SecurityHandler{
		store:      store,
		jwtSecret:  jwtSecret,
		mfaService: auth.NewMFAService("Rexec"),
	}
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

// resolveContainerForMFA tries to find a container by DB ID or Docker ID.
// Returns the container record and the DB ID to use for MFA lock operations.
func (h *SecurityHandler) resolveContainerForMFA(c *gin.Context, idOrDockerID, userID string) (*storage.ContainerRecord, string, error) {
	ctx := c.Request.Context()

	// Try DB ID first
	record, err := h.store.GetContainerByID(ctx, idOrDockerID)
	if err == nil && record != nil {
		if record.UserID != userID {
			return nil, "", nil // Not owned by this user
		}
		return record, record.ID, nil
	}

	// Try Docker ID as fallback
	record, err = h.store.GetContainerByDockerID(ctx, idOrDockerID)
	if err == nil && record != nil {
		if record.UserID != userID {
			return nil, "", nil // Not owned by this user
		}
		return record, record.ID, nil
	}

	return nil, "", err
}

// LockTerminalWithMFA locks a terminal/container with MFA protection.
// Requires MFA to be enabled for the user.
// POST /api/security/terminal/:id/mfa-lock
func (h *SecurityHandler) LockTerminalWithMFA(c *gin.Context) {
	userID := c.GetString("userID")
	terminalID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if terminalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "terminal ID is required"})
		return
	}

	// Check if user has MFA enabled
	user, err := h.store.GetUserByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	if !user.MFAEnabled {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "mfa_required",
			"message": "MFA must be enabled to use terminal MFA lock. Enable MFA in your account settings first.",
		})
		return
	}

	// Handle agent terminals (id starts with "agent:")
	if strings.HasPrefix(terminalID, "agent:") {
		agentID := strings.TrimPrefix(terminalID, "agent:")
		agent, err := h.store.GetAgent(c.Request.Context(), agentID)
		if err != nil || agent == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}
		if agent.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		// Set MFA lock on the agent
		if err := h.store.SetAgentMFALock(c.Request.Context(), agentID, true); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to lock terminal"})
			return
		}

		// Log the action
		h.store.CreateAuditLog(c.Request.Context(), &models.AuditLog{
			ID:        uuid.New().String(),
			UserID:    &userID,
			Action:    "terminal_mfa_locked",
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Details:   "Agent: " + agentID,
			CreatedAt: time.Now(),
		})

		c.JSON(http.StatusOK, gin.H{
			"mfa_locked": true,
			"message":    "Agent terminal is now protected with MFA.",
		})
		return
	}

	// Regular container - try to resolve by DB ID or Docker ID
	container, dbID, err := h.resolveContainerForMFA(c, terminalID, userID)
	if err != nil || container == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	// Set MFA lock on the container using the DB ID
	if err := h.store.SetContainerMFALock(c.Request.Context(), dbID, true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to lock terminal"})
		return
	}

	// Log the action
	h.store.CreateAuditLog(c.Request.Context(), &models.AuditLog{
		ID:        uuid.New().String(),
		UserID:    &userID,
		Action:    "terminal_mfa_locked",
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Details:   "Container: " + dbID,
		CreatedAt: time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{
		"mfa_locked": true,
		"message":    "Terminal is now protected with MFA. You will need to enter your MFA code to access it.",
	})
}

// UnlockTerminalWithMFA unlocks a terminal/container by verifying MFA code.
// POST /api/security/terminal/:id/mfa-unlock
func (h *SecurityHandler) UnlockTerminalWithMFA(c *gin.Context) {
	userID := c.GetString("userID")
	terminalID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if terminalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "terminal ID is required"})
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA code is required"})
		return
	}

	// Get user's MFA secret first (needed for both agent and container cases)
	secret, err := h.store.GetUserMFASecret(c.Request.Context(), userID)
	if err != nil || secret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA not configured"})
		return
	}

	// Handle agent terminals (id starts with "agent:")
	if strings.HasPrefix(terminalID, "agent:") {
		agentID := strings.TrimPrefix(terminalID, "agent:")
		agent, err := h.store.GetAgent(c.Request.Context(), agentID)
		if err != nil || agent == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}
		if agent.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		// Verify MFA code
		if !h.mfaService.Validate(req.Code, secret) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid MFA code"})
			return
		}

		// Remove MFA lock from the agent
		if err := h.store.SetAgentMFALock(c.Request.Context(), agentID, false); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unlock terminal"})
			return
		}

		// Log the action
		h.store.CreateAuditLog(c.Request.Context(), &models.AuditLog{
			ID:        uuid.New().String(),
			UserID:    &userID,
			Action:    "terminal_mfa_unlocked",
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Details:   "Agent: " + agentID,
			CreatedAt: time.Now(),
		})

		c.JSON(http.StatusOK, gin.H{
			"mfa_locked": false,
			"message":    "Agent terminal MFA lock removed.",
		})
		return
	}

	// Regular container - try to resolve by DB ID or Docker ID
	container, dbID, err := h.resolveContainerForMFA(c, terminalID, userID)
	if err != nil || container == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if !container.MFALocked {
		c.JSON(http.StatusBadRequest, gin.H{"error": "terminal is not MFA locked"})
		return
	}

	// Verify MFA code (secret already fetched above)
	if !h.mfaService.Validate(req.Code, secret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid MFA code"})
		return
	}

	// Unlock the container
	// Remove MFA lock from the container using the DB ID
	if err := h.store.SetContainerMFALock(c.Request.Context(), dbID, false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unlock terminal"})
		return
	}

	// Log the action
	h.store.CreateAuditLog(c.Request.Context(), &models.AuditLog{
		ID:        uuid.New().String(),
		UserID:    &userID,
		Action:    "terminal_mfa_unlocked",
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Details:   "Container: " + terminalID,
		CreatedAt: time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{
		"mfa_locked": false,
		"message":    "Terminal unlocked successfully.",
	})
}

// GetTerminalMFAStatus returns the MFA lock status of a terminal.
// GET /api/security/terminal/:id/mfa-status
func (h *SecurityHandler) GetTerminalMFAStatus(c *gin.Context) {
	userID := c.GetString("userID")
	terminalID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if terminalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "terminal ID is required"})
		return
	}

	// Check if user has MFA enabled
	user, err := h.store.GetUserByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// Handle agent terminals (id starts with "agent:")
	if strings.HasPrefix(terminalID, "agent:") {
		agentID := strings.TrimPrefix(terminalID, "agent:")
		agent, err := h.store.GetAgent(c.Request.Context(), agentID)
		if err != nil || agent == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}
		if agent.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"mfa_locked":      agent.MFALocked,
			"mfa_enabled":     user.MFAEnabled,
			"can_use_feature": user.MFAEnabled,
		})
		return
	}

	// Regular container - try to resolve by DB ID or Docker ID
	container, _, err := h.resolveContainerForMFA(c, terminalID, userID)
	if err != nil || container == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"mfa_locked":      container.MFALocked,
		"mfa_enabled":     user.MFAEnabled,
		"can_use_feature": user.MFAEnabled,
	})
}

// VerifyTerminalMFAAccess verifies MFA code for accessing a locked terminal without permanently unlocking it.
// This is used when connecting to a terminal via WebSocket - validates access but keeps the lock.
// POST /api/security/terminal/:id/mfa-verify
func (h *SecurityHandler) VerifyTerminalMFAAccess(c *gin.Context) {
	userID := c.GetString("userID")
	terminalID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if terminalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "terminal ID is required"})
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA code is required"})
		return
	}

	// Get user to check MFA is enabled
	user, err := h.store.GetUserByID(c.Request.Context(), userID)
	if err != nil || user == nil || !user.MFAEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA not configured"})
		return
	}

	// Get decrypted MFA secret
	secret, err := h.store.GetUserMFASecret(c.Request.Context(), userID)
	if err != nil || secret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA not configured"})
		return
	}

	// Handle agent terminals (id starts with "agent:")
	if strings.HasPrefix(terminalID, "agent:") {
		agentID := strings.TrimPrefix(terminalID, "agent:")
		agent, err := h.store.GetAgent(c.Request.Context(), agentID)
		if err != nil || agent == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}
		if agent.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		if !agent.MFALocked {
			c.JSON(http.StatusOK, gin.H{
				"verified": true,
				"message":  "Terminal is not MFA locked.",
			})
			return
		}

		// Verify MFA code
		if !h.mfaService.Validate(req.Code, secret) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid MFA code"})
			return
		}

		// Log the access verification
		h.store.CreateAuditLog(c.Request.Context(), &models.AuditLog{
			ID:        uuid.New().String(),
			UserID:    &userID,
			Action:    "terminal_mfa_access_verified",
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Details:   "Agent: " + agentID,
			CreatedAt: time.Now(),
		})

		c.JSON(http.StatusOK, gin.H{
			"verified": true,
			"message":  "MFA verified. You may now access the terminal.",
		})
		return
	}

	// Regular container - try to resolve by DB ID or Docker ID
	container, dbID, err := h.resolveContainerForMFA(c, terminalID, userID)
	if err != nil || container == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if !container.MFALocked {
		c.JSON(http.StatusOK, gin.H{
			"verified": true,
			"message":  "Terminal is not MFA locked.",
		})
		return
	}

	// Verify MFA code
	if !h.mfaService.Validate(req.Code, secret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid MFA code"})
		return
	}

	// Log the access verification
	h.store.CreateAuditLog(c.Request.Context(), &models.AuditLog{
		ID:        uuid.New().String(),
		UserID:    &userID,
		Action:    "terminal_mfa_access_verified",
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Details:   "Container: " + dbID,
		CreatedAt: time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{
		"verified": true,
		"message":  "MFA verified. You can now access the terminal.",
	})
}
