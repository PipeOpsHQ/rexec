package container

import (
	"context"
	"log"
	"time"
)

const (
	// GuestMaxSessionDuration is the maximum time a guest container can run
	GuestMaxSessionDuration = 2 * time.Hour
)

// CleanupService handles automatic cleanup of idle containers
type CleanupService struct {
	manager       *Manager
	idleTimeout   time.Duration
	checkInterval time.Duration
	stopChan      chan struct{}
}

// NewCleanupService creates a new cleanup service
func NewCleanupService(manager *Manager, idleTimeout, checkInterval time.Duration) *CleanupService {
	return &CleanupService{
		manager:       manager,
		idleTimeout:   idleTimeout,
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
	}
}

// Start begins the cleanup service
func (s *CleanupService) Start() {
	go s.run()
	log.Printf("🧹 Cleanup service started (idle timeout: %v, check interval: %v)", s.idleTimeout, s.checkInterval)
}

// Stop stops the cleanup service
func (s *CleanupService) Stop() {
	close(s.stopChan)
	log.Println("🧹 Cleanup service stopped")
}

// run is the main loop for the cleanup service
func (s *CleanupService) run() {
	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupIdleContainers()
		case <-s.stopChan:
			return
		}
	}
}

// cleanupIdleContainers stops guest containers that have been idle for too long
// Note: Only guest containers have idle timeout - authenticated users' containers run indefinitely
func (s *CleanupService) cleanupIdleContainers() {
	// First, cleanup expired guest containers (hard 50-hour session limit)
	s.cleanupExpiredGuestContainers()

	// Then cleanup idle guest containers (idle timeout only applies to guests)
	idleContainers := s.manager.GetIdleContainers(s.idleTimeout)

	if len(idleContainers) == 0 {
		return
	}

	log.Printf("🧹 Found %d idle guest containers to stop", len(idleContainers))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, container := range idleContainers {
		log.Printf("🧹 Stopping idle container: %s (user: %s, last used: %v ago)",
			container.ID[:12],
			container.UserID,
			time.Since(container.LastUsedAt).Round(time.Second),
		)

		if err := s.manager.StopContainer(ctx, container.ID); err != nil {
			log.Printf("⚠️  Failed to stop idle container %s: %v", container.ID[:12], err)
		} else {
			log.Printf("✅ Stopped idle container: %s", container.ID[:12])
		}
	}
}

// cleanupExpiredGuestContainers stops and removes guest containers that have exceeded 50 hours
func (s *CleanupService) cleanupExpiredGuestContainers() {
	expiredGuests := s.manager.GetExpiredGuestContainers()

	if len(expiredGuests) == 0 {
		return
	}

	log.Printf("🧹 Found %d expired guest containers to remove", len(expiredGuests))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, container := range expiredGuests {
		sessionDuration := time.Since(container.CreatedAt).Round(time.Second)
		log.Printf("🧹 Removing expired guest container: %s (session: %v, max: %v)",
			container.ID[:12],
			sessionDuration,
			GuestMaxSessionDuration,
		)

		// Stop the container first
		if err := s.manager.StopContainer(ctx, container.ID); err != nil {
			log.Printf("⚠️  Failed to stop guest container %s: %v", container.ID[:12], err)
		}

		// Then remove it
		if err := s.manager.RemoveContainer(ctx, container.ID); err != nil {
			log.Printf("⚠️  Failed to remove guest container %s: %v", container.ID[:12], err)
		} else {
			log.Printf("✅ Removed expired guest container: %s", container.ID[:12])
		}
	}
}

// CleanupConfig holds configuration for the cleanup service
type CleanupConfig struct {
	// IdleTimeout is how long a guest container can be idle before being stopped
	// Note: This only applies to guest containers - authenticated users have no idle timeout
	IdleTimeout time.Duration

	// CheckInterval is how often to check for idle/expired containers
	CheckInterval time.Duration

	// Enabled controls whether the cleanup service is active
	Enabled bool
}

