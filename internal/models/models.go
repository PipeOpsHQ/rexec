package models

import (
	"time"
)

// User represents a registered user
type User struct {
	ID                 string   `json:"id"`
	Email              string   `json:"email"`
	Username           string   `json:"username"`
	Name               string   `json:"name,omitempty"` // Display name (first + last, or username)
	FirstName          string   `json:"first_name,omitempty"`
	LastName           string   `json:"last_name,omitempty"`
	Avatar             string   `json:"avatar,omitempty"`
	Verified           bool     `json:"verified,omitempty"`
	SubscriptionActive bool     `json:"subscription_active,omitempty"`
	IsAdmin            bool     `json:"is_admin,omitempty"`
	Password           string   `json:"-"`                     // Never serialize password
	Tier               string   `json:"tier"`                  // free, pro, enterprise
	PipeOpsID          string   `json:"pipeops_id,omitempty"`  // PipeOps OAuth user ID
	MFAEnabled         bool     `json:"mfa_enabled"`           // Whether MFA is enabled
	MFASecret          string   `json:"-"`                     // TOTP secret (encrypted)
	AllowedIPs         []string `json:"allowed_ips,omitempty"` // Whitelisted IPs/CIDRs
	// Screen lock (server-enforced, cross-session)
	ScreenLockEnabled bool       `json:"screen_lock_enabled,omitempty"`
	LockAfterMinutes  int        `json:"lock_after_minutes,omitempty"`
	LockRequiredSince *time.Time `json:"lock_required_since,omitempty"`
	ScreenLockHash    string     `json:"-"` // Never serialize
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// AuditLog represents a system audit log entry
type AuditLog struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"` // e.g., "login", "container_create"
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Details   string    `json:"details,omitempty"` // JSON details
	CreatedAt time.Time `json:"created_at"`
}

// RemoteHost represents a saved remote SSH connection (Jump Host target)
type RemoteHost struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Name         string    `json:"name"`
	Hostname     string    `json:"hostname"`
	Port         int       `json:"port"`
	Username     string    `json:"username"`
	IdentityFile string    `json:"identity_file,omitempty"` // Path to private key in container
	CreatedAt    time.Time `json:"created_at"`
}

// PortForward represents a user-defined port forwarding rule for a container
type PortForward struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	ContainerID   string    `json:"container_id"`
	Name          string    `json:"name"`           // Optional user-friendly name
	ContainerPort int       `json:"container_port"` // Port inside the container
	LocalPort     int       `json:"local_port"`     // Port on the user's local machine (browser client)
	Protocol      string    `json:"protocol"`       // e.g., "tcp"
	IsActive      bool      `json:"is_active"`      // Whether the forward is currently active
	CreatedAt     time.Time `json:"created_at"`
}

// Snippet represents a saved script or macro
type Snippet struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Username        string    `json:"username,omitempty"` // For marketplace display
	Name            string    `json:"name"`
	Content         string    `json:"content"`
	Language        string    `json:"language"` // bash, python, etc.
	IsPublic        bool      `json:"is_public"`
	UsageCount      int       `json:"usage_count"`
	Description     string    `json:"description,omitempty"`
	Icon            string    `json:"icon,omitempty"`            // Emoji icon for display
	Category        string    `json:"category,omitempty"`        // devops, security, database, etc.
	InstallCommand  string    `json:"install_command,omitempty"` // Command to install dependencies
	RequiresInstall bool      `json:"requires_install"`          // Whether install step is needed
	CreatedAt       time.Time `json:"created_at"`
}

// APIToken represents a personal access token for CLI/API usage
type APIToken struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`                   // User-friendly name for the token
	TokenHash   string     `json:"-"`                      // Hashed token (never exposed)
	TokenPrefix string     `json:"token_prefix"`           // First 8 chars for identification
	Scopes      []string   `json:"scopes"`                 // Permissions: read, write, admin
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"` // Last time token was used
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`   // Optional expiration
	CreatedAt   time.Time  `json:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"` // When token was revoked
}

// Container represents a user's terminal container
type Container struct {
	ID         string            `json:"id"`
	UserID     string            `json:"user_id"`
	Name       string            `json:"name"`
	Image      string            `json:"image"` // ubuntu, debian, arch, etc.
	Role       string            `json:"role"`  // devops, node, python, etc.
	Status     ContainerStatus   `json:"status"`
	IPAddress  string            `json:"ip_address,omitempty"`
	VolumePath string            `json:"volume_path,omitempty"`
	Resources  ResourceLimits    `json:"resources"`
	Labels     map[string]string `json:"labels,omitempty"`
	DockerID   string            `json:"docker_id,omitempty"` // Actual Docker container ID
	CreatedAt  time.Time         `json:"created_at"`
	LastUsedAt time.Time         `json:"last_used_at"`
	DeletedAt  *time.Time        `json:"deleted_at,omitempty"` // Soft delete timestamp
}

