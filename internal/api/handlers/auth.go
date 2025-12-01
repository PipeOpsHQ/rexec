package handlers

import (
	"context"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rexec/rexec/internal/auth"
	"github.com/rexec/rexec/internal/container"
	"github.com/rexec/rexec/internal/models"
	"github.com/rexec/rexec/internal/storage"
)

const (
	// GuestSessionDuration is the maximum session time for guest users (1 hour)
	GuestSessionDuration = 1 * time.Hour
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	jwtSecret    []byte
	store        *storage.PostgresStore
	oauthService *auth.PKCEOAuthService
}

// OAuthState stores PKCE state for OAuth flow
type OAuthState struct {
	State        string    `json:"state"`
	CodeVerifier string    `json:"code_verifier"`
	CreatedAt    time.Time `json:"created_at"`
}

// In-memory state store (use Redis in production)
var oauthStates = make(map[string]*OAuthState)

// NewAuthHandler creates a new auth handler
func NewAuthHandler(store *storage.PostgresStore) *AuthHandler {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "rexec-dev-secret-change-in-production"
	}
	return &AuthHandler{
		jwtSecret:    []byte(secret),
		store:        store,
		oauthService: auth.NewPKCEOAuthService(),
	}
}

// GuestLogin handles guest login with just a username (1-hour session limit)
func (h *AuthHandler) GuestLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=2,max=30"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{
			Code:    http.StatusBadRequest,
			Message: "Username is required (2-30 characters)",
		})
		return
	}

	// Sanitize username - only alphanumeric, underscore, hyphen
	username := strings.TrimSpace(req.Username)
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(username) {
		c.JSON(http.StatusBadRequest, models.APIError{
			Code:    http.StatusBadRequest,
			Message: "Username can only contain letters, numbers, underscores, and hyphens",
		})
		return
	}

	ctx := context.Background()

	// Generate a unique guest email to avoid conflicts
	guestID := uuid.New().String()[:8]
	guestEmail := "guest_" + guestID + "@guest.rexec.local"

	// Create guest user
	user := &models.User{
		ID:        uuid.New().String(),
		Email:     guestEmail,
		Username:  username,
		Tier:      "guest",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store guest user
	if err := h.store.CreateUser(ctx, user, ""); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create guest session",
		})
		return
	}

	// Generate JWT token with 1-hour expiry
	token, err := h.generateGuestToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to generate token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"user":       user,
		"guest":      true,
		"expires_in": int(GuestSessionDuration.Seconds()),
		"message":    "Guest session active for 1 hour. Sign in with PipeOps for unlimited sessions.",
	})
}

// generateGuestToken creates a JWT token for a guest user with 1-hour expiry
func (h *AuthHandler) generateGuestToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"email":    user.Email,
		"username": user.Username,
		"tier":     "guest",
		"guest":    true,
		"exp":      time.Now().Add(GuestSessionDuration).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

// GetOAuthURL returns the PipeOps OAuth authorization URL
func (h *AuthHandler) GetOAuthURL(c *gin.Context) {
	// Generate PKCE challenge
	pkceChallenge, err := auth.GeneratePKCEChallenge()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to generate PKCE challenge",
		})
		return
	}

	// Generate state
	state, err := auth.GenerateRandomState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to generate state",
		})
		return
	}

	// Store state and code verifier (expires in 10 minutes)
	oauthStates[state] = &OAuthState{
		State:        state,
		CodeVerifier: pkceChallenge.CodeVerifier,
		CreatedAt:    time.Now(),
	}

	// Clean up old states
	go cleanupOldStates()

	// Get authorization URL
	authURL := h.oauthService.GetAuthorizationURL(state, pkceChallenge.CodeChallenge)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