// DefaultCleanupConfig returns sensible defaults for cleanup
func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		IdleTimeout:   1 * time.Hour,   // Stop guest containers idle for 1 hour
		CheckInterval: 5 * time.Minute, // Check every 5 minutes
		Enabled:       true,
	}
}

// DevelopmentCleanupConfig returns config suitable for development
func DevelopmentCleanupConfig() CleanupConfig {
	return CleanupConfig{
		IdleTimeout:   30 * time.Minute, // Shorter timeout for dev
		CheckInterval: 1 * time.Minute,  // More frequent checks
		Enabled:       true,
	}
}

// GetIdleTime returns how long a container has been idle
func (m *Manager) GetIdleTime(dockerID string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if info, ok := m.containers[dockerID]; ok {
		return time.Since(info.LastUsedAt)
	}
	return 0
}

// GetExpiredGuestContainers returns containers that have exceeded their session limit
// Checks both guest containers (hard limit) and any container with an explicit expiration label
func (m *Manager) GetExpiredGuestContainers() []*ContainerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var expired []*ContainerInfo
	now := time.Now()

	for _, info := range m.containers {
		// Check if this is a guest container
		isGuest := false
		var expiresAt time.Time

		if info.Labels != nil {
			if tier, ok := info.Labels["rexec.tier"]; ok && tier == "guest" {
				isGuest = true
			}
			if _, ok := info.Labels["rexec.guest"]; ok {
				isGuest = true
			}
			// Check for explicit expiration time (set from guest session token or auth user limit)
			if expStr, ok := info.Labels["rexec.expires_at"]; ok {
				if t, err := time.Parse(time.RFC3339, expStr); err == nil {
					expiresAt = t
				}
			}
		}

		// If not guest and no expiration time set, it's a persistent container - skip
		if !isGuest && expiresAt.IsZero() {
			continue
		}

		// Use explicit expiration time if set, otherwise fall back to creation time + max duration (for guests)
		isExpired := false
		if !expiresAt.IsZero() {
			isExpired = now.After(expiresAt)
		} else if isGuest {
			// Fallback for guests without explicit label (shouldn't happen often)
			isExpired = time.Since(info.CreatedAt) > GuestMaxSessionDuration
		}

		if isExpired {
			expired = append(expired, info)
		}
	}
	return expired
}

// IsGuestContainer checks if a container belongs to a guest user
func (m *Manager) IsGuestContainer(dockerID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if info, ok := m.containers[dockerID]; ok {
		if info.Labels != nil {
			if tier, ok := info.Labels["rexec.tier"]; ok && tier == "guest" {
				return true
			}
			if _, ok := info.Labels["rexec.guest"]; ok {
				return true
			}
		}
	}
	return false
}

// GetGuestSessionTimeRemaining returns how much time is left for a guest container
func (m *Manager) GetGuestSessionTimeRemaining(dockerID string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if info, ok := m.containers[dockerID]; ok {
		// Check for explicit expiration time from label
		if info.Labels != nil {
			if expStr, ok := info.Labels["rexec.expires_at"]; ok {
				if expiresAt, err := time.Parse(time.RFC3339, expStr); err == nil {
					remaining := time.Until(expiresAt)
					if remaining < 0 {
						return 0
					}
					return remaining
				}
			}
		}
		// Fallback to creation time based calculation
		elapsed := time.Since(info.CreatedAt)
		remaining := GuestMaxSessionDuration - elapsed
		if remaining < 0 {
			return 0
		}
		return remaining
	}
	return 0
}

// GetContainerStats returns stats about containers
func (m *Manager) GetContainerStats() ContainerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := ContainerStats{
		Total:   len(m.containers),
		ByUser:  make(map[string]int),
		ByImage: make(map[string]int),
	}

	for _, info := range m.containers {
		stats.ByUser[info.UserID]++
		stats.ByImage[info.ImageType]++

		switch info.Status {
		case "running":
			stats.Running++
		case "stopped":
			stats.Stopped++
		default:
			stats.Other++
		}
	}

	return stats
}

// ContainerStats holds aggregate statistics about containers
type ContainerStats struct {
	Total   int
	Running int
	Stopped int
	Other   int
	ByUser  map[string]int
	ByImage map[string]int
}
