package handlers

import (
	"context"
	"fmt"
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

// GuestLogin handles guest login with email (1-hour session limit)
// If a guest with the same email exists, returns their existing session
func (h *AuthHandler) GuestLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
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

	// Determine guest email - use provided email or generate one
	var guestEmail string
	var isReturningGuest bool
	var existingUser *models.User

	if req.Email != "" && strings.Contains(req.Email, "@") {
		// User provided an email - check if they're a returning guest
		// Normalize email to lowercase to avoid duplicate users
		guestEmail = strings.ToLower(strings.TrimSpace(req.Email))
		existingUser, _, _ = h.store.GetUserByEmail(ctx, guestEmail)
		if existingUser != nil && existingUser.Tier == "guest" {
			isReturningGuest = true
		}
	} else {
		// Generate a unique guest email
		guestID := uuid.New().String()[:8]
		guestEmail = "guest_" + guestID + "@guest.rexec.local"
	}

	var user *models.User

	if isReturningGuest && existingUser != nil {
		// Returning guest - use existing user
		user = existingUser
		// Update last activity
		user.UpdatedAt = time.Now()
		h.store.UpdateUser(ctx, user)
	} else {
		// New guest - create user
		user = &models.User{
			ID:        uuid.New().String(),
			Email:     guestEmail,
			Username:  username,
			Tier:      "guest",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Store guest user
		if err := h.store.CreateUser(ctx, user, ""); err != nil {
			// If email already exists for non-guest, generate a unique one
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
				guestID := uuid.New().String()[:8]
				user.Email = "guest_" + guestID + "@guest.rexec.local"
				if err := h.store.CreateUser(ctx, user, ""); err != nil {
					c.JSON(http.StatusInternalServerError, models.APIError{
						Code:    http.StatusInternalServerError,
						Message: "Failed to create guest session",
					})
					return
				}
			} else {
				c.JSON(http.StatusInternalServerError, models.APIError{
					Code:    http.StatusInternalServerError,
					Message: "Failed to create guest session",
				})
				return
			}
		}
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

	// Get container count for returning guests
	containerCount := 0
	if isReturningGuest {
		containers, _ := h.store.GetContainersByUserID(ctx, user.ID)
		if containers != nil {
			containerCount = len(containers)
		}
	}

	response := gin.H{
		"token":           token,
		"user":            user,
		"guest":           true,
		"expires_in":      int(GuestSessionDuration.Seconds()),
		"returning_guest": isReturningGuest,
		"containers":      containerCount,
	}

	if isReturningGuest {
		response["message"] = "Welcome back! Your previous session has been restored."
	} else {
		response["message"] = "Guest session active for 1 hour. Sign in with PipeOps for unlimited sessions."
	}

	c.JSON(http.StatusOK, response)
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

	// Create a state token (JWT) to store state and code verifier statelessly in a cookie
	// This prevents issues with server restarts or multiple instances
	stateToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"state":         state,
		"code_verifier": pkceChallenge.CodeVerifier,
		"exp":           time.Now().Add(15 * time.Minute).Unix(),
	})

	stateString, err := stateToken.SignedString(h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to sign state token",
		})
		return
	}

	// Set secure, HTTP-only cookie with the state token
	// Path must match the callback URL path
	isSecure := c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https"
	// Allow insecure cookies in dev mode if needed, but prefer secure
	if os.Getenv("GIN_MODE") != "release" {
		isSecure = false
	}

	// Note: SameSite=Lax is needed for the cookie to be sent on the return redirect
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("oauth_state", stateString, 900, "/api/auth", "", isSecure, true)

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
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(renderOAuthErrorPage(errorParam, errorDesc)))
		return
	}

	if code == "" || state == "" {
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(renderOAuthErrorPage("missing_params", "Missing code or state parameter")))
		return
	}

	// Retrieve state token from cookie
	cookieParam, err := c.Cookie("oauth_state")
	if err != nil {
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(renderOAuthErrorPage("invalid_cookie", "Authentication session expired or invalid cookies. Please try again.")))
		return
	}

	// Parse and validate the state token
	token, err := jwt.Parse(cookieParam, func(token *jwt.Token) (interface{}, error) {
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(renderOAuthErrorPage("invalid_token", "Invalid authentication session")))
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(renderOAuthErrorPage("token_claims", "Invalid token claims")))
		return
	}

	// Verify state matches
	storedState, ok := claims["state"].(string)
	if !ok || storedState != state {
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(renderOAuthErrorPage("state_mismatch", "Invalid state parameter")))
		return
	}

	// Get code verifier
	codeVerifier, ok := claims["code_verifier"].(string)
	if !ok || codeVerifier == "" {
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(renderOAuthErrorPage("invalid_verifier", "Invalid code verifier")))
		return
	}

	// Clear the state cookie
	c.SetCookie("oauth_state", "", -1, "/api/auth", "", false, true)

	// Exchange code for token
	tokenResp, err := h.oauthService.ExchangeCodeForToken(code, codeVerifier)
	if err != nil {
		// Log the full error for debugging (visible in server logs)
		println("OAuth Token Exchange Error: " + err.Error())
		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(renderOAuthErrorPage("token_exchange", "Failed to exchange code for token: "+err.Error())))
		return
	}

	// Get user info from PipeOps
	userInfo, err := h.oauthService.GetUserInfo(tokenResp.AccessToken)
	if err != nil {
		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(renderOAuthErrorPage("userinfo", "Failed to get user information: "+err.Error())))
		return
	}

	ctx := context.Background()

	// Normalize email to lowercase to avoid duplicate users
	normalizedEmail := strings.ToLower(strings.TrimSpace(userInfo.Email))

	// Check if user exists
	user, _, err := h.store.GetUserByEmail(ctx, normalizedEmail)
	if err != nil {
		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(renderOAuthErrorPage("database", "Database error")))
		return
	}

	if user == nil {
		// Create new user
		username := userInfo.Username
		if username == "" {
			username = userInfo.Name
		}
		if username == "" {
			username = normalizedEmail
		}

		user = &models.User{
			ID:                 uuid.New().String(),
			Email:              normalizedEmail,
			Username:           username,
			FirstName:          userInfo.FirstName,
			LastName:           userInfo.LastName,
			Avatar:             userInfo.Avatar,
			Verified:           userInfo.Verified,
			SubscriptionActive: userInfo.SubscriptionActive,
			Tier:               "free",
			PipeOpsID:          fmt.Sprintf("%d", userInfo.ID),
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		// Store user with empty password (OAuth user)
		if err := h.store.CreateUser(ctx, user, ""); err != nil {
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(renderOAuthErrorPage("create_user", "Failed to create user")))
			return
		}
	} else {
		// Update user info with latest from PipeOps
		user.FirstName = userInfo.FirstName
		user.LastName = userInfo.LastName
		user.Avatar = userInfo.Avatar
		user.Verified = userInfo.Verified
		user.SubscriptionActive = userInfo.SubscriptionActive

		// Ensure OAuth users are at least on the free tier (upgrade from guest)
		if user.Tier == "guest" {
			user.Tier = "free"
		}
		
		// Update PipeOps ID if not set
		if user.PipeOpsID == "" {
			user.PipeOpsID = fmt.Sprintf("%d", userInfo.ID)
		}
		
		user.UpdatedAt = time.Now()
		h.store.UpdateUser(ctx, user)
	}

	// Generate JWT token
	authToken, err := h.generateToken(user)
	if err != nil {
		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(renderOAuthErrorPage("token", "Failed to generate token")))
		return
	}

	// Render success page that posts token to parent window
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(renderOAuthSuccessPage(authToken, user)))
}