// OAuthCallback handles the OAuth callback from PipeOps
func (h *AuthHandler) OAuthCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	// Check for error from OAuth provider
	if errorParam != "" {
		errorDesc := c.Query("error_description")
		c.HTML(http.StatusBadRequest, "", renderOAuthErrorPage(errorParam, errorDesc))
		return
	}

	if code == "" || state == "" {
		c.HTML(http.StatusBadRequest, "", renderOAuthErrorPage("missing_params", "Missing code or state parameter"))
		return
	}

	// Verify state and get code verifier
	storedState, exists := oauthStates[state]
	if !exists {
		c.HTML(http.StatusBadRequest, "", renderOAuthErrorPage("invalid_state", "Invalid or expired state parameter"))
		return
	}

	// Check if state is expired (10 minutes)
	if time.Since(storedState.CreatedAt) > 10*time.Minute {
		delete(oauthStates, state)
		c.HTML(http.StatusBadRequest, "", renderOAuthErrorPage("expired_state", "Authentication session expired. Please try again."))
		return
	}

	// Exchange code for token
	tokenResp, err := h.oauthService.ExchangeCodeForToken(code, storedState.CodeVerifier)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "", renderOAuthErrorPage("token_exchange", "Failed to exchange code for token: "+err.Error()))
		return
	}

	// Clean up used state
	delete(oauthStates, state)

	// Get user info from PipeOps
	userInfo, err := h.oauthService.GetUserInfo(tokenResp.AccessToken)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "", renderOAuthErrorPage("userinfo", "Failed to get user information: "+err.Error()))
		return
	}

	ctx := context.Background()

	// Check if user exists
	user, _, err := h.store.GetUserByEmail(ctx, userInfo.Email)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "", renderOAuthErrorPage("database", "Database error"))
		return
	}

	if user == nil {
		// Create new user
		username := userInfo.Username
		if username == "" {
			username = userInfo.Name
		}
		if username == "" {
			username = userInfo.Email
		}

		user = &models.User{
			ID:        uuid.New().String(),
			Email:     userInfo.Email,
			Username:  username,
			Tier:      "free",
			PipeOpsID: userInfo.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Store user with empty password (OAuth user)
		if err := h.store.CreateUser(ctx, user, ""); err != nil {
			c.HTML(http.StatusInternalServerError, "", renderOAuthErrorPage("create_user", "Failed to create user"))
			return
		}
	} else {
		// Update PipeOps ID if not set
		if user.PipeOpsID == "" {
			user.PipeOpsID = userInfo.ID
			user.UpdatedAt = time.Now()
			h.store.UpdateUser(ctx, user)
		}
	}

	// Generate JWT token
	token, err := h.generateToken(user)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "", renderOAuthErrorPage("token", "Failed to generate token"))
		return
	}

	// Render success page that posts token to parent window
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(renderOAuthSuccessPage(token, user)))
}

