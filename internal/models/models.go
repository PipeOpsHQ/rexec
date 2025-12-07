package models

import (
	"time"
)

// User represents a registered user
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Name      string    `json:"name,omitempty"` // Display name (first + last, or username)
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	Avatar    string    `json:"avatar,omitempty"`
	Verified  bool      `json:"verified,omitempty"`
	SubscriptionActive bool `json:"subscription_active,omitempty"`
	Password  string    `json:"-"`                    // Never serialize password
	Tier      string    `json:"tier"`                 // free, pro, enterprise
	PipeOpsID string    `json:"pipeops_id,omitempty"` // PipeOps OAuth user ID
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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

// Container represents a user's terminal container
type Container struct {
	ID         string            `json:"id"`
	UserID     string            `json:"user_id"`
	Name       string            `json:"name"`
	Image      string            `json:"image"` // ubuntu, debian, arch, etc.
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
	CPUShares int64 `json:"cpu_shares"` // CPU shares (relative weight)
	MemoryMB  int64 `json:"memory_mb"`  // Memory limit in MB
	DiskMB    int64 `json:"disk_mb"`    // Disk quota in MB
	NetworkMB int64 `json:"network_mb"` // Network bandwidth limit in MB/s
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

// TierLimits returns resource limits based on user tier
// CPUShares represents CPU count in millicores (500 = 0.5 CPU, 1000 = 1 CPU)
func TierLimits(tier string) ResourceLimits {
	switch tier {
	case "trial", "guest", "free": // Unified 60-day trial
		return ResourceLimits{
			CPUShares: 500, // 0.5 CPU
			MemoryMB:  512,
			DiskMB:    2048, // Increased from 512-1024 for better trial experience
			NetworkMB: 10,
		}
	case "pro":
		return ResourceLimits{
			CPUShares: 2000, // 2 CPUs
			MemoryMB:  2048,
			DiskMB:    10240,
			NetworkMB: 100,
		}
	case "enterprise":
		return ResourceLimits{
			CPUShares: 4000, // 4 CPUs
			MemoryMB:  4096,
			DiskMB:    51200,
			NetworkMB: 500,
		}
	default: // Default to trial tier
		return ResourceLimits{
			CPUShares: 500, // 0.5 CPU
			MemoryMB:  512,
			DiskMB:    2048,
			NetworkMB: 10,
		}
	}
}

// ShellConfig defines shell customization options
type ShellConfig struct {
	Enhanced        bool `json:"enhanced"`          // Install oh-my-zsh + plugins (default: true)
	Theme           string `json:"theme,omitempty"` // zsh theme: "rexec" (default), "minimal", "powerlevel10k"
	Autosuggestions bool `json:"autosuggestions"`   // Enable zsh-autosuggestions (default: true)
	SyntaxHighlight bool `json:"syntax_highlight"`  // Enable zsh-syntax-highlighting (default: true)
	HistorySearch   bool `json:"history_search"`    // Enable history-substring-search (default: true)
	GitAliases      bool `json:"git_aliases"`       // Enable git shortcuts (default: true)
	SystemStats     bool `json:"system_stats"`      // Show system stats on login (default: true)
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

// ValidateTrialResources validates and clamps resource requests for trial users
func ValidateTrialResources(req *CreateContainerRequest, tier string) ResourceLimits {
	defaults := TierLimits(tier)
	limits := GetTrialResourceLimits()
	
	// Only trial/guest/free can customize within limits
	if tier != "trial" && tier != "guest" && tier != "free" {
		// Pro/Enterprise get their defaults (or could be expanded later)
		return defaults
	}
	
	result := defaults
	
	// Validate and clamp memory
	if req.MemoryMB > 0 {
		if req.MemoryMB < limits.MinMemoryMB {
			result.MemoryMB = limits.MinMemoryMB
		} else if req.MemoryMB > limits.MaxMemoryMB {
			result.MemoryMB = limits.MaxMemoryMB
		} else {
			result.MemoryMB = req.MemoryMB
		}
	}
	
	// Validate and clamp CPU shares
	if req.CPUShares > 0 {
		if req.CPUShares < limits.MinCPUShares {
			result.CPUShares = limits.MinCPUShares
		} else if req.CPUShares > limits.MaxCPUShares {
			result.CPUShares = limits.MaxCPUShares
		} else {
			result.CPUShares = req.CPUShares
		}
	}
	
	// Validate and clamp disk
	if req.DiskMB > 0 {
		if req.DiskMB < limits.MinDiskMB {
			result.DiskMB = limits.MinDiskMB
		} else if req.DiskMB > limits.MaxDiskMB {
			result.DiskMB = limits.MaxDiskMB
		} else {
			result.DiskMB = req.DiskMB
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
