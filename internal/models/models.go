package models

import (
	"time"
)

// User represents a registered user
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`                    // Never serialize password
	Tier      string    `json:"tier"`                 // free, pro, enterprise
	PipeOpsID string    `json:"pipeops_id,omitempty"` // PipeOps OAuth user ID
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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

// ImageInfo represents available container images
type ImageInfo struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Size        int64    `json:"size"`
}

// AvailableImages returns the list of supported OS images
func AvailableImages() []ImageInfo {
	return []ImageInfo{
		{
			Name:        "ubuntu",
			DisplayName: "Ubuntu 22.04 LTS",
			Description: "Ubuntu with bash and common tools",
			Tags:        []string{"latest", "22.04"},
		},
		{
			Name:        "debian",
			DisplayName: "Debian 12 (Bookworm)",
			Description: "Stable Debian with essential packages",
			Tags:        []string{"latest", "12"},
		},
		{
			Name:        "alpine",
			DisplayName: "Alpine Linux",
			Description: "Lightweight Linux with minimal footprint",
			Tags:        []string{"latest"},
		},
		{
			Name:        "fedora",
			DisplayName: "Fedora",
			Description: "Latest Fedora with DNF package manager",
			Tags:        []string{"latest"},
		},
	}
}

// TierLimits returns resource limits based on user tier
func TierLimits(tier string) ResourceLimits {
	switch tier {
	case "pro":
		return ResourceLimits{
			CPUShares: 2048,
			MemoryMB:  2048,
			DiskMB:    10240,
			NetworkMB: 100,
		}
	case "enterprise":
		return ResourceLimits{
			CPUShares: 4096,
			MemoryMB:  4096,
			DiskMB:    51200,
			NetworkMB: 500,
		}
	case "guest":
		// Guest tier - limited resources, 1-hour session max
		return ResourceLimits{
			CPUShares: 256,
			MemoryMB:  256,
			DiskMB:    512,
			NetworkMB: 5,
		}
	default: // free tier (authenticated users)
		return ResourceLimits{
			CPUShares: 512,
			MemoryMB:  512,
			DiskMB:    1024,
			NetworkMB: 10,
		}
	}
}

// CreateContainerRequest represents a request to create a new container
type CreateContainerRequest struct {
	Name        string `json:"name"`                     // Optional - auto-generated if empty
	Image       string `json:"image" binding:"required"` // Image type (ubuntu, debian, etc.) or "custom"
	CustomImage string `json:"custom_image,omitempty"`   // Required when Image is "custom"
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