// OAuthExchange handles token exchange for frontend (alternative to callback)
func (h *AuthHandler) OAuthExchange(c *gin.Context) {
	var req struct {
		Code  string `json:"code" binding:"required"`
		State string `json:"state" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{
			Code:    http.StatusBadRequest,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// Verify state and get code verifier
	storedState, exists := oauthStates[req.State]
	if !exists {
		c.JSON(http.StatusBadRequest, models.APIError{
			Code:    http.StatusBadRequest,
			Message: "Invalid or expired state",
		})
		return
	}

	// Check if state is expired
	if time.Since(storedState.CreatedAt) > 10*time.Minute {
		delete(oauthStates, req.State)
		c.JSON(http.StatusBadRequest, models.APIError{
			Code:    http.StatusBadRequest,
			Message: "Authentication session expired",
		})
		return
	}

	// Exchange code for token
	tokenResp, err := h.oauthService.ExchangeCodeForToken(req.Code, storedState.CodeVerifier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to exchange code: " + err.Error(),
		})
		return
	}

	// Clean up used state
	delete(oauthStates, req.State)

	// Get user info
	userInfo, err := h.oauthService.GetUserInfo(tokenResp.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get user info: " + err.Error(),
		})
		return
	}

	ctx := context.Background()

	// Check if user exists or create new
	user, _, err := h.store.GetUserByEmail(ctx, userInfo.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{
			Code:    http.StatusInternalServerError,
			Message: "Database error",
		})
		return
	}

	if user == nil {
		username := userInfo.Username
		if username == "" {
			username = userInfo.Name
		}
		if username == "" {
			username = userInfo.Email
		}

		user = &models.User{
			ID:        uuid.New().String(),
			Email:     userInfo.Email,
			Username:  username,
			Tier:      "free",
			PipeOpsID: userInfo.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := h.store.CreateUser(ctx, user, ""); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIError{
				Code:    http.StatusInternalServerError,
				Message: "Failed to create user",
			})
			return
		}
	}

	// Generate JWT token
	token, err := h.generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to generate token",
		})
		return
	}

	c.JSON(http.StatusOK, models.AuthResponse{
		Token: token,
		User:  *user,
	})
}

// generateToken creates a JWT token for a user
func (h *AuthHandler) generateToken(user *models.User) (string, error) {
	// Guest users get 1-hour tokens, authenticated users get 24-hour tokens
	expiry := 24 * time.Hour
	isGuest := user.Tier == "guest"
	if isGuest {
		expiry = GuestSessionDuration
	}

	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"email":    user.Email,
		"username": user.Username,
		"tier":     user.Tier,
		"guest":    isGuest,
		"exp":      time.Now().Add(expiry).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

// GetStore returns the storage instance
func (h *AuthHandler) GetStore() *storage.PostgresStore {
	return h.store
}

// GetProfile returns the current user's profile
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := context.Background()

	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Get container count
	containers, _ := h.store.GetContainersByUserID(ctx, userID)
	containerCount := 0
	if containers != nil {
		containerCount = len(containers)
	}
	containerLimit := container.UserContainerLimit(user.Tier)

	// Get SSH key count
	sshKeys, _ := h.store.GetSSHKeysByUserID(ctx, userID)
	sshKeyCount := 0
	if sshKeys != nil {
		sshKeyCount = len(sshKeys)
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"username":   user.Username,
			"tier":       user.Tier,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
		"stats": gin.H{
			"containers":      containerCount,
			"container_limit": containerLimit,
			"ssh_keys":        sshKeyCount,
		},
		"limits": gin.H{
			"containers": containerLimit,
			"memory_mb":  models.TierLimits(user.Tier).MemoryMB,
			"cpu_shares": models.TierLimits(user.Tier).CPUShares,
			"disk_mb":    models.TierLimits(user.Tier).DiskMB,
		},
	})
}

// UpdateProfile updates the current user's profile
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if len(req.Username) < 2 || len(req.Username) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username must be between 2 and 50 characters"})
		return
	}

	ctx := context.Background()

	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	user.Username = req.Username
	user.UpdatedAt = time.Now()

	if err := h.store.UpdateUser(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	token, err := h.generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"username":   user.Username,
			"tier":       user.Tier,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
		"token": token,
	})
}

// cleanupOldStates removes expired OAuth states
func cleanupOldStates() {
	for state, data := range oauthStates {
		if time.Since(data.CreatedAt) > 15*time.Minute {
			delete(oauthStates, state)
		}
	}
}

// renderOAuthSuccessPage returns HTML that sends token to parent window
func renderOAuthSuccessPage(token string, user *models.User) string {
	return `<!DOCTYPE html>
<html>
<head>
    <title>Authentication Successful</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 40px;
            background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .container {
            max-width: 400px;
            text-align: center;
            background: rgba(255, 255, 255, 0.95);
            padding: 40px;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
        }
        .icon { font-size: 64px; margin-bottom: 20px; }
        h1 { color: #1a1a1a; margin: 0 0 10px; font-size: 24px; }
        p { color: #666; margin: 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">✓</div>
        <h1>Welcome, ` + user.Username + `!</h1>
        <p>Redirecting you to Rexec...</p>
    </div>
    <script>
        const authData = {
            token: "` + token + `",
            user: ` + userToJSON(user) + `
        };

        // Try to communicate with opener/parent window
        if (window.opener) {
            window.opener.postMessage({ type: 'oauth_success', data: authData }, window.location.origin);
            setTimeout(() => window.close(), 1000);
        } else {
            // Fallback: store in localStorage and redirect
            localStorage.setItem('rexec_token', authData.token);
            localStorage.setItem('rexec_user', JSON.stringify(authData.user));
            window.location.href = '/';
        }
    </script>
</body>
</html>`
}

// renderOAuthErrorPage returns HTML for OAuth errors
func renderOAuthErrorPage(errorCode, errorDesc string) string {
	return `<!DOCTYPE html>
<html>
<head>
    <title>Authentication Failed</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 40px;
            background: linear-gradient(135deg, #ff6b6b 0%, #ee5a5a 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .container {
            max-width: 400px;
            text-align: center;
            background: rgba(255, 255, 255, 0.95);
            padding: 40px;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
        }
        .icon { font-size: 64px; margin-bottom: 20px; }
        h1 { color: #1a1a1a; margin: 0 0 10px; font-size: 24px; }
        p { color: #666; margin: 0 0 20px; }
        .btn {
            display: inline-block;
            padding: 12px 24px;
            background: #7c7bff;
            color: white;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 500;
        }
        .btn:hover { background: #6b6aee; }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">✕</div>
        <h1>Authentication Failed</h1>
        <p>` + errorDesc + `</p>
        <a href="/" class="btn">Return to Rexec</a>
    </div>
    <script>
        if (window.opener) {
            window.opener.postMessage({ type: 'oauth_error', error: '` + errorCode + `', message: '` + errorDesc + `' }, window.location.origin);
        }
    </script>
</body>
</html>`
}

// userToJSON converts user to JSON string for embedding in HTML
func userToJSON(user *models.User) string {
	return `{"id":"` + user.ID + `","email":"` + user.Email + `","username":"` + user.Username + `","tier":"` + user.Tier + `"}`
}
