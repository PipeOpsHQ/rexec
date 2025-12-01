package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     int           // requests per window
	window   time.Duration // time window
}

type visitor struct {
	tokens    int
	lastReset time.Time
}

// NewRateLimiter creates a new rate limiter
// rate: number of requests allowed per window
// window: time window duration
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}

	// Start cleanup goroutine to remove old entries
	go rl.cleanup()

	return rl
}

// cleanup removes stale visitor entries every minute
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastReset) > rl.window*2 {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// isAllowed checks if a request from the given IP is allowed
func (rl *RateLimiter) isAllowed(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists {
		rl.visitors[ip] = &visitor{
			tokens:    rl.rate - 1,
			lastReset: now,
		}
		return true
	}

	// Reset tokens if window has passed
	if now.Sub(v.lastReset) >= rl.window {
		v.tokens = rl.rate - 1
		v.lastReset = now
		return true
	}

	// Check if tokens available
	if v.tokens > 0 {
		v.tokens--
		return true
	}

	return false
}

// getRemaining returns remaining tokens for an IP
func (rl *RateLimiter) getRemaining(ip string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if v, exists := rl.visitors[ip]; exists {
		return v.tokens
	}
	return rl.rate
}

// Middleware returns a Gin middleware for rate limiting
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !rl.isAllowed(ip) {
			c.Header("X-RateLimit-Limit", string(rune(rl.rate)))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", rl.window.String())

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": rl.window.Seconds(),
			})
			c.Abort()
			return
		}

		// Add rate limit headers
		remaining := rl.getRemaining(ip)
		c.Header("X-RateLimit-Limit", string(rune(rl.rate)))
		c.Header("X-RateLimit-Remaining", string(rune(remaining)))

		c.Next()
	}
}

// Default rate limiters for different endpoints

// APIRateLimiter returns a rate limiter for general API requests
// 100 requests per minute
func APIRateLimiter() *RateLimiter {
	return NewRateLimiter(100, 1*time.Minute)
}

// AuthRateLimiter returns a stricter rate limiter for auth endpoints
// 10 requests per minute (prevents brute force)
func AuthRateLimiter() *RateLimiter {
	return NewRateLimiter(10, 1*time.Minute)
}

// WebSocketRateLimiter returns a rate limiter for WebSocket connections
// 20 connections per minute
func WebSocketRateLimiter() *RateLimiter {
	return NewRateLimiter(20, 1*time.Minute)
}

// ContainerRateLimiter returns a rate limiter for container operations
// 30 operations per minute
func ContainerRateLimiter() *RateLimiter {
	return NewRateLimiter(30, 1*time.Minute)
}
