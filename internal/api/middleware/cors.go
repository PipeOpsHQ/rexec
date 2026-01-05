package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// Default allowed origins for Rexec across all domains
var defaultAllowedOrigins = []string{
	"https://rexec.pipeops.app",
	"https://rexec.pipeops.io",
	"https://rexec.pipeops.sh",
	"https://rexec.io",
	"https://rexec.sh",
	"https://rexec.cloud",
}

// CORSMiddleware applies a CORS policy that supports the embed widget.
//   - WebSocket routes (/ws/*) and authenticated API requests allow any origin
//     since authentication is handled by the auth middleware.
//   - In development (non-release) with no ALLOWED_ORIGINS set, all origins are allowed.
//   - In release mode, unauthenticated requests only allow origins explicitly listed
//     in ALLOWED_ORIGINS (plus default Rexec domains).
//
// ALLOWED_ORIGINS is a comma-separated list like "https://app.rexec.com,https://staging.rexec.com".
func CORSMiddleware() gin.HandlerFunc {
	allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
	allowedOrigins := make(map[string]struct{})

	// Add default Rexec domains
	for _, origin := range defaultAllowedOrigins {
		allowedOrigins[origin] = struct{}{}
	}

	// Add any additional origins from environment
	if allowedOriginsStr != "" {
		for _, origin := range strings.Split(allowedOriginsStr, ",") {
			o := strings.TrimSpace(origin)
			if o != "" {
				allowedOrigins[o] = struct{}{}
			}
		}
	}
	isRelease := os.Getenv("GIN_MODE") == "release"

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		path := c.Request.URL.Path

		// Check if this is a WebSocket route or authenticated API request
		// These allow any origin since security is handled by token validation
		isWebSocketRoute := strings.HasPrefix(path, "/ws/")
		hasAuthHeader := c.Request.Header.Get("Authorization") != ""

		// For embed widget support, allow any origin when:
		// 1. It's a WebSocket route (auth handled by AuthMiddleware before upgrade)
		// 2. Request has Authorization header (authenticated API call from embed widget)
		allowAnyOrigin := isWebSocketRoute || hasAuthHeader

		if origin != "" {
			if allowAnyOrigin {
				// Allow any origin for WebSocket routes and authenticated requests
				// This is essential for the embed widget which can be loaded from ANY third-party domain
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Vary", "Origin")
			} else if _, ok := allowedOrigins[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Vary", "Origin")
			} else if !isRelease {
				// In development, allow any origin
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Vary", "Origin")
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		// Include Authorization and WebSocket headers
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Upgrade, Sec-WebSocket-Key, Sec-WebSocket-Version, Sec-WebSocket-Protocol, Sec-WebSocket-Extensions")

		if c.Request.Method == http.MethodOptions {
			// For preflight requests, we need to allow the origin if it will be allowed
			// for the actual request. Since we can't know if the actual request will have
			// an Authorization header, we allow all origins for preflight to API routes.
			isAPIRoute := strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws/")
			if isAPIRoute && origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Vary", "Origin")
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
