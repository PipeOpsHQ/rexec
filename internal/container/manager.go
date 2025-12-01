package container

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// ProgressEvent represents a progress update during container creation
type ProgressEvent struct {
	Stage       string  `json:"stage"`                  // "validating", "pulling", "creating", "starting", "ready"
	Message     string  `json:"message"`                // Human-readable message
	Progress    float64 `json:"progress"`               // 0-100 percentage
	Detail      string  `json:"detail"`                 // Additional detail (e.g., layer being pulled)
	Error       string  `json:"error,omitempty"`        // Error message if failed
	Complete    bool    `json:"complete"`               // Whether the whole process is complete
	ContainerID string  `json:"container_id,omitempty"` // Set when complete
}

// PullProgress represents Docker's image pull progress JSON
type PullProgress struct {
	Status         string `json:"status"`
	ID             string `json:"id"`
	Progress       string `json:"progress"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
}

// SupportedImages maps user-friendly names to Docker images
// Uses custom rexec images if available (with SSH pre-installed), otherwise falls back to base images
var SupportedImages = map[string]string{
	// Debian-based
	"ubuntu":    "ubuntu:22.04",
	"ubuntu-24": "ubuntu:24.04",
	"ubuntu-20": "ubuntu:20.04",
	"debian":    "debian:bookworm",
	"debian-11": "debian:bullseye",
	"kali":      "kalilinux/kali-rolling:latest",
	"parrot":    "parrotsec/core:latest",
	// Red Hat-based
	"fedora":    "fedora:latest",
	"fedora-39": "fedora:39",
	"centos":    "centos:stream9",
	"rocky":     "rockylinux:9",
	"alma":      "almalinux:9",
	"oracle":    "oraclelinux:9",
	// Other Linux
	"alpine":      "alpine:latest",
	"alpine-3.18": "alpine:3.18",
	"archlinux":   "archlinux:latest",
	"opensuse":    "opensuse/leap:latest",
	"gentoo":      "gentoo/stage3:latest",
	"void":        "voidlinux/voidlinux:latest",
	"nixos":       "nixos/nix:latest",
	// Specialized
	"amazonlinux": "amazonlinux:2023",
	"clearlinux":  "clearlinux:latest",
	"photon":      "photon:latest",
	// Minimal/Distroless for specific use cases
	"busybox": "busybox:latest",
}

// CustomImages maps to rexec custom images with SSH pre-installed
// Build these with: ./scripts/build-images.sh
var CustomImages = map[string]string{
	"ubuntu":    "rexec-ubuntu:latest",
	"debian":    "rexec-debian:latest",
	"alpine":    "rexec-alpine:latest",
	"fedora":    "rexec-fedora:latest",
	"archlinux": "rexec-archlinux:latest",
	"kali":      "rexec-kali:latest",
}

// ImageMetadata provides display information about images
type ImageMetadata struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Popular     bool     `json:"popular"`
}

// GetImageMetadata returns metadata for all supported images
func GetImageMetadata() []ImageMetadata {
	return []ImageMetadata{
		// Debian-based (Popular)
		{Name: "ubuntu", DisplayName: "Ubuntu 22.04 LTS", Description: "Most popular Linux distro with excellent package support", Category: "debian", Tags: []string{"lts", "popular", "beginner-friendly"}, Popular: true},
		{Name: "ubuntu-24", DisplayName: "Ubuntu 24.04 LTS", Description: "Latest Ubuntu LTS release", Category: "debian", Tags: []string{"lts", "latest"}, Popular: true},
		{Name: "ubuntu-20", DisplayName: "Ubuntu 20.04 LTS", Description: "Stable Ubuntu with long-term support", Category: "debian", Tags: []string{"lts", "stable"}, Popular: false},
		{Name: "debian", DisplayName: "Debian 12 (Bookworm)", Description: "Rock-solid stability with extensive packages", Category: "debian", Tags: []string{"stable", "server"}, Popular: true},
		{Name: "debian-11", DisplayName: "Debian 11 (Bullseye)", Description: "Previous stable Debian release", Category: "debian", Tags: []string{"oldstable"}, Popular: false},
		{Name: "kali", DisplayName: "Kali Linux", Description: "Penetration testing and security auditing", Category: "debian", Tags: []string{"security", "pentesting"}, Popular: true},
		{Name: "parrot", DisplayName: "Parrot Security OS", Description: "Security and privacy focused distribution", Category: "debian", Tags: []string{"security", "privacy"}, Popular: false},

		// Red Hat-based
		{Name: "fedora", DisplayName: "Fedora (Latest)", Description: "Cutting-edge features and technologies", Category: "redhat", Tags: []string{"latest", "developer"}, Popular: true},
		{Name: "fedora-39", DisplayName: "Fedora 39", Description: "Stable Fedora release", Category: "redhat", Tags: []string{"stable"}, Popular: false},
		{Name: "centos", DisplayName: "CentOS Stream 9", Description: "Enterprise Linux for development", Category: "redhat", Tags: []string{"enterprise", "rhel"}, Popular: false},
		{Name: "rocky", DisplayName: "Rocky Linux 9", Description: "Enterprise-grade RHEL-compatible OS", Category: "redhat", Tags: []string{"enterprise", "rhel-compatible"}, Popular: true},
		{Name: "alma", DisplayName: "AlmaLinux 9", Description: "Community-driven RHEL fork", Category: "redhat", Tags: []string{"enterprise", "rhel-compatible"}, Popular: false},
		{Name: "oracle", DisplayName: "Oracle Linux 9", Description: "Oracle's enterprise Linux", Category: "redhat", Tags: []string{"enterprise", "oracle"}, Popular: false},

		// Other Linux
		{Name: "alpine", DisplayName: "Alpine Linux", Description: "Lightweight and security-oriented", Category: "other", Tags: []string{"minimal", "docker", "fast"}, Popular: true},
		{Name: "alpine-3.18", DisplayName: "Alpine 3.18", Description: "Stable Alpine release", Category: "other", Tags: []string{"minimal", "stable"}, Popular: false},
		{Name: "archlinux", DisplayName: "Arch Linux", Description: "Rolling release with latest packages", Category: "other", Tags: []string{"rolling", "bleeding-edge", "aur"}, Popular: true},
		{Name: "opensuse", DisplayName: "openSUSE Leap", Description: "Stable enterprise-grade openSUSE", Category: "other", Tags: []string{"enterprise", "zypper"}, Popular: false},
		{Name: "gentoo", DisplayName: "Gentoo Linux", Description: "Source-based with extreme customization", Category: "other", Tags: []string{"source-based", "advanced"}, Popular: false},
		{Name: "void", DisplayName: "Void Linux", Description: "Independent distro with runit init", Category: "other", Tags: []string{"independent", "runit"}, Popular: false},
		{Name: "nixos", DisplayName: "NixOS", Description: "Declarative and reproducible builds", Category: "other", Tags: []string{"declarative", "nix", "reproducible"}, Popular: false},

		// Specialized
		{Name: "amazonlinux", DisplayName: "Amazon Linux 2023", Description: "Optimized for AWS", Category: "specialized", Tags: []string{"aws", "cloud"}, Popular: false},
		{Name: "clearlinux", DisplayName: "Clear Linux", Description: "Intel-optimized performance", Category: "specialized", Tags: []string{"performance", "intel"}, Popular: false},
		{Name: "photon", DisplayName: "VMware Photon OS", Description: "Optimized for VMware and containers", Category: "specialized", Tags: []string{"vmware", "container"}, Popular: false},
		{Name: "busybox", DisplayName: "BusyBox", Description: "Ultra-minimal Unix utilities", Category: "specialized", Tags: []string{"minimal", "embedded"}, Popular: false},
	}
}

// GetPopularImages returns only the popular/recommended images
func GetPopularImages() []ImageMetadata {
	all := GetImageMetadata()
	var popular []ImageMetadata
	for _, img := range all {
		if img.Popular {
			popular = append(popular, img)
		}
	}
	return popular
}

// GetImagesByCategory returns images grouped by category
func GetImagesByCategory() map[string][]ImageMetadata {
	all := GetImageMetadata()
	categories := make(map[string][]ImageMetadata)
	for _, img := range all {
		categories[img.Category] = append(categories[img.Category], img)
	}
	return categories
}

// IsCustomImageSupported checks if we have a custom rexec image for this type
func IsCustomImageSupported(imageType string) bool {
	_, ok := CustomImages[imageType]
	return ok
}

// GetImageName returns the best available image for the given type
// Prefers custom rexec images if they exist, otherwise uses base images
func GetImageName(imageType string) string {
	// Check if custom image exists
	if customImage, ok := CustomImages[imageType]; ok {
		// Check if the custom image exists locally
		if ImageExists(customImage) {
			return customImage
		}
	}

	// Return base image
	if baseImage, ok := SupportedImages[imageType]; ok {
		return baseImage
	}

	return ""
}

// ImageExists checks if a Docker image exists locally
func ImageExists(imageName string) bool {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, err = cli.ImageInspectWithRaw(ctx, imageName)
	return err == nil
}

// ValidateCustomImage validates that a custom Docker image exists and is pullable
func ValidateCustomImage(ctx context.Context, imageName string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	// First check if image exists locally
	_, _, err = cli.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		return nil // Image exists locally
	}

	// Try to pull the image
	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("image not found locally and failed to pull: %w", err)
	}
	defer reader.Close()

	// Wait for pull to complete
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed while pulling image: %w", err)
	}

	return nil
}

// ImageShells maps image types to their default shell
var ImageShells = map[string]string{
	"ubuntu":      "/bin/bash",
	"ubuntu-24":   "/bin/bash",
	"ubuntu-20":   "/bin/bash",
	"debian":      "/bin/bash",
	"debian-11":   "/bin/bash",
	"kali":        "/bin/bash",
	"parrot":      "/bin/bash",
	"fedora":      "/bin/bash",
	"fedora-39":   "/bin/bash",
	"centos":      "/bin/bash",
	"rocky":       "/bin/bash",
	"alma":        "/bin/bash",
	"oracle":      "/bin/bash",
	"alpine":      "/bin/sh",
	"alpine-3.18": "/bin/sh",
	"archlinux":   "/bin/bash",
	"opensuse":    "/bin/bash",
	"gentoo":      "/bin/bash",
	"void":        "/bin/bash",
	"nixos":       "/bin/bash",
	"amazonlinux": "/bin/bash",
	"clearlinux":  "/bin/bash",
	"photon":      "/bin/bash",
	"busybox":     "/bin/sh",
}

// ImageFallbackShells provides fallback shells to try if the primary fails
var ImageFallbackShells = []string{"/bin/sh", "/bin/bash", "/bin/ash"}

// ContainerConfig holds configuration for creating a new container
type ContainerConfig struct {
	UserID        string
	ContainerName string            // User-provided name for the container
	ImageType     string            // ubuntu, debian, arch, alpine, or "custom:imagename"
	CustomImage   string            // Custom Docker image name (if ImageType is "custom")
	MemoryLimit   int64             // in bytes, default 512MB
	CPULimit      int64             // CPU quota, default 100000 (1 CPU)
	DiskQuota     int64             // in bytes
	Labels        map[string]string // Custom labels for the container
}

// ContainerInfo holds information about a running container
type ContainerInfo struct {
	ID            string
	UserID        string
	ContainerName string
	ImageType     string
	Status        string
	CreatedAt     time.Time
	LastUsedAt    time.Time
	IPAddress     string
	Labels        map[string]string
}

// Manager handles Docker container lifecycle
type Manager struct {
	client     *client.Client
	containers map[string]*ContainerInfo // dockerID -> container info
	userIndex  map[string][]string       // userID -> list of dockerIDs
	mu         sync.RWMutex
	volumePath string // base path for user volumes
}

// NewManager creates a new container manager
func NewManager(volumePaths ...string) (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to docker daemon: %w", err)
	}

	// Default volume path
	volumePath := "/var/lib/rexec/volumes"
	if len(volumePaths) > 0 && volumePaths[0] != "" {
		volumePath = volumePaths[0]
	}

	return &Manager{
		client:     cli,
		containers: make(map[string]*ContainerInfo),
		userIndex:  make(map[string][]string),
		volumePath: volumePath,
	}, nil
}

// PullImage pulls the specified image if not present
func (m *Manager) PullImage(ctx context.Context, imageType string) error {
	var imageName string

	// Handle custom images
	if imageType == "custom" {
		return fmt.Errorf("custom image type requires CustomImage in config")
	}

	imageName, ok := SupportedImages[imageType]
	if !ok {
		return fmt.Errorf("unsupported image type: %s", imageType)
	}

	// Check if image exists
	_, _, err := m.client.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		return nil // Image already exists
	}

	// Pull the image
	reader, err := m.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Wait for pull to complete
	_, err = io.Copy(io.Discard, reader)
	return err
}

// PullCustomImage pulls a custom Docker image
func (m *Manager) PullCustomImage(ctx context.Context, imageName string) error {
	// Check if image exists
	_, _, err := m.client.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		return nil // Image already exists
	}

	// Pull the image
	reader, err := m.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull custom image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Wait for pull to complete
	_, err = io.Copy(io.Discard, reader)
	return err
}

// PullImageWithProgress pulls an image and sends progress updates to the channel
func (m *Manager) PullImageWithProgress(ctx context.Context, imageType string, progressCh chan<- ProgressEvent) error {
	var imageName string

	// Handle custom images
	if imageType == "custom" {
		return fmt.Errorf("custom image type requires CustomImage in config")
	}

	imageName, ok := SupportedImages[imageType]
	if !ok {
		return fmt.Errorf("unsupported image type: %s", imageType)
	}

	return m.pullImageWithProgressInternal(ctx, imageName, progressCh)
}

// PullCustomImageWithProgress pulls a custom image with progress updates
func (m *Manager) PullCustomImageWithProgress(ctx context.Context, imageName string, progressCh chan<- ProgressEvent) error {
	return m.pullImageWithProgressInternal(ctx, imageName, progressCh)
}

// pullImageWithProgressInternal handles the actual image pull with progress tracking
func (m *Manager) pullImageWithProgressInternal(ctx context.Context, imageName string, progressCh chan<- ProgressEvent) error {
	// Check if image exists
	_, _, err := m.client.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		// Image already exists, send quick progress
		progressCh <- ProgressEvent{
			Stage:    "pulling",
			Message:  "Image already available locally",
			Progress: 100,
			Detail:   imageName,
		}
		return nil
	}

	// Send initial pulling message
	progressCh <- ProgressEvent{
		Stage:    "pulling",
		Message:  fmt.Sprintf("Pulling image %s...", imageName),
		Progress: 0,
		Detail:   "Starting download",
	}

	// Pull the image
	reader, err := m.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Track layer progress
	layerProgress := make(map[string]int64)
	layerTotal := make(map[string]int64)
	scanner := bufio.NewScanner(reader)

	// Increase buffer size for large JSON responses
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		var progress PullProgress
		if err := json.Unmarshal(scanner.Bytes(), &progress); err != nil {
			continue
		}

		// Update layer tracking
		if progress.ID != "" && progress.ProgressDetail.Total > 0 {
			layerProgress[progress.ID] = progress.ProgressDetail.Current
			layerTotal[progress.ID] = progress.ProgressDetail.Total
		}

		// Calculate overall progress
		var totalBytes, downloadedBytes int64
		for id, total := range layerTotal {
			totalBytes += total
			if current, ok := layerProgress[id]; ok {
				downloadedBytes += current
			}
		}

		var percent float64
		if totalBytes > 0 {
			percent = float64(downloadedBytes) / float64(totalBytes) * 100
		}

		// Build detail message
		detail := progress.Status
		if progress.ID != "" {
			detail = fmt.Sprintf("%s: %s", progress.ID[:12], progress.Status)
			if progress.Progress != "" {
				detail = fmt.Sprintf("%s %s", detail, progress.Progress)
			}
		}

		progressCh <- ProgressEvent{
			Stage:    "pulling",
			Message:  fmt.Sprintf("Pulling %s", imageName),
			Progress: percent,
			Detail:   detail,
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading pull progress: %w", err)
	}

	// Send completion message
	progressCh <- ProgressEvent{
		Stage:    "pulling",
		Message:  "Image pull complete",
		Progress: 100,
		Detail:   imageName,
	}

	return nil
}

// CheckImageExists checks if an image exists locally
func (m *Manager) CheckImageExists(ctx context.Context, imageType string, isCustom bool, customImage string) (bool, string) {
	var imageName string

	if isCustom {
		imageName = customImage
	} else {
		var ok bool
		imageName, ok = SupportedImages[imageType]
		if !ok {
			return false, ""
		}
	}

	_, _, err := m.client.ImageInspectWithRaw(ctx, imageName)
	return err == nil, imageName
}

// CreateContainer creates a new container for a user
func (m *Manager) CreateContainer(ctx context.Context, cfg ContainerConfig) (*ContainerInfo, error) {
	var imageName string
	var imageType string

	// Handle custom images
	if cfg.ImageType == "custom" && cfg.CustomImage != "" {
		imageName = cfg.CustomImage
		imageType = "custom:" + cfg.CustomImage
	} else {
		var ok bool
		imageName, ok = SupportedImages[cfg.ImageType]
		if !ok {
			return nil, fmt.Errorf("unsupported image type: %s", cfg.ImageType)
		}
		imageType = cfg.ImageType
	}

	// Set defaults
	if cfg.MemoryLimit == 0 {
		cfg.MemoryLimit = 512 * 1024 * 1024 // 512MB
	}
	if cfg.CPULimit == 0 {
		cfg.CPULimit = 100000 // 1 CPU
	}

	// Generate unique container name: rexec-{userID}-{containerName}
	containerName := fmt.Sprintf("rexec-%s-%s", cfg.UserID, cfg.ContainerName)

	// Use Docker named volume for persistence (works on Mac/Windows/Linux)
	// Each container gets its own volume
	volumeName := fmt.Sprintf("rexec-%s-%s", cfg.UserID, cfg.ContainerName)

	// Get the appropriate shell for this image - use /bin/sh as it's universally available
	shell := ImageShells[cfg.ImageType]
	if shell == "" {
		shell = "/bin/sh" // Default fallback - always available
	}

	// Container configuration
	containerConfig := &container.Config{
		Image:        imageName,
		Cmd:          []string{shell},
		Tty:          true,
		OpenStdin:    true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env: []string{
			"TERM=xterm-256color",
			"SHELL=" + shell,
			fmt.Sprintf("USER_ID=%s", cfg.UserID),
			fmt.Sprintf("CONTAINER_NAME=%s", cfg.ContainerName),
		},
		Labels: mergeLabels(map[string]string{
			"rexec.user_id":        cfg.UserID,
			"rexec.container_name": cfg.ContainerName,
			"rexec.image_type":     imageType,
			"rexec.managed":        "true",
		}, cfg.Labels),
		// Expose SSH port
		ExposedPorts: nat.PortSet{
			"22/tcp": struct{}{},
		},
	}

	// Host configuration with resource limits
	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:   cfg.MemoryLimit,
			CPUQuota: cfg.CPULimit,
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/home/user",
			},
		},
		// Security options
		SecurityOpt: []string{
			"no-new-privileges:true",
		},
		// Restart policy
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		// Port bindings (optional, for future use)
		PortBindings: nat.PortMap{},
	}

	// Network configuration
	networkConfig := &network.NetworkingConfig{}

	// Create the container
	resp, err := m.client.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		networkConfig,
		nil, // platform
		containerName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start the container
	if err := m.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		// Cleanup on failure
		_ = m.client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Get container details
	inspect, err := m.client.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	now := time.Now()
	// Merge default and custom labels for storage
	allLabels := mergeLabels(map[string]string{
		"rexec.user_id":        cfg.UserID,
		"rexec.container_name": cfg.ContainerName,
		"rexec.image_type":     imageType,
		"rexec.managed":        "true",
	}, cfg.Labels)

	info := &ContainerInfo{
		ID:            resp.ID,
		UserID:        cfg.UserID,
		ContainerName: cfg.ContainerName,
		ImageType:     imageType,
		Status:        "running",
		CreatedAt:     now,
		LastUsedAt:    now,
		IPAddress:     inspect.NetworkSettings.IPAddress,
		Labels:        allLabels,
	}

	m.mu.Lock()
	m.containers[resp.ID] = info
	m.userIndex[cfg.UserID] = append(m.userIndex[cfg.UserID], resp.ID)
	m.mu.Unlock()

	return info, nil
}

// GetContainer returns container info by docker ID
func (m *Manager) GetContainer(dockerID string) (*ContainerInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	info, ok := m.containers[dockerID]
	return info, ok
}

// GetContainerByUserID returns a single container for backward compatibility
// Returns the first container found for the user
func (m *Manager) GetContainerByUserID(userID string) (*ContainerInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dockerIDs, ok := m.userIndex[userID]
	if !ok || len(dockerIDs) == 0 {
		return nil, false
	}

	info, ok := m.containers[dockerIDs[0]]
	return info, ok
}

// GetUserContainers returns all containers for a user
func (m *Manager) GetUserContainers(userID string) []*ContainerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dockerIDs, ok := m.userIndex[userID]
	if !ok {
		return nil
	}

	result := make([]*ContainerInfo, 0, len(dockerIDs))
	for _, dockerID := range dockerIDs {
		if info, ok := m.containers[dockerID]; ok {
			result = append(result, info)
		}
	}
	return result
}

// StopContainer stops a container by docker ID
func (m *Manager) StopContainer(ctx context.Context, dockerID string) error {
	m.mu.RLock()
	info, ok := m.containers[dockerID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("container not found: %s", dockerID)
	}

	timeout := 10 // seconds
	if err := m.client.ContainerStop(ctx, dockerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	m.mu.Lock()
	info.Status = "stopped"
	m.mu.Unlock()

	return nil
}

// StopContainerByUserID stops the first container for a user (backward compatibility)
func (m *Manager) StopContainerByUserID(ctx context.Context, userID string) error {
	m.mu.RLock()
	dockerIDs, ok := m.userIndex[userID]
	m.mu.RUnlock()

	if !ok || len(dockerIDs) == 0 {
		return fmt.Errorf("no containers found for user: %s", userID)
	}

	return m.StopContainer(ctx, dockerIDs[0])
}

// StartContainer starts a stopped container by docker ID
func (m *Manager) StartContainer(ctx context.Context, dockerID string) error {
	m.mu.RLock()
	info, ok := m.containers[dockerID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("container not found: %s", dockerID)
	}

	if err := m.client.ContainerStart(ctx, dockerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	m.mu.Lock()
	info.Status = "running"
	info.LastUsedAt = time.Now()
	m.mu.Unlock()

	return nil
}

// StartContainerByUserID starts the first container for a user (backward compatibility)
func (m *Manager) StartContainerByUserID(ctx context.Context, userID string) error {
	m.mu.RLock()
	dockerIDs, ok := m.userIndex[userID]
	m.mu.RUnlock()

	if !ok || len(dockerIDs) == 0 {
		return fmt.Errorf("no containers found for user: %s", userID)
	}

	return m.StartContainer(ctx, dockerIDs[0])
}

// RemoveContainer removes a container by docker ID
func (m *Manager) RemoveContainer(ctx context.Context, dockerID string) error {
	m.mu.Lock()
	info, ok := m.containers[dockerID]
	if ok {
		delete(m.containers, dockerID)
		// Remove from user index
		if dockerIDs, exists := m.userIndex[info.UserID]; exists {
			newIDs := make([]string, 0, len(dockerIDs)-1)
			for _, id := range dockerIDs {
				if id != dockerID {
					newIDs = append(newIDs, id)
				}
			}
			if len(newIDs) > 0 {
				m.userIndex[info.UserID] = newIDs
			} else {
				delete(m.userIndex, info.UserID)
			}
		}
	}
	m.mu.Unlock()

	// Remove from Docker
	return m.client.ContainerRemove(ctx, dockerID, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: false, // Keep volumes for data persistence
	})
}

// RemoveContainerByUserID removes the first container for a user (backward compatibility)
func (m *Manager) RemoveContainerByUserID(ctx context.Context, userID string) error {
	m.mu.RLock()
	dockerIDs, ok := m.userIndex[userID]
	m.mu.RUnlock()

	if !ok || len(dockerIDs) == 0 {
		return fmt.Errorf("no containers found for user: %s", userID)
	}

	return m.RemoveContainer(ctx, dockerIDs[0])
}

// ListContainers returns all managed containers
func (m *Manager) ListContainers() []*ContainerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*ContainerInfo, 0, len(m.containers))
	for _, info := range m.containers {
		result = append(result, info)
	}
	return result
}

// CountUserContainers returns the number of containers for a user
func (m *Manager) CountUserContainers(userID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.userIndex[userID])
}

// TouchContainer updates the last used timestamp for a container (alias for backward compatibility)
func (m *Manager) TouchContainer(dockerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if info, ok := m.containers[dockerID]; ok {
		info.LastUsedAt = time.Now()
	}
}

// GetClient returns the underlying Docker client (for advanced operations)
func (m *Manager) GetClient() *client.Client {
	return m.client
}

// GetIdleContainers returns containers that have been idle for longer than the threshold
func (m *Manager) GetIdleContainers(threshold time.Duration) []*ContainerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	result := make([]*ContainerInfo, 0)

	for _, info := range m.containers {
		if info.Status == "running" && now.Sub(info.LastUsedAt) > threshold {
			result = append(result, info)
		}
	}

	return result
}

// Close closes the Docker client
func (m *Manager) Close() error {
	return m.client.Close()
}

// LoadExistingContainers loads rexec-managed containers from Docker
func (m *Manager) LoadExistingContainers(ctx context.Context) error {
	containers, err := m.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, c := range containers {
		// Check if this is a rexec-managed container
		if c.Labels["rexec.managed"] != "true" {
			continue
		}

		userID := c.Labels["rexec.user_id"]
		containerName := c.Labels["rexec.container_name"]
		imageType := c.Labels["rexec.image_type"]

		if userID == "" || containerName == "" {
			continue
		}

		// Determine status
		status := "stopped"
		if c.State == "running" {
			status = "running"
		}

		// Get IP address
		ipAddress := ""
		if c.NetworkSettings != nil {
			for _, network := range c.NetworkSettings.Networks {
				ipAddress = network.IPAddress
				break
			}
		}

		info := &ContainerInfo{
			ID:            c.ID,
			UserID:        userID,
			ContainerName: containerName,
			ImageType:     imageType,
			Status:        status,
			CreatedAt:     time.Unix(c.Created, 0),
			LastUsedAt:    time.Now(),
			IPAddress:     ipAddress,
		}

		m.containers[c.ID] = info
		m.userIndex[userID] = append(m.userIndex[userID], c.ID)
	}

	return nil
}

// mergeLabels merges two label maps, with custom labels taking precedence
func mergeLabels(base, custom map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range custom {
		result[k] = v
	}
	return result
}

// UserContainerLimit returns the max containers allowed for a tier
func UserContainerLimit(tier string) int {
	switch tier {
	case "pro":
		return 5
	case "enterprise":
		return 20
	case "guest":
		return 1 // Guest users limited to 1 container
	default: // free (authenticated users)
		return 2
	}
}
