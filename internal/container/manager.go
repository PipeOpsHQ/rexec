package container

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// ProgressEvent represents a progress update during container creation
type ProgressEvent struct {
	Stage       string                 `json:"stage"`                  // "validating", "pulling", "creating", "starting", "ready"
	Message     string                 `json:"message"`                // Human-readable message
	Progress    float64                `json:"progress"`               // 0-100 percentage
	Detail      string                 `json:"detail"`                 // Additional detail (e.g., layer being pulled)
	Error       string                 `json:"error,omitempty"`        // Error message if failed
	Complete    bool                   `json:"complete"`               // Whether the whole process is complete
	ContainerID string                 `json:"container_id,omitempty"` // Set when complete
	Container   map[string]interface{} `json:"container,omitempty"`    // Full container data when complete
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

// formatBytes converts bytes to human-readable format (e.g., 512MB, 2GB)
func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "2G" // default
	}
	const gb = 1024 * 1024 * 1024
	const mb = 1024 * 1024
	if bytes >= gb {
		return fmt.Sprintf("%dG", bytes/gb)
	}
	return fmt.Sprintf("%dM", bytes/mb)
}

// SanitizeError removes sensitive information from error messages before showing to users.
// This prevents leaking Docker host IPs, ports, and internal infrastructure details.
func SanitizeError(err error) string {
	if err == nil {
		return ""
	}
	return SanitizeErrorString(err.Error())
}

// SanitizeErrorString removes sensitive information from error strings.
func SanitizeErrorString(errMsg string) string {
	if errMsg == "" {
		return ""
	}

	// Patterns to remove (Docker host details, IPs, ports)
	// Match: "tcp://IP:PORT" or "unix://path"
	// Match: "Cannot connect to the Docker daemon at ..."
	// Match: "Is the docker daemon running?"

	// Replace Docker connection errors with generic message
	if strings.Contains(errMsg, "Cannot connect to the Docker daemon") ||
		strings.Contains(errMsg, "Is the docker daemon running") ||
		strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "tcp://") {
		return "Container service temporarily unavailable. Please try again."
	}

	// Remove any tcp:// or unix:// URLs with IP/port
	// Pattern: tcp://X.X.X.X:PORT or tcp://hostname:PORT
	result := errMsg
	for {
		tcpIdx := strings.Index(result, "tcp://")
		if tcpIdx == -1 {
			break
		}
		// Find the end of the URL (space, quote, or end of string)
		endIdx := tcpIdx + 6 // after "tcp://"
		for endIdx < len(result) && result[endIdx] != ' ' && result[endIdx] != '"' && result[endIdx] != '\'' && result[endIdx] != ')' {
			endIdx++
		}
		result = result[:tcpIdx] + "[docker-host]" + result[endIdx:]
	}

	// Remove unix:// paths too
	for {
		unixIdx := strings.Index(result, "unix://")
		if unixIdx == -1 {
			break
		}
		endIdx := unixIdx + 7
		for endIdx < len(result) && result[endIdx] != ' ' && result[endIdx] != '"' && result[endIdx] != '\'' && result[endIdx] != ')' {
			endIdx++
		}
		result = result[:unixIdx] + "[docker-socket]" + result[endIdx:]
	}

	return result
}

