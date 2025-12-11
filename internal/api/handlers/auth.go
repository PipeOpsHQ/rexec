package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rexec/rexec/internal/auth"
	"github.com/rexec/rexec/internal/models"
	"github.com/rexec/rexec/internal/storage"
	admin_events "github.com/rexec/rexec/internal/api/handlers/admin_events"
)

const (
	// GuestSessionDuration is the maximum session time for guest users (1 hour)
	GuestSessionDuration = 1 * time.Hour
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	jwtSecret      []byte
	store          *storage.PostgresStore
	oauthService   *auth.PKCEOAuthService
	mfaService     *auth.MFAService
	adminEventsHub *admin_events.AdminEventsHub
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(store *storage.PostgresStore, adminEventsHub *admin_events.AdminEventsHub) *AuthHandler {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "rexec-dev-secret-change-in-production"
	}
	return &AuthHandler{
		jwtSecret:      []byte(secret),
		store:          store,
		oauthService:   auth.NewPKCEOAuthService(),
		mfaService:     auth.NewMFAService("Rexec"),
		adminEventsHub: adminEventsHub,
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
		// Broadcast user_updated event
		if h.adminEventsHub != nil {
			h.adminEventsHub.Broadcast("user_updated", user)
		}
	} else {
		// New guest - create user
		user = &models.User{
			ID:        uuid.New().String(),
			Email:     guestEmail,
			Username:  username,
			Tier:      "guest",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			IsAdmin:   false, // Explicitly set false for guests
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
		// Broadcast user_created event
		if h.adminEventsHub != nil {
			h.adminEventsHub.Broadcast("user_created", user)
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

	response := gin.H{
		"token":           token,
		"user":            user,
		"guest":           true,
		"expires_in":      int(GuestSessionDuration.Seconds()),
		"returning_guest": isReturningGuest,
		"containers":      models.GetUserResourceLimits(user.Tier, user.SubscriptionActive).MaxContainers,
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
		"user_id":             user.ID,
		"email":               user.Email,
		"username":            user.Username,
		"tier":                "guest",
		"guest":               true,
		"subscription_active": false,
		"exp":                 time.Now().Add(GuestSessionDuration).Unix(),
		"iat":                 time.Now().Unix(),
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
		log.Printf("OAuth Token Exchange Error: %v", err)
		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(renderOAuthErrorPage("token_exchange", "Failed to exchange code for token: "+err.Error())))
		return
	}

	// Get user info from PipeOps
	userInfo, err := h.oauthService.GetUserInfo(tokenResp.AccessToken)
	if err != nil {
		log.Printf("OAuth User Info Error: %v", err)
		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(renderOAuthErrorPage("userinfo", "Failed to get user information: "+err.Error())))
		return
	}

	ctx := context.Background()

	// Normalize email to lowercase to avoid duplicate users
	normalizedEmail := strings.ToLower(strings.TrimSpace(userInfo.Email))

	// Check if user exists
	user, _, err := h.store.GetUserByEmail(ctx, normalizedEmail)
	if err != nil {
		log.Printf("Database error fetching user by email for OAuth: %v", err)
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

		// Determine tier based on PipeOps subscription status
		tier := "free"
		if userInfo.SubscriptionActive {
			tier = "pro"
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
			Tier:               tier,
			PipeOpsID:          fmt.Sprintf("%d", userInfo.ID),
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
			IsAdmin:            false, // Explicitly set false for new users
		}

		// Store user with empty password (OAuth user)
		if err := h.store.CreateUser(ctx, user, ""); err != nil {
			log.Printf("Failed to create new OAuth user in DB: %v", err)
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(renderOAuthErrorPage("create_user", "Failed to create user")))
			return
		}
		// Broadcast user_created event
		if h.adminEventsHub != nil {
			h.adminEventsHub.Broadcast("user_created", user)
		}
	} else { // Existing user, update it and broadcast
		// Update user info with latest from PipeOps
		user.FirstName = userInfo.FirstName
		user.LastName = userInfo.LastName
		user.Avatar = userInfo.Avatar
		user.Verified = userInfo.Verified
		user.SubscriptionActive = userInfo.SubscriptionActive

		// Sync tier with subscription status from PipeOps
		if userInfo.SubscriptionActive {
			// Active PipeOps subscription = Pro tier
			if user.Tier != "pro" && user.Tier != "enterprise" {
				user.Tier = "pro"
			}
		} else if user.Tier == "guest" {
			// Ensure OAuth users are at least on the free tier (upgrade from guest)
			user.Tier = "free"
		}
		
		// Update PipeOps ID if not set
		if user.PipeOpsID == "" {
			user.PipeOpsID = fmt.Sprintf("%d", userInfo.ID)
		}
		
		user.UpdatedAt = time.Now()
		if err := h.store.UpdateUser(ctx, user); err != nil {
			// Log error but continue, as user exists
			log.Printf("Failed to update OAuth user %s: %v", user.ID, err)
		}
		// Broadcast user_updated event
		if h.adminEventsHub != nil {
			h.adminEventsHub.Broadcast("user_updated", user)
		}
	}

	// Check if MFA is enabled for this user
	if user.MFAEnabled {
		// Generate a temporary MFA token (short-lived, only for MFA validation)
		mfaToken, err := h.generateMFAToken(user)
		if err != nil {
			log.Printf("Failed to generate MFA token for OAuth user: %v", err)
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(renderOAuthErrorPage("token", "Failed to generate token")))
			return
		}
		// Render MFA page that asks for code
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(renderMFAPage(mfaToken, user)))
		return
	}

	// Generate JWT token
	authToken, err := h.generateToken(user)
	if err != nil {
		log.Printf("Failed to generate JWT token for OAuth user: %v", err)
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
		"user_id":             user.ID,
		"email":               user.Email,
		"username":            user.Username,
		"tier":                user.Tier,
		"subscription_active": user.SubscriptionActive,
		"guest":               isGuest,
		"exp":                 time.Now().Add(expiry).Unix(),
		"iat":                 time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

// generateMFAToken creates a short-lived token for MFA validation
func (h *AuthHandler) generateMFAToken(user *models.User) (string, error) {
	// MFA tokens are valid for 5 minutes only
	claims := jwt.MapClaims{
		"user_id":      user.ID,
		"email":        user.Email,
		"mfa_pending":  true,
		"exp":          time.Now().Add(5 * time.Minute).Unix(),
		"iat":          time.Now().Unix(),
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
		"updated_at":  user.UpdatedAt,
		"is_admin":    user.IsAdmin,
		"mfa_enabled": user.MFAEnabled,
		"allowed_ips": user.AllowedIPs,
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
			"container_count": models.GetUserResourceLimits(user.Tier, user.SubscriptionActive).MaxContainers,
			"container_limit": models.GetUserResourceLimits(user.Tier, user.SubscriptionActive).MaxContainers,
			"ssh_keys":        sshKeyCount,
		},
		"limits": gin.H{
			"containers": models.GetUserResourceLimits(user.Tier, user.SubscriptionActive).MaxContainers,
			"memory_mb":  models.GetUserResourceLimits(user.Tier, user.SubscriptionActive).MemoryMB,
			"cpu_shares": models.GetUserResourceLimits(user.Tier, user.SubscriptionActive).CPUShares,
			"disk_mb":    models.GetUserResourceLimits(user.Tier, user.SubscriptionActive).DiskMB,
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
		Username   string   `json:"username"`
		AllowedIPs []string `json:"allowed_ips"`
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
	// Only update AllowedIPs if provided (or allow clearing if empty list is sent explicitly? JSON zero value is nil/empty)
	// If the user sends [], it clears the list. If they don't send the field, it might be nil.
	// But `req.AllowedIPs` will be nil if missing.
	// Let's assume if it's not nil, update it.
	if req.AllowedIPs != nil {
		user.AllowedIPs = req.AllowedIPs
	}
	user.UpdatedAt = time.Now()

	if err := h.store.UpdateUser(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	// Broadcast user_updated event
	if h.adminEventsHub != nil {
		h.adminEventsHub.Broadcast("user_updated", user)
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
			"is_admin":   user.IsAdmin,
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
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .logo-icon svg {
            width: 44px;
            height: 44px;
            border-radius: 10px;
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
            <div class="logo-icon">
                <svg width="44" height="44" viewBox="0 0 256 256" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <rect width="256" height="256" rx="64" fill="#0A0A0A"/>
                    <g transform="translate(128, 128)">
                        <path d="M0 -80 L70 -40 L70 40 L0 80 L-70 40 L-70 -40 Z" fill="#0A0A0A" stroke="#00FF41" stroke-width="12"/>
                        <g transform="translate(-40, -40) scale(2.2)">
                            <path d="M5 10L15 20L5 30" stroke="#00FF41" stroke-width="5" stroke-linecap="round" stroke-linejoin="round"/>
                            <line x1="18" y1="30" x2="32" y2="30" stroke="#00FF41" stroke-width="5" stroke-linecap="round"/>
                        </g>
                    </g>
                </svg>
            </div>
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

// renderMFAPage returns HTML that prompts for MFA code
func renderMFAPage(mfaToken string, user *models.User) string {
	appURL := getAppURL()

	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Two-Factor Authentication - Rexec</title>
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
            background: radial-gradient(circle, rgba(139, 92, 246, 0.15) 0%, transparent 70%);
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
            border: 1px solid rgba(139, 92, 246, 0.2);
            box-shadow: 0 25px 80px rgba(0, 0, 0, 0.6), 0 0 60px rgba(139, 92, 246, 0.1);
            backdrop-filter: blur(20px);
        }
        .logo {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 12px;
            margin-bottom: 36px;
        }
        .logo-icon svg {
            width: 44px;
            height: 44px;
            border-radius: 10px;
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
            background: linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%);
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            margin: 0 auto 28px;
            color: white;
        }
        .icon-container svg {
            width: 44px;
            height: 44px;
        }
        h1 {
            font-size: 24px;
            font-weight: 600;
            color: #fff;
            margin-bottom: 12px;
        }
        p {
            font-size: 15px;
            color: #888;
            margin-bottom: 24px;
        }
        .input-group {
            margin-bottom: 20px;
        }
        input {
            width: 100%;
            padding: 14px 16px;
            font-size: 24px;
            font-family: 'JetBrains Mono', monospace;
            text-align: center;
            letter-spacing: 8px;
            background: #1a1a1a;
            border: 1px solid #333;
            border-radius: 10px;
            color: #fff;
            outline: none;
            transition: border-color 0.2s;
        }
        input:focus {
            border-color: #8b5cf6;
        }
        input::placeholder {
            letter-spacing: normal;
            font-size: 16px;
        }
        .btn {
            width: 100%;
            padding: 14px 24px;
            font-size: 16px;
            font-weight: 600;
            background: linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%);
            border: none;
            border-radius: 10px;
            color: #fff;
            cursor: pointer;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 25px rgba(139, 92, 246, 0.4);
        }
        .btn:disabled {
            opacity: 0.6;
            cursor: not-allowed;
            transform: none;
        }
        .error {
            color: #ef4444;
            font-size: 14px;
            margin-top: 12px;
            display: none;
        }
        .error.show {
            display: block;
        }
        .spinner {
            display: none;
            width: 18px;
            height: 18px;
            border: 2px solid rgba(255,255,255,0.3);
            border-top-color: #fff;
            border-radius: 50%;
            animation: spin 1s linear infinite;
            margin: 0 auto;
        }
        .spinner.show {
            display: inline-block;
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
            <div class="logo-icon">
                <svg width="44" height="44" viewBox="0 0 256 256" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <rect width="256" height="256" rx="64" fill="#0A0A0A"/>
                    <g transform="translate(128, 128)">
                        <path d="M0 -80 L70 -40 L70 40 L0 80 L-70 40 L-70 -40 Z" fill="#0A0A0A" stroke="#00FF41" stroke-width="12"/>
                        <g transform="translate(-40, -40) scale(2.2)">
                            <path d="M5 10L15 20L5 30" stroke="#00FF41" stroke-width="5" stroke-linecap="round" stroke-linejoin="round"/>
                            <line x1="18" y1="30" x2="32" y2="30" stroke="#00FF41" stroke-width="5" stroke-linecap="round"/>
                        </g>
                    </g>
                </svg>
            </div>
            <span class="logo-text">Rexec</span>
        </div>
        <div class="icon-container">
            <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path>
            </svg>
        </div>
        <h1>Two-Factor Authentication</h1>
        <p>Enter the 6-digit code from your authenticator app</p>
        <div class="input-group">
            <input type="text" id="mfa-code" maxlength="6" placeholder="000000" autocomplete="one-time-code" inputmode="numeric" pattern="[0-9]*">
        </div>
        <button class="btn" id="verify-btn" onclick="verifyMFA()">
            <span id="btn-text">Verify</span>
            <div class="spinner" id="spinner"></div>
        </button>
        <p class="error" id="error-msg"></p>
    </div>
    <script>
        const mfaToken = "` + mfaToken + `";
        const appURL = "` + appURL + `";
        const input = document.getElementById('mfa-code');
        const btn = document.getElementById('verify-btn');
        const btnText = document.getElementById('btn-text');
        const spinner = document.getElementById('spinner');
        const errorMsg = document.getElementById('error-msg');

        input.focus();

        // Auto-submit when 6 digits entered
        input.addEventListener('input', (e) => {
            e.target.value = e.target.value.replace(/\D/g, '');
            if (e.target.value.length === 6) {
                verifyMFA();
            }
        });

        input.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && input.value.length === 6) {
                verifyMFA();
            }
        });

        async function verifyMFA() {
            const code = input.value;
            if (code.length !== 6) {
                showError('Please enter a 6-digit code');
                return;
            }

            btn.disabled = true;
            btnText.style.display = 'none';
            spinner.classList.add('show');
            errorMsg.classList.remove('show');

            try {
                const res = await fetch('/api/mfa/complete-login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + mfaToken
                    },
                    body: JSON.stringify({ code: code })
                });

                const data = await res.json();

                if (!res.ok) {
                    throw new Error(data.error || 'Verification failed');
                }

                // Success - store token and redirect
                const authData = {
                    token: data.token,
                    user: data.user
                };

                if (window.opener) {
                    window.opener.postMessage({ type: 'oauth_success', data: authData }, window.location.origin);
                    setTimeout(() => window.close(), 500);
                } else {
                    localStorage.setItem('rexec_token', authData.token);
                    localStorage.setItem('rexec_user', JSON.stringify(authData.user));
                    if (appURL.startsWith('http')) {
                        window.location.href = appURL;
                    } else {
                        window.location.href = window.location.origin + appURL;
                    }
                }
            } catch (err) {
                showError(err.message);
                btn.disabled = false;
                btnText.style.display = 'inline';
                spinner.classList.remove('show');
                input.value = '';
                input.focus();
            }
        }

        function showError(msg) {
            errorMsg.textContent = msg;
            errorMsg.classList.add('show');
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
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .logo-icon svg {
            width: 44px;
            height: 44px;
            border-radius: 10px;
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
            <div class="logo-icon">
                <svg width="44" height="44" viewBox="0 0 256 256" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <rect width="256" height="256" rx="64" fill="#0A0A0A"/>
                    <g transform="translate(128, 128)">
                        <path d="M0 -80 L70 -40 L70 40 L0 80 L-70 40 L-70 -40 Z" fill="#0A0A0A" stroke="#00FF41" stroke-width="12"/>
                        <g transform="translate(-40, -40) scale(2.2)">
                            <path d="M5 10L15 20L5 30" stroke="#00FF41" stroke-width="5" stroke-linecap="round" stroke-linejoin="round"/>
                            <line x1="18" y1="30" x2="32" y2="30" stroke="#00FF41" stroke-width="5" stroke-linecap="round"/>
                        </g>
                    </g>
                </svg>
            </div>
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
	username := strings.ReplaceAll(user.Username, `"`, `\"`)
	email := strings.ReplaceAll(user.Email, `"`, `\"`)
	tier := strings.ReplaceAll(user.Tier, `"`, `\"`)
	firstName := strings.ReplaceAll(user.FirstName, `"`, `\"`)
	lastName := strings.ReplaceAll(user.LastName, `"`, `\"`)
	avatar := strings.ReplaceAll(user.Avatar, `"`, `\"`)
	
	// Determine display name
	name := username
	if firstName != "" || lastName != "" {
		name = strings.TrimSpace(firstName + " " + lastName)
	}
	name = strings.ReplaceAll(name, `"`, `\"`)
	
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

// SetupMFA initiates MFA setup for a user
func (h *AuthHandler) SetupMFA(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Generate secret
	secret, err := auth.GenerateRandomSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate secret"})
		return
	}

	userEmail := c.GetString("email")
	if userEmail == "" {
		// Fallback to fetching user if email not in context
		ctx := context.Background()
		user, err := h.store.GetUserByID(ctx, userID)
		if err == nil && user != nil {
			userEmail = user.Email
		} else {
			userEmail = "user"
		}
	}

	// Get OTP URL for QR code
	otpURL := h.mfaService.GetOTPURL(userEmail, secret)

	c.JSON(http.StatusOK, gin.H{
		"secret":  secret,
		"otp_url": otpURL,
	})
}

// VerifyMFA verifies the code and enables MFA
func (h *AuthHandler) VerifyMFA(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Secret string `json:"secret" binding:"required"`
		Code   string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate code
	if !h.mfaService.Validate(req.Code, req.Secret) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid code"})
		return
	}

	ctx := context.Background()
	
	// Enable MFA for user
	if err := h.store.EnableMFA(ctx, userID, req.Secret); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable MFA"})
		return
	}

	// Log audit event
	h.store.CreateAuditLog(ctx, &models.AuditLog{
		ID:        uuid.New().String(),
		UserID:    userID,
		Action:    "mfa_enabled",
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		CreatedAt: time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "MFA enabled successfully"})
}