// OAuthExchange handles token exchange for frontend (alternative to callback)
func (h *AuthHandler) OAuthExchange(c *gin.Context) {
	// Not used in standard flow, but keeping for compatibility if frontend uses direct exchange
	// For cookie-based flow, this endpoint is less relevant unless we pass the cookie manually,
	// but standard OAuth usually uses the Callback endpoint above.
	// Leaving as is or deprecating.
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Use callback endpoint"})
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

	// Determine display name
	name := user.Username
	if user.FirstName != "" || user.LastName != "" {
		name = strings.TrimSpace(user.FirstName + " " + user.LastName)
	}

	// Build user response
	userResponse := gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"username":   user.Username,
		"name":       name,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"avatar":     user.Avatar,
		"verified":   user.Verified,
		"tier":       user.Tier,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}

	// For guest users, include expiration time from token
	isGuest := c.GetBool("guest")
	if isGuest || user.Tier == "guest" {
		// Use token expiration if available (set by middleware), otherwise calculate from creation
		if tokenExp, exists := c.Get("tokenExp"); exists {
			userResponse["expires_at"] = tokenExp.(int64)
		} else {
			// Fallback to calculating from user creation time
			expiresAt := user.CreatedAt.Add(GuestSessionDuration)
			userResponse["expires_at"] = expiresAt.Unix()
		}
		userResponse["is_guest"] = true
	}

	c.JSON(http.StatusOK, gin.H{
		"user": userResponse,
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

// getAppURL returns the app URL for redirects (from env or default)
func getAppURL() string {
	appURL := os.Getenv("REXEC_APP_URL")
	if appURL == "" {
		appURL = "/"
	}
	return appURL
}

// renderOAuthSuccessPage returns HTML that sends token to parent window
func renderOAuthSuccessPage(token string, user *models.User) string {
	appURL := getAppURL()

	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authentication Successful - Rexec</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif;
            background: #0a0a0a;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
            overflow: hidden;
        }
        .bg-glow {
            position: fixed;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            width: 600px;
            height: 600px;
            background: radial-gradient(circle, rgba(34, 197, 94, 0.15) 0%, transparent 70%);
            pointer-events: none;
            animation: pulse 3s ease-in-out infinite;
        }
        @keyframes pulse {
            0%, 100% { opacity: 0.5; transform: translate(-50%, -50%) scale(1); }
            50% { opacity: 0.8; transform: translate(-50%, -50%) scale(1.1); }
        }
        .container {
            position: relative;
            max-width: 440px;
            width: 100%;
            text-align: center;
            background: linear-gradient(145deg, rgba(26, 26, 26, 0.95) 0%, rgba(15, 15, 15, 0.98) 100%);
            padding: 48px 44px;
            border-radius: 20px;
            border: 1px solid rgba(34, 197, 94, 0.2);
            box-shadow: 0 25px 80px rgba(0, 0, 0, 0.6), 0 0 60px rgba(34, 197, 94, 0.1);
            backdrop-filter: blur(20px);
            animation: slideUp 0.5s ease-out;
        }
        @keyframes slideUp {
            from { opacity: 0; transform: translateY(30px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .logo {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 12px;
            margin-bottom: 36px;
        }
        .logo-icon {
            width: 44px;
            height: 44px;
            background: linear-gradient(135deg, #7c7bff 0%, #6366f1 100%);
            border-radius: 12px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 22px;
            box-shadow: 0 4px 16px rgba(124, 123, 255, 0.3);
        }
        .logo-text {
            font-size: 26px;
            font-weight: 700;
            color: #ffffff;
            letter-spacing: -0.5px;
        }
        .icon-container {
            width: 88px;
            height: 88px;
            background: linear-gradient(135deg, #22c55e 0%, #16a34a 100%);
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            margin: 0 auto 28px;
            box-shadow: 0 8px 32px rgba(34, 197, 94, 0.4), 0 0 20px rgba(34, 197, 94, 0.2);
            animation: checkPop 0.6s ease-out 0.2s both;
        }
        @keyframes checkPop {
            0% { transform: scale(0); opacity: 0; }
            50% { transform: scale(1.1); }
            100% { transform: scale(1); opacity: 1; }
        }
        .icon-container svg {
            width: 44px;
            height: 44px;
            color: white;
            animation: checkDraw 0.4s ease-out 0.5s both;
        }
        @keyframes checkDraw {
            from { stroke-dashoffset: 50; opacity: 0; }
            to { stroke-dashoffset: 0; opacity: 1; }
        }
        .icon-container svg path {
            stroke-dasharray: 50;
        }
        h1 {
            color: #ffffff;
            margin: 0 0 10px;
            font-size: 26px;
            font-weight: 600;
        }
        .username {
            background: linear-gradient(135deg, #22c55e 0%, #4ade80 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            font-weight: 700;
        }
        p {
            color: #999;
            margin: 0;
            font-size: 15px;
        }
        .success-badge {
            display: inline-flex;
            align-items: center;
            gap: 6px;
            background: rgba(34, 197, 94, 0.15);
            color: #22c55e;
            padding: 6px 14px;
            border-radius: 20px;
            font-size: 13px;
            font-weight: 500;
            margin-bottom: 20px;
            border: 1px solid rgba(34, 197, 94, 0.2);
        }
        .success-badge svg {
            width: 14px;
            height: 14px;
        }
        .redirect-section {
            margin-top: 28px;
            padding-top: 24px;
            border-top: 1px solid rgba(255, 255, 255, 0.08);
        }
        .progress-bar {
            width: 100%;
            height: 4px;
            background: #1f1f1f;
            border-radius: 2px;
            overflow: hidden;
            margin-bottom: 16px;
        }
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #22c55e, #4ade80);
            border-radius: 2px;
            animation: progressFill 2s ease-out forwards;
        }
        @keyframes progressFill {
            from { width: 0%; }
            to { width: 100%; }
        }
        .redirect-text {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 10px;
            color: #666;
            font-size: 14px;
        }
        .spinner {
            width: 18px;
            height: 18px;
            border: 2px solid #2a2a2a;
            border-top-color: #22c55e;
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
    </style>
</head>
<body>
    <div class="bg-glow"></div>
    <div class="container">
        <div class="logo">
            <div class="logo-icon">⌘</div>
            <span class="logo-text">Rexec</span>
        </div>
        <div class="icon-container">
            <svg fill="none" stroke="currentColor" stroke-width="3" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"></path>
            </svg>
        </div>
        <div class="success-badge">
            <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"></path>
            </svg>
            Verified with PipeOps
        </div>
        <h1>Welcome, <span class="username">` + user.Username + `</span>!</h1>
        <p>You've successfully signed in to Rexec</p>
        <div class="redirect-section">
            <div class="progress-bar">
                <div class="progress-fill"></div>
            </div>
            <div class="redirect-text">
                <div class="spinner"></div>
                <span>Redirecting to your dashboard...</span>
            </div>
        </div>
    </div>
    <script>
        const authData = {
            token: "` + token + `",
            user: ` + userToJSON(user) + `
        };
        const appURL = "` + appURL + `";

        // Try to communicate with opener/parent window
        if (window.opener) {
            window.opener.postMessage({ type: 'oauth_success', data: authData }, window.location.origin);
            setTimeout(() => window.close(), 2000);
        } else {
            // Store in localStorage and redirect to app URL
            localStorage.setItem('rexec_token', authData.token);
            localStorage.setItem('rexec_user', JSON.stringify(authData.user));

            // Redirect to configured app URL after animation
            setTimeout(() => {
                if (appURL.startsWith('http')) {
                    window.location.href = appURL;
                } else {
                    window.location.href = window.location.origin + appURL;
                }
            }, 2000);
        }
    </script>
</body>
</html>`
}

// renderOAuthErrorPage returns HTML for OAuth errors
func renderOAuthErrorPage(errorCode, errorDesc string) string {
	appURL := getAppURL()

	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authentication Failed - Rexec</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif;
            background: #0a0a0a;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
            overflow: hidden;
        }
        .bg-glow {
            position: fixed;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            width: 600px;
            height: 600px;
            background: radial-gradient(circle, rgba(239, 68, 68, 0.1) 0%, transparent 70%);
            pointer-events: none;
        }
        .container {
            position: relative;
            max-width: 440px;
            width: 100%;
            text-align: center;
            background: linear-gradient(145deg, rgba(26, 26, 26, 0.95) 0%, rgba(15, 15, 15, 0.98) 100%);
            padding: 48px 44px;
            border-radius: 20px;
            border: 1px solid rgba(239, 68, 68, 0.15);
            box-shadow: 0 25px 80px rgba(0, 0, 0, 0.6), 0 0 40px rgba(239, 68, 68, 0.05);
            backdrop-filter: blur(20px);
            animation: slideUp 0.5s ease-out;
        }
        @keyframes slideUp {
            from { opacity: 0; transform: translateY(30px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .logo {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 12px;
            margin-bottom: 36px;
        }
        .logo-icon {
            width: 44px;
            height: 44px;
            background: linear-gradient(135deg, #7c7bff 0%, #6366f1 100%);
            border-radius: 12px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 22px;
            box-shadow: 0 4px 16px rgba(124, 123, 255, 0.3);
        }
        .logo-text {
            font-size: 26px;
            font-weight: 700;
            color: #ffffff;
            letter-spacing: -0.5px;
        }
        .icon-container {
            width: 88px;
            height: 88px;
            background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            margin: 0 auto 28px;
            box-shadow: 0 8px 32px rgba(239, 68, 68, 0.35);
            animation: shake 0.5s ease-out 0.2s;
        }
        @keyframes shake {
            0%, 100% { transform: translateX(0); }
            20% { transform: translateX(-8px); }
            40% { transform: translateX(8px); }
            60% { transform: translateX(-6px); }
            80% { transform: translateX(6px); }
        }
        .icon-container svg {
            width: 44px;
            height: 44px;
            color: white;
        }
        h1 {
            color: #ffffff;
            margin: 0 0 14px;
            font-size: 26px;
            font-weight: 600;
        }
        p {
            color: #999;
            margin: 0 0 20px;
            font-size: 15px;
            line-height: 1.6;
        }
        .error-badge {
            display: inline-flex;
            align-items: center;
            gap: 6px;
            background: rgba(239, 68, 68, 0.1);
            color: #ef4444;
            padding: 8px 16px;
            border-radius: 8px;
            font-size: 12px;
            font-family: 'SF Mono', Monaco, 'Courier New', monospace;
            margin-bottom: 28px;
            border: 1px solid rgba(239, 68, 68, 0.15);
        }
        .error-badge svg {
            width: 14px;
            height: 14px;
        }
        .action-section {
            margin-top: 28px;
            padding-top: 24px;
            border-top: 1px solid rgba(255, 255, 255, 0.08);
        }
        .btn {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            gap: 10px;
            width: 100%;
            padding: 16px 28px;
            background: linear-gradient(135deg, #7c7bff 0%, #6366f1 100%);
            color: white;
            text-decoration: none;
            border-radius: 12px;
            font-weight: 600;
            font-size: 15px;
            transition: all 0.2s ease;
            box-shadow: 0 4px 16px rgba(124, 123, 255, 0.3);
        }
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 24px rgba(124, 123, 255, 0.4);
        }
        .btn svg {
            width: 18px;
            height: 18px;
        }
        .help-text {
            margin-top: 20px;
            font-size: 13px;
            color: #666;
        }
        .help-text a {
            color: #7c7bff;
            text-decoration: none;
        }
        .help-text a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="bg-glow"></div>
    <div class="container">
        <div class="logo">
            <div class="logo-icon">⌘</div>
            <span class="logo-text">Rexec</span>
        </div>
        <div class="icon-container">
            <svg fill="none" stroke="currentColor" stroke-width="3" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
        </div>
        <h1>Authentication Failed</h1>
        <p>` + errorDesc + `</p>
        <div class="error-badge">
            <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
            </svg>
            Error: ` + errorCode + `
        </div>
        <div class="action-section">
            <a href="` + appURL + `" class="btn">
                <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path>
                </svg>
                Return to Rexec
            </a>
            <p class="help-text">Need help? <a href="https://pipeops.io/support">Contact Support</a></p>
        </div>
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
	// Escape quotes in strings to prevent JS syntax errors
	username := strings.ReplaceAll(user.Username, "\"", "\\\"")
	email := strings.ReplaceAll(user.Email, "\"", "\\\"")
	tier := strings.ReplaceAll(user.Tier, "\"", "\\\"")
	firstName := strings.ReplaceAll(user.FirstName, "\"", "\\\"")
	lastName := strings.ReplaceAll(user.LastName, "\"", "\\\"")
	avatar := strings.ReplaceAll(user.Avatar, "\"", "\\\"")
	
	// Determine display name
	name := username
	if firstName != "" || lastName != "" {
		name = strings.TrimSpace(firstName + " " + lastName)
	}
	name = strings.ReplaceAll(name, "\"", "\\\"")
	
	// Determine is_guest based on tier
	isGuest := "false"
	if tier == "guest" {
		isGuest = "true"
	}
	
			verified := "false"
		if user.Verified {
			verified = "true"
		}
	
		// Determine subscription status
		subscriptionActive := "false"
		if user.SubscriptionActive {
			subscriptionActive = "true"
		}
	
		return `{"id":"` + user.ID + `","email":"` + email + `","username":"` + username + `","name":"` + name + `","first_name":"` + firstName + `","last_name":"` + lastName + `","avatar":"` + avatar + `","verified":` + verified + `,"subscription_active":` + subscriptionActive + `,"tier":"` + tier + `","isGuest":` + isGuest + `}`
	}
