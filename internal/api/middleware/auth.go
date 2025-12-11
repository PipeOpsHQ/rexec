package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rexec/rexec/internal/auth"
	"github.com/rexec/rexec/internal/models"
	"github.com/rexec/rexec/internal/storage"
)

// AuthMiddleware validates JWT tokens and extracts user info
// AuthMiddleware validates JWT tokens and extracts user info, enforcing MFA if enabled.
func AuthMiddleware(store *storage.PostgresStore, mfaService *auth.MFAService) gin.HandlerFunc {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "rexec-dev-secret-change-in-production"
	}
	jwtSecret := []byte(secret)

	return func(c *gin.Context) {
		// Get token from Authorization header or query params
		authHeader := c.GetHeader("Authorization")
		tokenString := ""

		if authHeader != "" {
			parts := strings.Fields(authHeader)
			if len(parts) >= 2 && strings.ToLower(parts[0]) == "bearer" {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			// Check for token in query params (for WebSocket connections)
			tokenQuery := c.Query("token")
			if tokenQuery != "" {
				tokenString = tokenQuery
			}
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
			c.Abort()
			return
		}

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return jwtSecret, nil
		})

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			// Log failed audit attempt
			store.CreateAuditLog(context.Background(), &models.AuditLog{
				ID:        uuid.New().String(),
				UserID:    "anonymous", // No user ID yet
				Action:    "authentication_failed",
				IPAddress: c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
				Details:   fmt.Sprintf("Invalid or expired token: %v", err),
				CreatedAt: time.Now(),
			})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Extract claims and set user info in context
		userID, userID_ok := claims["user_id"].(string)
		email, email_ok := claims["email"].(string)
		username, username_ok := claims["username"].(string)
		tier, tier_ok := claims["tier"].(string)
		guest, guest_ok := claims["guest"].(bool)
		subActive, subActive_ok := claims["subscription_active"].(bool)
		exp, exp_ok := claims["exp"].(float64)

		if !userID_ok || !email_ok || !username_ok || !tier_ok || !guest_ok || !subActive_ok || !exp_ok {
			store.CreateAuditLog(context.Background(), &models.AuditLog{
				ID:        uuid.New().String(),
				UserID:    userID,
				Action:    "authentication_failed",
				IPAddress: c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
				Details:   "Invalid token claims structure",
				CreatedAt: time.Now(),
			})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Set("email", email)
		c.Set("username", username)
		c.Set("tier", tier)
		c.Set("guest", guest)
		c.Set("subscription_active", subActive)
		c.Set("tokenExp", int64(exp))

		// Fetch user from DB to check MFA status
		user, err := store.GetUserByID(context.Background(), userID)
		if err != nil || user == nil {
			store.CreateAuditLog(context.Background(), &models.AuditLog{
				ID:        uuid.New().String(),
				UserID:    userID,
				Action:    "authentication_failed",
				IPAddress: c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
				Details:   fmt.Sprintf("User not found in DB: %v", err),
				CreatedAt: time.Now(),
			})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		// Store full user object in context for later use (e.g., in AdminOnly middleware)
		c.Set("user", user)

		// --- IP Whitelist Enforcement ---
		if len(user.AllowedIPs) > 0 {
			clientIP := c.ClientIP()
			if !checkIPWhitelist(clientIP, user.AllowedIPs) {
				store.CreateAuditLog(context.Background(), &models.AuditLog{
					ID:        uuid.New().String(),
					UserID:    userID,
					Action:    "ip_blocked",
					IPAddress: clientIP,
					UserAgent: c.Request.UserAgent(),
					Details:   fmt.Sprintf("IP %s not in allowed list", clientIP),
					CreatedAt: time.Now(),
				})
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied from this IP address"})
				c.Abort()
				return
			}
		}

		// --- MFA Enforcement ---
		if user.MFAEnabled {
			mfaCode := c.GetHeader("X-MFA-Code")
			if mfaCode == "" {
				mfaCode = c.Query("mfa_code")
			}

			if mfaCode == "" {
				store.CreateAuditLog(context.Background(), &models.AuditLog{
					ID:        uuid.New().String(),
					UserID:    userID,
					Action:    "mfa_required",
					IPAddress: c.ClientIP(),
					UserAgent: c.Request.UserAgent(),
					Details:   "MFA code missing for enabled account",
					CreatedAt: time.Now(),
				})
				c.JSON(http.StatusUnauthorized, gin.H{"error": "MFA code required"})
				c.Abort()
				return
			}

			mfaSecret, err := store.GetUserMFASecret(context.Background(), userID)
			if err != nil {
				store.CreateAuditLog(context.Background(), &models.AuditLog{
					ID:        uuid.New().String(),
					UserID:    userID,
					Action:    "mfa_validation_failed",
					IPAddress: c.ClientIP(),
					UserAgent: c.Request.UserAgent(),
					Details:   fmt.Sprintf("Failed to retrieve MFA secret: %v", err),
					CreatedAt: time.Now(),
				})
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify MFA"})
				c.Abort()
				return
			}

			if !mfaService.Validate(mfaCode, mfaSecret) {
				store.CreateAuditLog(context.Background(), &models.AuditLog{
					ID:        uuid.New().String(),
					UserID:    userID,
					Action:    "mfa_validation_failed",
					IPAddress: c.ClientIP(),
					UserAgent: c.Request.UserAgent(),
					Details:   "Invalid MFA code provided",
					CreatedAt: time.Now(),
				})
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid MFA code"})
				c.Abort()
				return
			}
			c.Set("mfa_verified", true) // Indicate MFA was successfully verified
		}

		// Log successful authentication
		store.CreateAuditLog(context.Background(), &models.AuditLog{
			ID:        uuid.New().String(),
			UserID:    userID,
			Action:    "authentication_success",
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Details:   fmt.Sprintf("User '%s' authenticated successfully.", username),
			CreatedAt: time.Now(),
		})

		c.Next()
	}
}

// checkIPWhitelist checks if a client IP is allowed
func checkIPWhitelist(clientIP string, allowedIPs []string) bool {
	if len(allowedIPs) == 0 {
		return true
	}

	client := net.ParseIP(clientIP)
	if client == nil {
		return false // Invalid client IP
	}

	for _, ipStr := range allowedIPs {
		ipStr = strings.TrimSpace(ipStr)
		if ipStr == "" {
			continue
		}

		// Check for CIDR
		if strings.Contains(ipStr, "/") {
			_, subnet, err := net.ParseCIDR(ipStr)
			if err == nil && subnet.Contains(client) {
				return true
			}
		} else {
			// Exact match
			ip := net.ParseIP(ipStr)
			if ip != nil && ip.Equal(client) {
				return true
			}
		}
	}

	return false
}