// DisableMFA disables MFA for a user
func (h *AuthHandler) DisableMFA(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	// Get current secret to validate
	secret, err := h.store.GetUserMFASecret(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify"})
		return
	}

	if secret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA not enabled"})
		return
	}

	// Validate code
	if !h.mfaService.Validate(req.Code, secret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid code"})
		return
	}

	// Disable MFA
	if err := h.store.DisableMFA(ctx, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable MFA"})
		return
	}

	// Log audit event
	h.store.CreateAuditLog(ctx, &models.AuditLog{
		ID:        uuid.New().String(),
		UserID:    userID,
		Action:    "mfa_disabled",
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		CreatedAt: time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "MFA disabled successfully"})
}

// ValidateMFA validates an MFA code for session elevation or login
func (h *AuthHandler) ValidateMFA(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	// Get secret
	secret, err := h.store.GetUserMFASecret(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify"})
		return
	}

	if secret == "" {
		// MFA not enabled, so technically valid, or error?
		// For validation endpoint, we expect it to be enabled.
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA not enabled"})
		return
	}

	if !h.mfaService.Validate(req.Code, secret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

// CompleteMFALogin validates MFA code and returns full auth token
func (h *AuthHandler) CompleteMFALogin(c *gin.Context) {
	// Get user ID from the MFA token
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Check if this is an MFA pending token
	mfaPending := c.GetBool("mfa_pending")
	if !mfaPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token type"})
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	// Get user
	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// Get MFA secret
	secret, err := h.store.GetUserMFASecret(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify"})
		return
	}

	if secret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA not enabled"})
		return
	}

	if !h.mfaService.Validate(req.Code, secret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid code"})
		return
	}

	// MFA verified - generate full auth token
	authToken, err := h.generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Build user response
	name := user.Username
	if user.FirstName != "" || user.LastName != "" {
		name = strings.TrimSpace(user.FirstName + " " + user.LastName)
	}

	c.JSON(http.StatusOK, gin.H{
		"token": authToken,
		"user": gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"username":   user.Username,
			"name":       name,
			"avatar":     user.Avatar,
			"tier":       user.Tier,
			"is_admin":   user.IsAdmin,
			"mfa_enabled": user.MFAEnabled,
		},
	})
}

// GetAuditLogs returns the audit logs for the current user
func (h *AuthHandler) GetAuditLogs(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Pagination
	limit := 50
	offset := 0
	
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}
	
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offset = val
		}
	}

	ctx := context.Background()
	logs, err := h.store.GetAuditLogsByUserID(ctx, userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
		return
	}

	if logs == nil {
		logs = []*models.AuditLog{}
	}

	c.JSON(http.StatusOK, logs)
}