// ContainerStatus represents the state of a container
type ContainerStatus string

const (
	StatusCreating ContainerStatus = "creating"
	StatusRunning  ContainerStatus = "running"
	StatusStopped  ContainerStatus = "stopped"
	StatusPaused   ContainerStatus = "paused"
	StatusError    ContainerStatus = "error"
)

// ResourceLimits defines resource constraints for a container
type ResourceLimits struct {
	CPUShares       int64         `json:"cpu_shares"`       // CPU shares (relative weight)
	MemoryMB        int64         `json:"memory_mb"`        // Memory limit in MB
	DiskMB          int64         `json:"disk_mb"`          // Disk quota in MB
	NetworkMB       int64         `json:"network_mb"`       // Network bandwidth limit in MB/s
	SessionDuration time.Duration `json:"session_duration"` // 0 = unlimited
	MaxContainers   int64         `json:"max_containers"`   // Maximum number of containers allowed
}

// GuestResourceLimits defines the very restricted limits for anonymous guest users
var GuestResourceLimits = ResourceLimits{
	CPUShares:       500, // 0.5 CPU
	MemoryMB:        512,
	DiskMB:          2048,
	NetworkMB:       5,
	SessionDuration: 1 * time.Hour,
	MaxContainers:   1,
}

// Session represents an active terminal session
type Session struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	ContainerID string    `json:"container_id"`
	PTY         string    `json:"-"` // PTY file descriptor path
	Cols        uint16    `json:"cols"`
	Rows        uint16    `json:"rows"`
	CreatedAt   time.Time `json:"created_at"`
	LastPingAt  time.Time `json:"last_ping_at"`
}

// AuthenticatedSessionDuration is the limit for free users without active subscription
const AuthenticatedSessionDuration = 50 * time.Hour

// TierLimits is deprecated, use GetUserResourceLimits instead
func TierLimits(tier string) ResourceLimits {
	return GetUserResourceLimits(tier, false)
}

// GetUserResourceLimits returns resource limits based on user tier and subscription status
func GetUserResourceLimits(tier string, subscriptionActive bool) ResourceLimits {
	// Active subscription gets "Pro" limits regardless of tier label
	if subscriptionActive {
		return ResourceLimits{
			CPUShares:       4000,  // 4 CPUs
			MemoryMB:        4096,  // 4GB RAM
			DiskMB:          20480, // 20GB Storage
			NetworkMB:       100,
			SessionDuration: 0, // Unlimited
			MaxContainers:   10,
		}
	}

	switch tier {
	case "guest":
		return GuestResourceLimits
	case "free":
		// Authenticated but no subscription: Half resources, 50h limit
		return ResourceLimits{
			CPUShares:       2000,  // 2 CPUs
			MemoryMB:        2048,  // 2GB RAM
			DiskMB:          10240, // 10GB Storage
			NetworkMB:       10,
			SessionDuration: AuthenticatedSessionDuration,
			MaxContainers:   5,
		}
	case "pro":
		// Legacy pro tier (if not covered by subscriptionActive check)
		return ResourceLimits{
			CPUShares:       4000,
			MemoryMB:        4096,
			DiskMB:          20480,
			NetworkMB:       100,
			SessionDuration: 0,
			MaxContainers:   10,
		}
	case "enterprise":
		return ResourceLimits{
			CPUShares:       8000, // 8 CPUs
			MemoryMB:        8192,
			DiskMB:          51200,
			NetworkMB:       500,
			SessionDuration: 0,
			MaxContainers:   25,
		}
	default: // Default to free limits
		return ResourceLimits{
			CPUShares:       2000,
			MemoryMB:        2048,
			DiskMB:          10240,
			NetworkMB:       10,
			SessionDuration: AuthenticatedSessionDuration,
			MaxContainers:   3,
		}
	}
}

// ShellConfig defines shell customization options
type ShellConfig struct {
	Enhanced        bool   `json:"enhanced"`         // Install oh-my-zsh + plugins (default: true)
	Theme           string `json:"theme,omitempty"`  // zsh theme: "rexec" (default), "minimal", "powerlevel10k"
	Autosuggestions bool   `json:"autosuggestions"`  // Enable zsh-autosuggestions (default: true)
	SyntaxHighlight bool   `json:"syntax_highlight"` // Enable zsh-syntax-highlighting (default: true)
	HistorySearch   bool   `json:"history_search"`   // Enable history-substring-search (default: true)
	GitAliases      bool   `json:"git_aliases"`      // Enable git shortcuts (default: true)
	SystemStats     bool   `json:"system_stats"`     // Show system stats on login (default: true)
}

