package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore handles Redis operations for sessions and rate limiting
type RedisStore struct {
	client *redis.Client
	prefix string
}

// NewRedisStore creates a new Redis store
func NewRedisStore(redisURL string) (*RedisStore, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisStore{
		client: client,
		prefix: "rexec:",
	}, nil
}

// Close closes the Redis connection
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// Key helpers
func (s *RedisStore) sessionKey(sessionID string) string {
	return s.prefix + "session:" + sessionID
}

func (s *RedisStore) userSessionsKey(userID string) string {
	return s.prefix + "user_sessions:" + userID
}

func (s *RedisStore) rateLimitKey(ip, endpoint string) string {
	return s.prefix + "ratelimit:" + endpoint + ":" + ip
}

func (s *RedisStore) containerSessionKey(containerID string) string {
	return s.prefix + "container_session:" + containerID
}

// Session represents a user session
type Session struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Email      string    `json:"email"`
	Username   string    `json:"username"`
	Tier       string    `json:"tier"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
}

// Session operations

// CreateSession creates a new session
func (s *RedisStore) CreateSession(ctx context.Context, session *Session, ttl time.Duration) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	pipe := s.client.Pipeline()

	// Store session
	pipe.Set(ctx, s.sessionKey(session.ID), data, ttl)

	// Add to user's session set
	pipe.SAdd(ctx, s.userSessionsKey(session.UserID), session.ID)
	pipe.Expire(ctx, s.userSessionsKey(session.UserID), ttl)

	_, err = pipe.Exec(ctx)
	return err
}

// GetSession retrieves a session by ID
func (s *RedisStore) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	data, err := s.client.Get(ctx, s.sessionKey(sessionID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// UpdateSessionLastSeen updates the last seen timestamp
func (s *RedisStore) UpdateSessionLastSeen(ctx context.Context, sessionID string) error {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil || session == nil {
		return err
	}

	session.LastSeenAt = time.Now()
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	// Keep existing TTL
	ttl, err := s.client.TTL(ctx, s.sessionKey(sessionID)).Result()
	if err != nil || ttl <= 0 {
		ttl = 24 * time.Hour
	}

	return s.client.Set(ctx, s.sessionKey(sessionID), data, ttl).Err()
}

// DeleteSession removes a session
func (s *RedisStore) DeleteSession(ctx context.Context, sessionID string) error {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return nil
	}

	pipe := s.client.Pipeline()
	pipe.Del(ctx, s.sessionKey(sessionID))
	pipe.SRem(ctx, s.userSessionsKey(session.UserID), sessionID)
	_, err = pipe.Exec(ctx)
	return err
}

// GetUserSessions returns all session IDs for a user
func (s *RedisStore) GetUserSessions(ctx context.Context, userID string) ([]string, error) {
	return s.client.SMembers(ctx, s.userSessionsKey(userID)).Result()
}

// DeleteAllUserSessions removes all sessions for a user
func (s *RedisStore) DeleteAllUserSessions(ctx context.Context, userID string) error {
	sessionIDs, err := s.GetUserSessions(ctx, userID)
	if err != nil {
		return err
	}

	if len(sessionIDs) == 0 {
		return nil
	}

	pipe := s.client.Pipeline()
	for _, sessionID := range sessionIDs {
		pipe.Del(ctx, s.sessionKey(sessionID))
	}
	pipe.Del(ctx, s.userSessionsKey(userID))
	_, err = pipe.Exec(ctx)
	return err
}

// Rate limiting operations

// RateLimitResult contains the result of a rate limit check
type RateLimitResult struct {
	Allowed   bool
	Remaining int64
	ResetAt   time.Time
}

// CheckRateLimit checks and updates rate limit for an IP/endpoint combination
// Uses sliding window algorithm
func (s *RedisStore) CheckRateLimit(ctx context.Context, ip, endpoint string, limit int64, window time.Duration) (*RateLimitResult, error) {
	key := s.rateLimitKey(ip, endpoint)
	now := time.Now()
	windowStart := now.Add(-window)

	pipe := s.client.Pipeline()

	// Remove old entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// Count current requests in window
	countCmd := pipe.ZCard(ctx, key)

	// Add current request
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})

	// Set expiry on the key
	pipe.Expire(ctx, key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	count := countCmd.Val()
	remaining := limit - count - 1
	if remaining < 0 {
		remaining = 0
	}

	return &RateLimitResult{
		Allowed:   count < limit,
		Remaining: remaining,
		ResetAt:   now.Add(window),
	}, nil
}

// GetRateLimitRemaining returns remaining requests for an IP/endpoint
func (s *RedisStore) GetRateLimitRemaining(ctx context.Context, ip, endpoint string, limit int64, window time.Duration) (int64, error) {
	key := s.rateLimitKey(ip, endpoint)
	windowStart := time.Now().Add(-window)

	// Count requests in current window
	count, err := s.client.ZCount(ctx, key, fmt.Sprintf("%d", windowStart.UnixNano()), "+inf").Result()
	if err != nil {
		return limit, err
	}

	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// ResetRateLimit clears rate limit for an IP/endpoint
func (s *RedisStore) ResetRateLimit(ctx context.Context, ip, endpoint string) error {
	return s.client.Del(ctx, s.rateLimitKey(ip, endpoint)).Err()
}

// Terminal session tracking (for reconnection)

// TerminalSessionInfo stores info about active terminal sessions
type TerminalSessionInfo struct {
	ContainerID string    `json:"container_id"`
	UserID      string    `json:"user_id"`
	SessionID   string    `json:"session_id"`
	ConnectedAt time.Time `json:"connected_at"`
	LastPingAt  time.Time `json:"last_ping_at"`
}

// SetTerminalSession stores terminal session info
func (s *RedisStore) SetTerminalSession(ctx context.Context, info *TerminalSessionInfo, ttl time.Duration) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.containerSessionKey(info.ContainerID), data, ttl).Err()
}

// GetTerminalSession retrieves terminal session info
func (s *RedisStore) GetTerminalSession(ctx context.Context, containerID string) (*TerminalSessionInfo, error) {
	data, err := s.client.Get(ctx, s.containerSessionKey(containerID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var info TerminalSessionInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// DeleteTerminalSession removes terminal session info
func (s *RedisStore) DeleteTerminalSession(ctx context.Context, containerID string) error {
	return s.client.Del(ctx, s.containerSessionKey(containerID)).Err()
}

// UpdateTerminalPing updates the last ping time for a terminal session
func (s *RedisStore) UpdateTerminalPing(ctx context.Context, containerID string) error {
	info, err := s.GetTerminalSession(ctx, containerID)
	if err != nil || info == nil {
		return err
	}

	info.LastPingAt = time.Now()
	return s.SetTerminalSession(ctx, info, 30*time.Minute)
}

// Cache operations for general use

// Set stores a value with optional TTL
func (s *RedisStore) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.prefix+key, data, ttl).Err()
}

// Get retrieves a value
func (s *RedisStore) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := s.client.Get(ctx, s.prefix+key).Bytes()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Delete removes a key
func (s *RedisStore) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, s.prefix+key).Err()
}

// Exists checks if a key exists
func (s *RedisStore) Exists(ctx context.Context, key string) (bool, error) {
	result, err := s.client.Exists(ctx, s.prefix+key).Result()
	return result > 0, err
}

// Increment atomically increments a counter
func (s *RedisStore) Increment(ctx context.Context, key string) (int64, error) {
	return s.client.Incr(ctx, s.prefix+key).Result()
}

// Stats returns Redis connection stats
func (s *RedisStore) Stats() *redis.PoolStats {
	return s.client.PoolStats()
}

// Ping tests the Redis connection
func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}