// SupportedImages maps user-friendly names to Docker images
// Uses custom rexec images if available (with SSH pre-installed), otherwise falls back to base images
// NOTE: Only verified working images are included here
var SupportedImages = map[string]string{
	// Debian-based (Updated Dec 2025)
	"ubuntu":     "ubuntu:24.04",    // LTS until 2029, latest security patches
	"ubuntu-24":  "ubuntu:24.04",    // Noble Numbat LTS
	"ubuntu-22":  "ubuntu:22.04",    // Jammy Jellyfish LTS
	"ubuntu-20":  "ubuntu:20.04",    // Focal Fossa LTS (EOL Apr 2025, still supported)
	"debian":     "debian:12",       // Bookworm (current stable)
	"debian-12":  "debian:12",       // Bookworm
	"debian-11":  "debian:11",       // Bullseye (oldstable)
	"kali":       "kalilinux/kali-rolling:latest",
	"parrot":     "parrotsec/core:latest",
	"mint":       "linuxmintd/mint22-amd64:latest", // Linux Mint 22 (latest)
	"elementary": "elementary/docker:stable",
	"devuan":     "devuan/devuan:excalibur", // Devuan 5.0
	// Security & Penetration Testing
	"blackarch":       "blackarchlinux/blackarch:latest",
	"parrot-security": "parrotsec/security:latest",
	// Red Hat-based (Updated Dec 2025)
	"fedora":    "fedora:41",       // Fedora 41 (latest stable)
	"fedora-40": "fedora:40",       // Previous stable
	"fedora-39": "fedora:39",       // Older stable
	"centos":    "quay.io/centos/centos:stream9", // CentOS Stream 9
	"centos-stream": "quay.io/centos/centos:stream9",
	"rocky":     "rockylinux:9.5",  // Rocky Linux 9.5 (latest)
	"rocky-8":   "rockylinux:8.10", // Rocky Linux 8.10
	"alma":      "almalinux:9.5",   // AlmaLinux 9.5 (latest)
	"alma-8":    "almalinux:8.10",  // AlmaLinux 8.10
	"oracle":    "oraclelinux:9",   // Oracle Linux 9
	"rhel":      "redhat/ubi9:latest", // Red Hat UBI 9
	"openeuler": "openeuler/openeuler:24.03-lts",
	// Arch-based
	"archlinux": "archlinux:latest", // Rolling release
	"manjaro":   "manjarolinux/base:latest",
	"artix":     "artixlinux/artixlinux:latest", // Arch without systemd
	// SUSE-based (Updated Dec 2025)
	"opensuse":   "opensuse/leap:15.6",     // openSUSE Leap 15.6
	"tumbleweed": "opensuse/tumbleweed:latest", // Rolling release
	"mageia":     "mageia:9",               // Mandriva fork
	// Independent Distributions
	"gentoo":    "gentoo/stage3:latest",
	"void":      "voidlinux/voidlinux:latest",
	"nixos":     "nixos/nix:latest",
	"slackware": "aclemons/slackware:15.0",
	"crux":      "crux/crux:latest",
	"guix":      "gnu/guix:latest",
	// Minimal / Embedded (Updated Dec 2025)
	"alpine":      "alpine:3.21",   // Alpine 3.21 (latest stable)
	"alpine-3.20": "alpine:3.20",   // Previous stable
	"alpine-3.18": "alpine:3.18",   // Older stable
	"busybox":     "busybox:1.37",  // Latest busybox
	"tinycore":    "tatocaster/tinycore:latest",
	"openwrt":     "openwrt/rootfs:latest",
	// Container / Cloud Optimized
	"rancheros": "alpine:3.21", // RancherOS discontinued, using Alpine as lightweight alternative
	// Cloud Provider Specific (Updated Dec 2025)
	"amazonlinux":     "amazonlinux:2023", // Amazon Linux 2023 latest
	"amazonlinux2":    "amazonlinux:2",      // Amazon Linux 2 (EOL 2025)
	"oracle-slim":     "oraclelinux:9-slim",
	"azurelinux":      "mcr.microsoft.com/azurelinux/base/core:3.0",
	// Scientific
	"scientific":  "scientificlinux/sl:latest",
	"neurodebian": "neurodebian:bookworm",
	// Specialized
	"clearlinux": "clearlinux:latest",
	"photon":     "photon:5.0",     // VMware Photon OS 5.0
	// Raspberry Pi / ARM
	"raspberrypi": "balenalib/raspberry-pi-debian:bookworm",
	// macOS (VM-based)
	"macos": "sickcodes/docker-osx:latest",
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
// NOTE: Only includes verified working Docker images - Updated Dec 2025
func GetImageMetadata() []ImageMetadata {
	return []ImageMetadata{
		// Debian-based (Popular) - Updated Dec 2025
		{Name: "ubuntu", DisplayName: "Ubuntu 24.04 LTS", Description: "Latest Ubuntu LTS with best-in-class security", Category: "debian", Tags: []string{"lts", "popular", "beginner-friendly"}, Popular: true},
		{Name: "ubuntu-24", DisplayName: "Ubuntu 24.04 LTS", Description: "Noble Numbat - Latest Ubuntu LTS release", Category: "debian", Tags: []string{"lts", "latest"}, Popular: true},
		{Name: "ubuntu-22", DisplayName: "Ubuntu 22.04 LTS", Description: "Jammy Jellyfish - Stable and well-tested", Category: "debian", Tags: []string{"lts", "stable"}, Popular: false},
		{Name: "ubuntu-20", DisplayName: "Ubuntu 20.04 LTS", Description: "Focal Fossa - Long-term support until 2025", Category: "debian", Tags: []string{"lts", "legacy"}, Popular: false},
		{Name: "debian", DisplayName: "Debian 12 (Bookworm)", Description: "Rock-solid stability with extensive packages", Category: "debian", Tags: []string{"stable", "server"}, Popular: true},
		{Name: "debian-12", DisplayName: "Debian 12 (Bookworm)", Description: "Current stable Debian release", Category: "debian", Tags: []string{"stable"}, Popular: false},
		{Name: "debian-11", DisplayName: "Debian 11 (Bullseye)", Description: "Previous stable Debian release", Category: "debian", Tags: []string{"oldstable"}, Popular: false},
		{Name: "mint", DisplayName: "Linux Mint 22", Description: "User-friendly Ubuntu derivative with Cinnamon", Category: "debian", Tags: []string{"desktop", "beginner-friendly"}, Popular: true},
		{Name: "elementary", DisplayName: "elementary OS", Description: "Beautiful and privacy-focused desktop OS", Category: "debian", Tags: []string{"desktop", "beautiful"}, Popular: false},
		{Name: "devuan", DisplayName: "Devuan 5 (Excalibur)", Description: "Debian without systemd - init freedom", Category: "debian", Tags: []string{"init-freedom", "advanced"}, Popular: false},

		// Security & Penetration Testing
		{Name: "kali", DisplayName: "Kali Linux", Description: "Industry-standard penetration testing distro", Category: "security", Tags: []string{"security", "pentest", "hacking"}, Popular: true},
		{Name: "parrot", DisplayName: "Parrot OS", Description: "Security-focused with privacy tools", Category: "security", Tags: []string{"security", "privacy", "pentest"}, Popular: true},
		{Name: "parrot-security", DisplayName: "Parrot Security", Description: "Full Parrot security edition with all tools", Category: "security", Tags: []string{"security", "pentest", "full"}, Popular: false},
		{Name: "blackarch", DisplayName: "BlackArch Linux", Description: "Arch-based with 2900+ security tools", Category: "security", Tags: []string{"security", "pentest", "arch"}, Popular: false},

		// Red Hat-based - Updated Dec 2025
		{Name: "fedora", DisplayName: "Fedora 41", Description: "Latest Fedora with cutting-edge features", Category: "redhat", Tags: []string{"modern", "rhel-upstream", "latest"}, Popular: true},
		{Name: "fedora-40", DisplayName: "Fedora 40", Description: "Previous stable Fedora release", Category: "redhat", Tags: []string{"stable"}, Popular: false},
		{Name: "fedora-39", DisplayName: "Fedora 39", Description: "Older Fedora release", Category: "redhat", Tags: []string{"stable"}, Popular: false},
		{Name: "centos", DisplayName: "CentOS Stream 9", Description: "Upstream for RHEL, community-driven", Category: "redhat", Tags: []string{"enterprise", "rhel-upstream"}, Popular: true},
		{Name: "centos-stream", DisplayName: "CentOS Stream 9", Description: "Rolling preview of future RHEL", Category: "redhat", Tags: []string{"enterprise", "rolling"}, Popular: false},
		{Name: "rocky", DisplayName: "Rocky Linux 9.5", Description: "Enterprise-grade 1:1 RHEL-compatible", Category: "redhat", Tags: []string{"enterprise", "rhel-compatible"}, Popular: true},
		{Name: "rocky-8", DisplayName: "Rocky Linux 8.10", Description: "Rocky Linux 8 branch - stable and tested", Category: "redhat", Tags: []string{"enterprise", "stable"}, Popular: false},
		{Name: "alma", DisplayName: "AlmaLinux 9.5", Description: "Community-driven RHEL fork with long support", Category: "redhat", Tags: []string{"enterprise", "rhel-compatible"}, Popular: true},
		{Name: "alma-8", DisplayName: "AlmaLinux 8.10", Description: "AlmaLinux 8 branch - stable enterprise", Category: "redhat", Tags: []string{"enterprise", "stable"}, Popular: false},
		{Name: "oracle", DisplayName: "Oracle Linux 9", Description: "Oracle's enterprise Linux with Ksplice", Category: "redhat", Tags: []string{"enterprise", "oracle"}, Popular: false},
		{Name: "rhel", DisplayName: "Red Hat UBI 9", Description: "Official Red Hat Universal Base Image", Category: "redhat", Tags: []string{"enterprise", "commercial"}, Popular: true},
		{Name: "openeuler", DisplayName: "openEuler 24.03 LTS", Description: "Enterprise Linux from Huawei", Category: "redhat", Tags: []string{"enterprise", "lts"}, Popular: false},

		// Arch-based
		{Name: "archlinux", DisplayName: "Arch Linux", Description: "Rolling release with latest packages and AUR", Category: "arch", Tags: []string{"rolling", "bleeding-edge", "aur"}, Popular: true},
		{Name: "manjaro", DisplayName: "Manjaro", Description: "User-friendly Arch with curated updates", Category: "arch", Tags: []string{"rolling", "beginner-friendly"}, Popular: true},
		{Name: "artix", DisplayName: "Artix Linux", Description: "Arch without systemd (OpenRC)", Category: "arch", Tags: []string{"rolling", "init-freedom"}, Popular: false},

		// SUSE-based - Updated Dec 2025
		{Name: "opensuse", DisplayName: "openSUSE Leap 15.6", Description: "Stable enterprise-grade openSUSE", Category: "suse", Tags: []string{"enterprise", "stable", "zypper"}, Popular: true},
		{Name: "tumbleweed", DisplayName: "openSUSE Tumbleweed", Description: "Rolling release with tested updates", Category: "suse", Tags: []string{"rolling", "tested"}, Popular: false},
		{Name: "mageia", DisplayName: "Mageia 9", Description: "Community-driven Mandriva fork", Category: "suse", Tags: []string{"rpm", "desktop", "stable"}, Popular: false},

		// Independent Distributions
		{Name: "gentoo", DisplayName: "Gentoo Linux", Description: "Source-based with extreme customization", Category: "independent", Tags: []string{"source-based", "advanced", "performance"}, Popular: false},
		{Name: "void", DisplayName: "Void Linux", Description: "Independent distro with runit init system", Category: "independent", Tags: []string{"independent", "runit", "rolling"}, Popular: false},
		{Name: "nixos", DisplayName: "NixOS", Description: "Declarative configuration and reproducible builds", Category: "independent", Tags: []string{"declarative", "nix", "reproducible"}, Popular: false},
		{Name: "slackware", DisplayName: "Slackware 15.0", Description: "Oldest maintained Linux distro, Unix-like", Category: "independent", Tags: []string{"classic", "stable", "unix-like"}, Popular: false},
		{Name: "crux", DisplayName: "CRUX 3.7", Description: "Lightweight, BSD-style init scripts", Category: "independent", Tags: []string{"lightweight", "bsd-style", "simple"}, Popular: false},
		{Name: "guix", DisplayName: "Guix System", Description: "Advanced transactional package manager", Category: "independent", Tags: []string{"functional", "gnu", "scheme"}, Popular: false},

		// Minimal / Embedded - Updated Dec 2025
		{Name: "alpine", DisplayName: "Alpine 3.21", Description: "Lightweight and security-oriented (6MB)", Category: "minimal", Tags: []string{"minimal", "docker", "security"}, Popular: true},
		{Name: "alpine-3.20", DisplayName: "Alpine 3.20", Description: "Previous stable Alpine release", Category: "minimal", Tags: []string{"minimal", "stable"}, Popular: false},
		{Name: "alpine-3.18", DisplayName: "Alpine 3.18", Description: "Older stable Alpine release", Category: "minimal", Tags: []string{"minimal", "legacy"}, Popular: false},
		{Name: "busybox", DisplayName: "BusyBox 1.37", Description: "Ultra-minimal Unix utilities (~2MB)", Category: "minimal", Tags: []string{"minimal", "embedded"}, Popular: false},
		{Name: "tinycore", DisplayName: "TinyCore Linux", Description: "The smallest subset of Linux (~16MB)", Category: "minimal", Tags: []string{"micro", "fast", "ram-only"}, Popular: false},
		{Name: "openwrt", DisplayName: "OpenWrt 23.05", Description: "Embedded operating system for routers", Category: "minimal", Tags: []string{"network", "embedded", "router"}, Popular: false},

		// Container / Cloud Optimized
		{Name: "rancheros", DisplayName: "RancherOS (Alpine)", Description: "Lightweight container-optimized OS (Alpine-based)", Category: "container", Tags: []string{"containers", "docker", "minimal"}, Popular: false},

		// Cloud Provider Specific - Updated Dec 2025
		{Name: "amazonlinux", DisplayName: "Amazon Linux 2023", Description: "Latest Amazon Linux optimized for AWS", Category: "cloud", Tags: []string{"aws", "cloud", "enterprise"}, Popular: true},
		{Name: "amazonlinux2", DisplayName: "Amazon Linux 2", Description: "LTS Amazon Linux (EOL Jun 2025)", Category: "cloud", Tags: []string{"aws", "cloud", "legacy"}, Popular: false},
		{Name: "oracle-slim", DisplayName: "Oracle Linux 9 Slim", Description: "Lightweight Oracle Linux", Category: "cloud", Tags: []string{"oracle", "cloud", "minimal"}, Popular: false},
		{Name: "azurelinux", DisplayName: "Azure Linux 3.0", Description: "Microsoft's OS for Azure Kubernetes Service", Category: "cloud", Tags: []string{"azure", "microsoft", "cloud"}, Popular: false},

		// Scientific
		{Name: "scientific", DisplayName: "Scientific Linux", Description: "For scientific computing and research", Category: "developer", Tags: []string{"scientific", "research", "rhel"}, Popular: false},
		{Name: "neurodebian", DisplayName: "NeuroDebian", Description: "Neuroscience-oriented Debian", Category: "developer", Tags: []string{"science", "brain", "research"}, Popular: false},

		// Specialized
		{Name: "clearlinux", DisplayName: "Clear Linux", Description: "Intel-optimized for maximum performance", Category: "specialized", Tags: []string{"performance", "intel", "cloud"}, Popular: false},
		{Name: "photon", DisplayName: "VMware Photon OS 5.0", Description: "Optimized for VMware and containers", Category: "specialized", Tags: []string{"vmware", "container", "minimal"}, Popular: false},

		// Raspberry Pi / ARM
		{Name: "raspberrypi", DisplayName: "Raspberry Pi OS", Description: "Debian-based OS for Raspberry Pi/ARM", Category: "embedded", Tags: []string{"raspberry-pi", "arm", "iot"}, Popular: false},

		// macOS
		{Name: "macos", DisplayName: "macOS", Description: "Apple macOS in a VM container", Category: "macos", Tags: []string{"macos", "apple", "vm"}, Popular: true},
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
	// Debian-based
	"ubuntu":     "/bin/bash",
	"ubuntu-24":  "/bin/bash",
	"ubuntu-22":  "/bin/bash",
	"ubuntu-20":  "/bin/bash",
	"debian":     "/bin/bash",
	"debian-12":  "/bin/bash",
	"debian-11":  "/bin/bash",
	"kali":       "/bin/bash",
	"parrot":     "/bin/bash",
	"mint":       "/bin/bash",
	"popos":      "/bin/bash",
	"elementary": "/bin/bash",
	"zorin":      "/bin/bash",
	"mxlinux":    "/bin/bash",
	"devuan":     "/bin/bash",
	"antix":      "/bin/bash",
	// Security
	"blackarch":       "/bin/bash",
	"parrot-security": "/bin/bash",
	"backbox":         "/bin/bash",
	"dracos":          "/bin/bash",
	"pentoo":          "/bin/bash",
	"samurai":         "/bin/bash",
	"kali-purple":     "/bin/bash",
	// Red Hat-based
	"fedora":        "/bin/bash",
	"fedora-41":     "/bin/bash",
	"fedora-40":     "/bin/bash",
	"fedora-39":     "/bin/bash",
	"centos":        "/bin/bash",
	"centos-stream": "/bin/bash",
	"rocky":         "/bin/bash",
	"rocky-8":       "/bin/bash",
	"alma":          "/bin/bash",
	"alma-8":        "/bin/bash",
	"oracle":        "/bin/bash",
	"oracle-slim":   "/bin/bash",
	"rhel":          "/bin/bash",
	"openeuler":     "/bin/bash",
	"springdale":    "/bin/bash",
	"navy":          "/bin/bash",
	// Arch-based
	"archlinux":   "/bin/bash",
	"manjaro":     "/bin/bash",
	"endeavouros": "/bin/bash",
	"garuda":      "/bin/bash",
	"arcolinux":   "/bin/bash",
	"artix":       "/bin/bash",
	// SUSE-based
	"opensuse":   "/bin/bash",
	"tumbleweed": "/bin/bash",
	"sles":       "/bin/bash",
	"mageia":     "/bin/bash",
	// Independent
	"gentoo":    "/bin/bash",
	"void":      "/bin/bash",
	"nixos":     "/bin/sh",
	"slackware": "/bin/bash",
	"solus":     "/bin/bash",
	"pclinuxos": "/bin/bash",
	"crux":      "/bin/bash",
	"guix":      "/bin/bash",
	// Minimal (use sh)
	"alpine":      "/bin/sh",
	"alpine-3.21": "/bin/sh",
	"alpine-3.20": "/bin/sh",
	"alpine-3.18": "/bin/sh",
	"tinycore":    "/bin/sh",
	"puppy":       "/bin/sh",
	"dsl":         "/bin/sh",
	"busybox":     "/bin/sh",
	"openwrt":     "/bin/ash",
	// Container optimized
	"flatcar":      "/bin/bash",
	"rancheros":    "/bin/sh", // Alpine-based, uses sh
	"bottlerocket": "/bin/bash",
	"talos":        "/bin/sh",
	"k3os":         "/bin/sh",
	// BSD (use sh as default)
	"freebsd":      "/bin/sh",
	"openbsd":      "/bin/sh",
	"netbsd":       "/bin/sh",
	"dragonflybsd": "/bin/sh",
	// Special Purpose
	"qubes":       "/bin/bash",
	"tails":       "/bin/bash",
	"whonix":      "/bin/bash",
	"raspbian":    "/bin/bash",
	"raspberrypi": "/bin/bash",
	"ubuntucore":  "/bin/bash",
	// Gaming
	"steamos":   "/bin/bash",
	"chimeraos": "/bin/bash",
	// Developer
	"ubuntustudio": "/bin/bash",
	"scientific":   "/bin/bash",
	"neurodebian":  "/bin/bash",
	// Cloud Provider Specific
	"amazonlinux":      "/bin/bash",
	"amazonlinux2":     "/bin/bash",
	"cos":              "/bin/bash",
	"azurelinux":       "/bin/bash",
	"oracleautonomous": "/bin/bash",
	"alibabacloud":     "/bin/bash",
	"ibmcloud":         "/bin/bash",
	"digitalocean":     "/bin/bash",
	// Specialized
	"clearlinux": "/bin/bash",
	"photon":     "/bin/bash",
	// macOS
	"macos": "/bin/bash",
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
	client            *client.Client
	containers        map[string]*ContainerInfo // dockerID -> container info
	userIndex         map[string][]string       // userID -> list of dockerIDs
	mu                sync.RWMutex
	volumePath        string // base path for user volumes
	diskQuotaEnabled  bool   // whether disk quota is available
	diskQuotaChecked  bool   // whether we've checked for disk quota support
	diskQuotaCheckMu  sync.Once
}

// NewManager creates a new container manager
// Supports local Docker socket or remote Docker host via DOCKER_HOST environment variable.
// For remote connections, set:
//   - DOCKER_HOST=tcp://host:2377 (Docker TLS) or tcp://host:2378 (Podman TLS)
//   - DOCKER_TLS_VERIFY=1 (for TLS connections)
//   - DOCKER_CERT_PATH=/path/to/certs (for TLS connections)
//
// Note: Rexec uses non-standard ports (2377/2378) instead of 2376 for security.
//
// Works with both Docker and Podman (Podman implements Docker's API).
// Set CONTAINER_RUNTIME=podman to enable Podman-specific features.
func NewManager(volumePaths ...string) (*Manager, error) {
	dockerHost := os.Getenv("DOCKER_HOST")
	containerRuntime := os.Getenv("CONTAINER_RUNTIME") // "docker" or "podman"

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		if dockerHost != "" {
			return nil, fmt.Errorf("failed to create docker client for remote host %s: %w", dockerHost, err)
		}
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// Test connection with longer timeout for remote hosts
	timeout := 5 * time.Second
	if dockerHost != "" {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err = cli.Ping(ctx)
	if err != nil {
		cli.Close()
		if dockerHost != "" {
			// Provide helpful error message for remote Docker/Podman host issues
			runtimeName := "Docker"
			if containerRuntime == "podman" {
				runtimeName = "Podman"
			}
			errMsg := fmt.Sprintf("failed to connect to remote %s daemon at %s", runtimeName, dockerHost)
			// Check for TLS ports (standard 2376 or our custom 2377/2378)
			if strings.Contains(dockerHost, ":2376") || strings.Contains(dockerHost, ":2377") || strings.Contains(dockerHost, ":2378") {
				errMsg += " (TLS enabled - check DOCKER_TLS_VERIFY and DOCKER_CERT_PATH)"
			} else if strings.Contains(dockerHost, "ssh://") {
				errMsg += " (SSH connection - check SSH_PRIVATE_KEY and host accessibility)"
			}
			return nil, fmt.Errorf("%s: %w", errMsg, err)
		}
		return nil, fmt.Errorf("failed to connect to container daemon: %w", err)
	}

	// Log runtime info
	if containerRuntime == "podman" {
		log.Printf("[Container] Using Podman runtime at %s", dockerHost)
	} else if dockerHost != "" {
		log.Printf("[Container] Using Docker runtime at %s", dockerHost)
	}

	// Default volume path
	volumePath := "/var/lib/rexec/volumes"
	if len(volumePaths) > 0 && volumePaths[0] != "" {
		volumePath = volumePaths[0]
	}

	mgr := &Manager{
		client:           cli,
		containers:       make(map[string]*ContainerInfo),
		userIndex:        make(map[string][]string),
		volumePath:       volumePath,
		diskQuotaEnabled: false,
		diskQuotaChecked: false,
	}

	// Check disk quota availability asynchronously
	go mgr.checkDiskQuotaSupport()

	return mgr, nil
}

// checkDiskQuotaSupport checks if disk quotas are available on the Docker host
func (m *Manager) checkDiskQuotaSupport() {
	m.diskQuotaCheckMu.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Try to get Docker info to check storage driver
		info, err := m.client.Info(ctx)
		if err != nil {
			log.Printf("[DiskQuota] Failed to get Docker info: %v", err)
			m.diskQuotaChecked = true
			return
		}

		// Check if storage driver is overlay2 (required for disk quotas)
		if info.Driver != "overlay2" {
			log.Printf("[DiskQuota] Storage driver is %s (not overlay2) - disk quotas disabled", info.Driver)
			m.diskQuotaChecked = true
			return
		}

		// Check driver status for backing filesystem info
		for _, status := range info.DriverStatus {
			if len(status) >= 2 {
				if status[0] == "Backing Filesystem" {
					if status[1] != "xfs" && status[1] != "ext4" {
						log.Printf("[DiskQuota] Backing filesystem is %s (needs xfs or ext4) - disk quotas disabled", status[1])
						m.diskQuotaChecked = true
						return
					}
				}
			}
		}

		// Try creating a test container with storage-opt to verify quota support
		// This is the most reliable way to check if quotas work
		testContainerName := "rexec-quota-test-" + fmt.Sprintf("%d", time.Now().UnixNano())
		testConfig := &container.Config{
			Image: "alpine:latest",
			Cmd:   []string{"true"},
		}
		testHostConfig := &container.HostConfig{
			StorageOpt: map[string]string{
				"size": "100M",
			},
			AutoRemove: true,
		}

		// Try to create a test container
		resp, err := m.client.ContainerCreate(ctx, testConfig, testHostConfig, nil, nil, testContainerName)
		if err != nil {
			if strings.Contains(err.Error(), "storage-opt") || strings.Contains(err.Error(), "pquota") || strings.Contains(err.Error(), "quota") {
				log.Printf("[DiskQuota] Disk quotas not available: %v", err)
				m.diskQuotaChecked = true
				return
			}
			// Other error - assume quotas might work
			log.Printf("[DiskQuota] Test container creation failed (non-quota error): %v", err)
		} else {
			// Clean up test container
			m.client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
			log.Printf("[DiskQuota] Disk quotas are available and working")
			m.diskQuotaEnabled = true
		}

		m.diskQuotaChecked = true
	})
}

// IsDiskQuotaEnabled returns whether disk quotas are available
func (m *Manager) IsDiskQuotaEnabled() bool {
	// Wait for quota check to complete (with timeout)
	for i := 0; i < 50; i++ { // 5 seconds max
		if m.diskQuotaChecked {
			return m.diskQuotaEnabled
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
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
			idDisplay := progress.ID
			if len(idDisplay) > 12 {
				idDisplay = idDisplay[:12]
			}
			detail = fmt.Sprintf("%s: %s", idDisplay, progress.Status)
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
		cfg.CPULimit = 500 // 0.5 CPU in millicores (1000 = 1 CPU)
	}
	
	// Cap CPU to available host CPUs (in millicores)
	maxCPUMillicores := int64(runtime.NumCPU()) * 1000
	if cfg.CPULimit > maxCPUMillicores {
		cfg.CPULimit = maxCPUMillicores
	}

	// Generate unique container name: rexec-{userID}-{containerName}
	containerName := fmt.Sprintf("rexec-%s-%s", cfg.UserID, cfg.ContainerName)

	// Get the appropriate shell for this image - use /bin/sh as it's universally available
	shell := ImageShells[cfg.ImageType]
	if shell == "" {
		shell = "/bin/sh" // Default fallback - always available
	}

	// Generate a short container hostname (first 12 chars of container name hash)
	containerHostname := cfg.ContainerName
	if len(containerHostname) > 12 {
		containerHostname = containerHostname[:12]
	}

	// Container configuration
	containerConfig := &container.Config{
		Image:        imageName,
		Hostname:     containerHostname, // Custom hostname to hide real host
		Domainname:   "rexec.local",     // Custom domain
		Tty:          true,
		OpenStdin:    true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env: []string{
			"HOME=/home/user",
			"TERM=xterm-256color",
			"SHELL=" + shell,
			fmt.Sprintf("USER_ID=%s", cfg.UserID),
			fmt.Sprintf("CONTAINER_NAME=%s", cfg.ContainerName),
			// Override system info environment variables
			"HOSTNAME=" + containerHostname,
			// Pass resource limits for MOTD display
			fmt.Sprintf("REXEC_DISK_QUOTA=%s", formatBytes(cfg.DiskQuota)),
			fmt.Sprintf("REXEC_MEMORY_LIMIT=%s", formatBytes(cfg.MemoryLimit)),
			fmt.Sprintf("REXEC_CPU_LIMIT=%.1f", float64(cfg.CPULimit)/1000.0),
		},
		Labels: mergeLabels(map[string]string{
			"rexec.user_id":        cfg.UserID,
			"rexec.container_name": cfg.ContainerName,
			"rexec.image_type":     imageType,
			"rexec.managed":        "true",
			"rexec.memory_limit":   fmt.Sprintf("%d", cfg.MemoryLimit),
			"rexec.cpu_limit":      fmt.Sprintf("%d", cfg.CPULimit),
			"rexec.disk_quota":     fmt.Sprintf("%d", cfg.DiskQuota),
		}, cfg.Labels),
		// Expose SSH port
		ExposedPorts: nat.PortSet{
			"22/tcp": struct{}{},
		},
	}

	// Special resource handling for macOS (requires more resources)
	if cfg.ImageType == "macos" {
		log.Printf("[Container] Enforcing minimum resources for macOS")
		minMemory := int64(4096 * 1024 * 1024) // 4GB
		minCPU := int64(2000)                  // 2 vCPU
		minDisk := int64(20 * 1024 * 1024 * 1024) // 20GB

		if cfg.MemoryLimit < minMemory {
			cfg.MemoryLimit = minMemory
		}
		if cfg.CPULimit < minCPU {
			cfg.CPULimit = minCPU
		}
		if cfg.DiskQuota < minDisk {
			cfg.DiskQuota = minDisk
		}
	}

	// Host configuration with resource limits and security hardening
	// Use CpuQuota and CpuPeriod for CPU limiting (compatible with all runtimes including gVisor)
	// Note: Cannot use both NanoCPUs and CpuPeriod/CpuQuota - Docker will error
	// CpuPeriod is typically 100000 (100ms), CpuQuota limits CPU time within that period
	// CPULimit is in millicores (500 = 0.5 CPU)
	// For 0.5 CPU: CpuQuota = 50000 (50ms per 100ms period)
	cpuPeriod := int64(100000) // 100ms in microseconds
	cpuQuota := (cfg.CPULimit * cpuPeriod) / 1000 // Convert millicores to quota
	
	// Use default Docker runtime (runc)
	// OCI runtime can be configured via OCI_RUNTIME env var
	// Valid runtimes: "runc" (default), "kata", "kata-fc", "runsc" (gVisor), "runsc-kvm"
	ociRuntime := os.Getenv("OCI_RUNTIME")
	if ociRuntime == "" {
		ociRuntime = "runc" // Default to runc for maximum compatibility
	}

	// Build storage options conditionally based on quota support
	// NOTE: gVisor (runsc) does NOT support overlay2 storage quotas - use runc for disk limits
	storageOpts := make(map[string]string)
	if m.IsDiskQuotaEnabled() && cfg.DiskQuota > 0 {
		storageOpts["size"] = formatBytes(cfg.DiskQuota)
		if strings.HasPrefix(ociRuntime, "runsc") {
			log.Printf("[Container] WARNING: Disk quota (%s) ignored - gVisor doesn't support overlay2 quotas. Use runc for disk limits.", formatBytes(cfg.DiskQuota))
		} else {
			log.Printf("[Container] Disk quota enabled: %s", formatBytes(cfg.DiskQuota))
		}
	} else if cfg.DiskQuota > 0 {
		log.Printf("[Container] Disk quota requested (%s) but quotas not available on host", formatBytes(cfg.DiskQuota))
	}
	
	hostConfig := &container.HostConfig{
		Runtime: ociRuntime, // "runc" (default), "kata", "kata-fc", "runsc" (gVisor), "runsc-kvm"
		Resources: container.Resources{
			Memory:     cfg.MemoryLimit,
			MemorySwap: cfg.MemoryLimit, // Set equal to Memory to disable swap and enforce hard limit
			CPUPeriod:  cpuPeriod,
			CPUQuota:   cpuQuota,
			PidsLimit:  &[]int64{256}[0], // Limit number of processes to prevent fork bombs
		},
		// Storage options for disk quota (requires overlay2 on XFS with pquota mount option)
		StorageOpt: storageOpts,
		// Security options - prevent privilege escalation and add seccomp
		SecurityOpt: []string{
			"no-new-privileges:true",
			"seccomp=unconfined", // TODO: Use custom seccomp profile for tighter control
		},
		// Prevent container from gaining new privileges
		Privileged: false,
		// Mask sensitive host information from /proc and /sys
		MaskedPaths: []string{
			"/proc/acpi",
			"/proc/asound",
			"/proc/kcore",
			"/proc/keys",
			"/proc/latency_stats",
			"/proc/timer_list",
			"/proc/timer_stats",
			"/proc/sched_debug",
			"/proc/scsi",
			"/sys/firmware",
			"/sys/devices/virtual/powercap",
		},
		// Make certain paths read-only to prevent tampering
		ReadonlyPaths: []string{
			"/proc/bus",
			"/proc/fs",
			"/proc/irq",
			"/proc/sys",
			"/proc/sysrq-trigger",
		},
		// Drop all capabilities except minimal required for terminal use
		CapDrop: []string{"ALL"},
		CapAdd: []string{
			"CHOWN",        // Change file ownership
			"DAC_OVERRIDE", // Bypass file permission checks (needed for sudo)
			"FOWNER",       // Bypass permission checks on file owner
			"SETGID",       // Set group ID
			"SETUID",       // Set user ID (needed for su/sudo)
			"KILL",         // Send signals
		},
		// Restart policy
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		// Port bindings (optional, for future use)
		PortBindings: nat.PortMap{},
	}

	// Special handling for macOS (docker-osx)
	if cfg.ImageType == "macos" {
		// Do NOT override Entrypoint - let the VM boot script run
		// Do NOT use /home/user working dir - use default
		log.Printf("[Container] Configuring macOS container (privileged, kvm, default entrypoint)")
		
		hostConfig.Privileged = true
		hostConfig.SecurityOpt = nil // Clear security opts to allow KVM
		hostConfig.CapDrop = nil     // Don't drop caps
		hostConfig.CapAdd = nil      // Allow all caps
		
		// Map /dev/kvm if available
		if _, err := os.Stat("/dev/kvm"); err == nil {
			hostConfig.Devices = []container.DeviceMapping{
				{
					PathOnHost:        "/dev/kvm",
					PathInContainer:   "/dev/kvm",
					CgroupPermissions: "rwm",
				},
			}
		} else {
			log.Printf("[Container] WARNING: /dev/kvm not found. macOS VM will be extremely slow.")
		}
	} else {
		// Standard Linux container setup
		// Create /home/user directory on startup since we're not mounting a volume
		containerConfig.Entrypoint = []string{"/bin/sh", "-c", "mkdir -p /home/user && chmod 777 /home/user && exec " + shell}
		containerConfig.WorkingDir = "/home/user"
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
func (m *Manager) GetContainer(idOrName string) (*ContainerInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// First try lookup by Docker ID (the map key)
	if info, ok := m.containers[idOrName]; ok {
		return info, ok
	}
	
	// Fallback: search by container name (terminal ID like 'parrot-mirgp')
	for _, info := range m.containers {
		if info.ContainerName == idOrName {
			return info, true
		}
	}
	
	return nil, false
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

// RestartContainer restarts a container by docker ID
func (m *Manager) RestartContainer(ctx context.Context, dockerID string) error {
	timeout := 10 // seconds
	return m.client.ContainerRestart(ctx, dockerID, container.StopOptions{
		Timeout: &timeout,
	})
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

// RemoveFromTracking removes a container from the manager's in-memory tracking
// without touching Docker (used by reconciler when container is already gone)
func (m *Manager) RemoveFromTracking(dockerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.containers[dockerID]
	if !ok {
		return
	}

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

// UpdateContainerStatus updates the status of a container in memory
func (m *Manager) UpdateContainerStatus(dockerID string, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if info, ok := m.containers[dockerID]; ok {
		info.Status = status
	}
}

// UpdateContainerResources updates a running container's resource limits via Docker API
// Note: This does NOT work with gVisor runtime - use RecreateContainer instead for gVisor
// Note: Disk quota cannot be changed on a running container
func (m *Manager) UpdateContainerResources(ctx context.Context, dockerID string, memoryMB int64, cpuMillicores int64) error {
	log.Printf("[UpdateContainerResources] Updating container %s: memory=%dMB, cpu=%d millicores", dockerID, memoryMB, cpuMillicores)
	
	// Convert memory from MB to bytes
	memoryBytes := memoryMB * 1024 * 1024
	
	// Use CpuQuota and CpuPeriod for CPU limiting (compatible with all runtimes including gVisor)
	// Note: Cannot use both NanoCPUs and CpuPeriod/CpuQuota - Docker will error
	cpuPeriod := int64(100000) // 100ms in microseconds
	cpuQuota := (cpuMillicores * cpuPeriod) / 1000 // Convert millicores to quota
	
	// Cap CPU to available host CPUs
	maxCPUMillicores := int64(runtime.NumCPU()) * 1000
	if cpuMillicores > maxCPUMillicores {
		log.Printf("[UpdateContainerResources] CPU capped from %d to %d millicores (max host CPUs)", cpuMillicores, maxCPUMillicores)
		cpuQuota = (maxCPUMillicores * cpuPeriod) / 1000
	}

	log.Printf("[UpdateContainerResources] Docker update: memory=%d bytes, cpuPeriod=%d, cpuQuota=%d", memoryBytes, cpuPeriod, cpuQuota)

	// Update the container's resources using Docker API
	updateConfig := container.UpdateConfig{
		Resources: container.Resources{
			Memory:     memoryBytes,
			MemorySwap: memoryBytes, // Set equal to Memory to disable swap and enforce hard limit
			CPUPeriod:  cpuPeriod,
			CPUQuota:   cpuQuota,
		},
	}

	_, err := m.client.ContainerUpdate(ctx, dockerID, updateConfig)
	if err != nil {
		log.Printf("[UpdateContainerResources] Docker API error: %v", err)
		return fmt.Errorf("failed to update container resources: %w", err)
	}

	log.Printf("[UpdateContainerResources] Successfully updated container %s", dockerID)
	return nil
}

// ExecInContainer runs a command inside a running container
func (m *Manager) ExecInContainer(ctx context.Context, dockerID string, cmd []string) error {
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}
	
	execResp, err := m.client.ContainerExecCreate(ctx, dockerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec: %w", err)
	}
	
	// Use ContainerExecAttach for Podman compatibility (attach implicitly starts)
	attachResp, err := m.client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return fmt.Errorf("failed to attach/start exec: %w", err)
	}
	attachResp.Close()
	
	return nil
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
	case "trial", "guest", "free": // Unified 60-day trial experience
		return 5 // Generous trial limit
	case "pro":
		return 10 // Increased pro limit
	case "enterprise":
		return 20
	default:
		return 5 // Default to trial
	}
}

// DockerContainerExists checks if a container exists in Docker by its ID
func (m *Manager) DockerContainerExists(ctx context.Context, dockerID string) bool {
	_, err := m.client.ContainerInspect(ctx, dockerID)
	return err == nil
}

// RecreateContainerConfig holds info needed to recreate a container
type RecreateContainerConfig struct {
	UserID        string
	ContainerName string
	Image         string // Can be "ubuntu-24" or "custom:image/name"
	OldDockerID   string
	Tier          string
	// Optional custom resource limits (if set, these override tier defaults)
	MemoryMB      int64 // Memory in MB (0 = use tier default)
	CPUMillicores int64 // CPU in millicores (0 = use tier default)
	DiskMB        int64 // Disk quota in MB (0 = use tier default)
}

// RecreateContainer recreates a container that was removed from Docker
// It preserves the user's volume data if it still exists
func (m *Manager) RecreateContainer(ctx context.Context, cfg RecreateContainerConfig) (*ContainerInfo, error) {
	// Parse image type
	var imageType, customImage string
	if strings.HasPrefix(cfg.Image, "custom:") {
		imageType = "custom"
		customImage = strings.TrimPrefix(cfg.Image, "custom:")
	} else {
		imageType = cfg.Image
	}

	// Remove old entry from manager's tracking if exists
	m.mu.Lock()
	if _, exists := m.containers[cfg.OldDockerID]; exists {
		delete(m.containers, cfg.OldDockerID)
		// Remove from user index
		if dockerIDs, ok := m.userIndex[cfg.UserID]; ok {
			newIDs := make([]string, 0, len(dockerIDs))
			for _, id := range dockerIDs {
				if id != cfg.OldDockerID {
					newIDs = append(newIDs, id)
				}
			}
			if len(newIDs) > 0 {
				m.userIndex[cfg.UserID] = newIDs
			} else {
				delete(m.userIndex, cfg.UserID)
			}
		}
	}
	m.mu.Unlock()

	// Build labels
	labels := map[string]string{
		"rexec.tier":    cfg.Tier,
		"rexec.user_id": cfg.UserID,
	}

	// Create new container using existing method
	containerCfg := ContainerConfig{
		UserID:        cfg.UserID,
		ContainerName: cfg.ContainerName,
		ImageType:     imageType,
		CustomImage:   customImage,
		Labels:        labels,
	}

	// Apply tier-based resource limits (CPULimit in millicores: 1000 = 1 CPU)
	// Use custom limits if provided, otherwise use tier defaults
	if cfg.MemoryMB > 0 && cfg.CPUMillicores > 0 {
		containerCfg.MemoryLimit = cfg.MemoryMB * 1024 * 1024
		containerCfg.CPULimit = cfg.CPUMillicores
		if cfg.DiskMB > 0 {
			containerCfg.DiskQuota = cfg.DiskMB * 1024 * 1024
		}
		log.Printf("[RecreateContainer] Using custom resource limits: memory=%dMB, cpu=%d millicores, disk=%dMB", cfg.MemoryMB, cfg.CPUMillicores, cfg.DiskMB)
	} else {
		switch cfg.Tier {
		case "pro":
			containerCfg.MemoryLimit = 2048 * 1024 * 1024 // 2GB
			containerCfg.CPULimit = 2000                  // 2 CPUs
			containerCfg.DiskQuota = 20 * 1024 * 1024 * 1024 // 20GB
		case "enterprise":
			containerCfg.MemoryLimit = 4096 * 1024 * 1024 // 4GB
			containerCfg.CPULimit = 4000                  // 4 CPUs
			containerCfg.DiskQuota = 50 * 1024 * 1024 * 1024 // 50GB
		default: // free/guest
			containerCfg.MemoryLimit = 512 * 1024 * 1024 // 512MB
			containerCfg.CPULimit = 500                  // 0.5 CPU
			containerCfg.DiskQuota = 5 * 1024 * 1024 * 1024 // 5GB
		}
	}

	return m.CreateContainer(ctx, containerCfg)
}

// ContainerResourceStats represents simplified container resource usage
type ContainerResourceStats struct {
	CPUPercent  float64 `json:"cpu_percent"`
	Memory      float64 `json:"memory"`       // in bytes
	MemoryLimit float64 `json:"memory_limit"` // in bytes
	DiskRead    float64 `json:"disk_read"`    // bytes read
	DiskWrite   float64 `json:"disk_write"`   // bytes written
	DiskLimit   float64 `json:"disk_limit"`   // bytes (from storage-opt size)
	NetRx       float64 `json:"net_rx"`       // bytes received
	NetTx       float64 `json:"net_tx"`       // bytes transmitted
}

// StreamContainerStats streams container stats to the provided channel
func (m *Manager) StreamContainerStats(ctx context.Context, containerID string, statsCh chan<- ContainerResourceStats) error {
	// Get container's configured memory limit first (Docker stats may return host memory)
	var configuredMemoryLimit int64
	var configuredDiskLimit int64
	inspectInfo, err := m.client.ContainerInspect(ctx, containerID)
	if err == nil && inspectInfo.HostConfig != nil {
		if inspectInfo.HostConfig.Memory > 0 {
			configuredMemoryLimit = inspectInfo.HostConfig.Memory
			log.Printf("[StreamContainerStats] Container %s has configured memory limit: %d bytes", containerID, configuredMemoryLimit)
		}
		
		// Get disk limit from StorageOpt
		if sizeStr, ok := inspectInfo.HostConfig.StorageOpt["size"]; ok {
			// Parse size like "2G", "500M", etc.
			configuredDiskLimit = parseSizeString(sizeStr)
			if configuredDiskLimit > 0 {
				log.Printf("[StreamContainerStats] Container %s has configured disk limit: %d bytes", containerID, configuredDiskLimit)
			}
		}
	}
	
	// Fallback: try to get memory/disk limits from container labels (stored during creation)
	if inspectInfo.Config != nil && inspectInfo.Config.Labels != nil {
		// Memory limit from label
		if configuredMemoryLimit == 0 {
			if memLimitStr, ok := inspectInfo.Config.Labels["rexec.memory_limit"]; ok {
				if memLimit, err := strconv.ParseInt(memLimitStr, 10, 64); err == nil && memLimit > 0 {
					configuredMemoryLimit = memLimit
					log.Printf("[StreamContainerStats] Container %s: using label memory limit: %d bytes", containerID, configuredMemoryLimit)
				}
			}
		}
		
		// Disk limit from label
		if configuredDiskLimit == 0 {
			if diskLimitStr, ok := inspectInfo.Config.Labels["rexec.disk_quota"]; ok {
				if diskLimit, err := strconv.ParseInt(diskLimitStr, 10, 64); err == nil && diskLimit > 0 {
					configuredDiskLimit = diskLimit
					log.Printf("[StreamContainerStats] Container %s: using label disk limit: %d bytes", containerID, configuredDiskLimit)
				}
			}
		}
		
		// If still not set, use tier-based fallback
		if configuredMemoryLimit == 0 {
			tier := inspectInfo.Config.Labels["rexec.tier"]
			switch tier {
			case "pro":
				configuredMemoryLimit = 2048 * 1024 * 1024 // 2GB
			case "enterprise":
				configuredMemoryLimit = 4096 * 1024 * 1024 // 4GB
			default: // free/guest
				configuredMemoryLimit = 512 * 1024 * 1024 // 512MB
			}
			log.Printf("[StreamContainerStats] Container %s: using tier-based memory limit (%s): %d bytes", containerID, tier, configuredMemoryLimit)
		}
	}

	stats, err := m.client.ContainerStats(ctx, containerID, true)
	if err != nil {
		return err
	}
	defer stats.Body.Close()

	decoder := json.NewDecoder(stats.Body)
	var previousCPU uint64
	var previousSystem uint64

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var v *container.StatsResponse
			if err := decoder.Decode(&v); err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}

			// Calculate CPU percent
			// Based on: https://github.com/docker/cli/blob/master/cli/command/container/stats_helpers.go
			var cpuPercent = 0.0
			previousCPU = v.PreCPUStats.CPUUsage.TotalUsage
			previousSystem = v.PreCPUStats.SystemUsage

			// If PreCPUStats is empty (first reading), use the values from the struct if available
			// Docker API sometimes returns 0 for PreCPUStats on the first read
			if previousCPU == 0 && previousSystem == 0 {
				// We can't calculate CPU usage without a delta, so skip this reading
				// or just send 0.
			} else {
				cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(previousCPU)
				systemDelta := float64(v.CPUStats.SystemUsage) - float64(previousSystem)

				if systemDelta > 0.0 && cpuDelta > 0.0 {
					cpuPercent = (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100.0
				}
			}

			// Calculate Memory usage
			// On cgroup v1, Cache is included in Usage, so we subtract it
			// On cgroup v2, it might be different, but for simplicity we use Usage - Stats["cache"]
			memUsage := float64(v.MemoryStats.Usage)
			if v.MemoryStats.Stats != nil {
				if cache, ok := v.MemoryStats.Stats["cache"]; ok {
					memUsage -= float64(cache)
				} else if inactiveFile, ok := v.MemoryStats.Stats["inactive_file"]; ok {
					// cgroup v2
					memUsage -= float64(inactiveFile)
				}
			}

			// Calculate Disk I/O
			var diskRead, diskWrite float64
			if v.BlkioStats.IoServiceBytesRecursive != nil {
				for _, entry := range v.BlkioStats.IoServiceBytesRecursive {
					if entry.Op == "Read" || entry.Op == "read" {
						diskRead += float64(entry.Value)
					} else if entry.Op == "Write" || entry.Op == "write" {
						diskWrite += float64(entry.Value)
					}
				}
			}

			// Calculate Network I/O
			var netRx, netTx float64
			for _, netStats := range v.Networks {
				netRx += float64(netStats.RxBytes)
				netTx += float64(netStats.TxBytes)
			}

			// Use configured memory limit if Docker stats returns host memory (common with gVisor)
			memLimit := float64(v.MemoryStats.Limit)
			if configuredMemoryLimit > 0 && (memLimit == 0 || memLimit > float64(configuredMemoryLimit)*2) {
				// Docker returned 0 or host memory, use our configured limit
				memLimit = float64(configuredMemoryLimit)
			} else if configuredMemoryLimit == 0 && memLimit > 2*1024*1024*1024 {
				// No configured limit found but Docker returned very high value (likely host memory)
				// Default to 512MB for safety
				memLimit = 512 * 1024 * 1024
			}

			statsCh <- ContainerResourceStats{
				CPUPercent:  cpuPercent,
				Memory:      memUsage,
				MemoryLimit: memLimit,
				DiskRead:    diskRead,
				DiskWrite:   diskWrite,
				DiskLimit:   float64(configuredDiskLimit),
				NetRx:       netRx,
				NetTx:       netTx,
			}
		}
	}
}

// parseSizeString parses Docker size strings like "2G", "500M", "1024K" to bytes
func parseSizeString(s string) int64 {
	s = strings.TrimSpace(strings.ToUpper(s))
	if len(s) == 0 {
		return 0
	}
	
	multiplier := int64(1)
	numStr := s
	
	// Check for suffix
	lastChar := s[len(s)-1]
	switch lastChar {
	case 'K':
		multiplier = 1024
		numStr = s[:len(s)-1]
	case 'M':
		multiplier = 1024 * 1024
		numStr = s[:len(s)-1]
	case 'G':
		multiplier = 1024 * 1024 * 1024
		numStr = s[:len(s)-1]
	case 'T':
		multiplier = 1024 * 1024 * 1024 * 1024
		numStr = s[:len(s)-1]
	}
	
	// Also handle "GB", "MB" etc.
	if len(numStr) > 0 && numStr[len(numStr)-1] == 'B' {
		numStr = numStr[:len(numStr)-1]
	}
	
	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}
	
	return int64(val * float64(multiplier))
}