// DefaultShellConfig returns the default shell configuration
func DefaultShellConfig() ShellConfig {
	return ShellConfig{
		Enhanced:        true,
		Theme:           "rexec",
		Autosuggestions: true,
		SyntaxHighlight: true,
		HistorySearch:   true,
		GitAliases:      true,
		SystemStats:     true,
	}
}

// MinimalShellConfig returns a minimal shell configuration (for "use Arch btw" users)
func MinimalShellConfig() ShellConfig {
	return ShellConfig{
		Enhanced:        false,
		Theme:           "",
		Autosuggestions: false,
		SyntaxHighlight: false,
		HistorySearch:   false,
		GitAliases:      false,
		SystemStats:     false,
	}
}

// CreateContainerRequest represents a request to create a new container
type CreateContainerRequest struct {
	Name        string `json:"name"`                     // Optional - auto-generated if empty
	Image       string `json:"image" binding:"required"` // Image type (ubuntu, debian, etc.) or "custom"
	CustomImage string `json:"custom_image,omitempty"`   // Required when Image is "custom"
	Role        string `json:"role,omitempty"`           // Optional role (node, python, etc.)
	// Shell customization
	Shell *ShellConfig `json:"shell,omitempty"` // Optional shell config (defaults to enhanced)
	// Trial resource customization (within limits)
	MemoryMB  int64 `json:"memory_mb,omitempty"`  // Optional: custom memory (256-1024 MB for trial)
	CPUShares int64 `json:"cpu_shares,omitempty"` // Optional: custom CPU shares (256-1024 for trial)
	DiskMB    int64 `json:"disk_mb,omitempty"`    // Optional: custom disk (1024-4096 MB for trial)
}

// TrialResourceLimits defines the min/max resource limits for trial users
type TrialResourceLimits struct {
	MinMemoryMB  int64
	MaxMemoryMB  int64
	MinCPUShares int64
	MaxCPUShares int64
	MinDiskMB    int64
	MaxDiskMB    int64
}

// GetTrialResourceLimits returns the allowed resource customization range for trial users
// CPUShares in millicores (500 = 0.5 CPU, 1000 = 1 CPU)
// During 60-day trial, allow generous limits - enforcement happens after trial ends
func GetTrialResourceLimits() TrialResourceLimits {
	return TrialResourceLimits{
		MinMemoryMB:  256,
		MaxMemoryMB:  4096, // 4GB max for trial (generous during 60-day period)
		MinCPUShares: 250,  // 0.25 CPU
		MaxCPUShares: 4000, // 4 CPU max for trial
		MinDiskMB:    1024,
		MaxDiskMB:    16384, // 16GB max for trial
	}
}

// ValidateTrialResources validates and clamps resource requests for various tiers
func ValidateTrialResources(req *CreateContainerRequest, tier string) ResourceLimits {
	// Determine the base limits for the tier
	baseLimits := TierLimits(tier)

	// Get the overall allowed customization range for free/trial users
	customizationLimits := GetTrialResourceLimits()

	result := baseLimits // Start with the base limits for the tier

	// Only "free", "guest", "trial" tiers are allowed to customize within the defined range.
	// For "pro" and "enterprise", we just return their predefined baseLimits, ignoring req.
	// For "guest", it gets its fixed restricted limits and no customization.
	// So, actual customization is for "free" and "trial" tiers only, clamped by customizationLimits.

	if tier == "free" || tier == "trial" {
		// Validate and clamp memory
		if req.MemoryMB > 0 {
			if req.MemoryMB < customizationLimits.MinMemoryMB {
				result.MemoryMB = customizationLimits.MinMemoryMB
			} else if req.MemoryMB > customizationLimits.MaxMemoryMB {
				result.MemoryMB = customizationLimits.MaxMemoryMB
			} else {
				result.MemoryMB = req.MemoryMB
			}
		}

		// Validate and clamp CPU shares
		if req.CPUShares > 0 {
			if req.CPUShares < customizationLimits.MinCPUShares {
				result.CPUShares = customizationLimits.MinCPUShares
			} else if req.CPUShares > customizationLimits.MaxCPUShares {
				result.CPUShares = customizationLimits.MaxCPUShares
			} else {
				result.CPUShares = req.CPUShares
			}
		}

		// Validate and clamp disk
		if req.DiskMB > 0 {
			if req.DiskMB < customizationLimits.MinDiskMB {
				result.DiskMB = customizationLimits.MinDiskMB
			} else if req.DiskMB > customizationLimits.MaxDiskMB {
				result.DiskMB = customizationLimits.MaxDiskMB
			} else {
				result.DiskMB = req.DiskMB
			}
		}
	}

	return result
}

// ResizeRequest represents a terminal resize request
type ResizeRequest struct {
	Cols uint16 `json:"cols" binding:"required"`
	Rows uint16 `json:"rows" binding:"required"`
}

// AuthRequest represents login/register request
type AuthRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Username string `json:"username,omitempty"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// APIError represents an API error response
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
