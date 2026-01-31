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
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const IsolatedNetworkName = "rexec-isolated"

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
		strings.Contains(errMsg, "failed to connect to remote") ||
		strings.Contains(errMsg, "failed to create docker client for remote host") ||
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
	"ubuntu":     "ubuntu:24.04", // LTS until 2029, latest security patches
	"ubuntu-24":  "ubuntu:24.04", // Noble Numbat LTS
	"ubuntu-22":  "ubuntu:22.04", // Jammy Jellyfish LTS
	"ubuntu-20":  "ubuntu:20.04", // Focal Fossa LTS (EOL Apr 2025, still supported)
	"debian":     "debian:12",    // Bookworm (current stable)
	"debian-12":  "debian:12",    // Bookworm
	"debian-11":  "debian:11",    // Bullseye (oldstable)
	"kali":       "kalilinux/kali-rolling:latest",
	"parrot":     "parrotsec/core:latest",
	"mint":       "linuxmintd/mint22-amd64:latest", // Linux Mint 22 (latest)
	"elementary": "elementary/docker:stable",
	"devuan":     "devuan/devuan:excalibur", // Devuan 5.0
	// Security & Penetration Testing
	"blackarch":       "blackarchlinux/blackarch:latest",
	"parrot-security": "parrotsec/security:latest",
	// Red Hat-based (Updated Dec 2025)
	"fedora":        "fedora:41",                     // Fedora 41 (latest stable)
	"fedora-40":     "fedora:40",                     // Previous stable
	"fedora-39":     "fedora:39",                     // Older stable
	"centos":        "quay.io/centos/centos:stream9", // CentOS Stream 9
	"centos-stream": "quay.io/centos/centos:stream9",
	"rocky":         "rockylinux:9",       // Rocky Linux 9 (latest)
	"rocky-8":       "rockylinux:8",       // Rocky Linux 8
	"alma":          "almalinux:9",        // AlmaLinux 9 (latest)
	"alma-8":        "almalinux:8",        // AlmaLinux 8
	"oracle":        "oraclelinux:9",      // Oracle Linux 9
	"rhel":          "redhat/ubi9:latest", // Red Hat UBI 9
	"openeuler":     "openeuler/openeuler:24.03-lts",
	// Arch-based
	"archlinux": "archlinux:latest", // Rolling release
	"manjaro":   "manjarolinux/base:latest",
	"artix":     "artixlinux/artixlinux:latest", // Arch without systemd
	// SUSE-based (Updated Dec 2025)
	"opensuse":   "opensuse/leap:15.6",         // openSUSE Leap 15.6
	"tumbleweed": "opensuse/tumbleweed:latest", // Rolling release
	"mageia":     "mageia:9",                   // Mandriva fork
	// Independent Distributions
	"gentoo":    "gentoo/stage3:latest",
	"void":      "voidlinux/voidlinux:latest",
	"nixos":     "nixos/nix:latest",
	"slackware": "aclemons/slackware:15.0",
	"crux":      "dopsi/crux:latest", // Alternative for missing official image
	"guix":      "gnu/guix:latest",
	// Minimal / Embedded (Updated Dec 2025)
	"alpine":      "alpine:3.21",  // Alpine 3.21 (latest stable)
	"alpine-3.20": "alpine:3.20",  // Previous stable
	"alpine-3.18": "alpine:3.18",  // Older stable
	"busybox":     "busybox:1.37", // Latest busybox
	"tinycore":    "tatsushid/tinycore:latest",
	"openwrt":     "openwrt/rootfs:latest",
	// Container / Cloud Optimized
	"rancheros": "alpine:3.21", // RancherOS discontinued, using Alpine as lightweight alternative
	// Cloud Provider Specific (Updated Dec 2025)
	"amazonlinux":  "amazonlinux:2023", // Amazon Linux 2023 latest
	"amazonlinux2": "amazonlinux:2",    // Amazon Linux 2 (EOL 2025)
	"oracle-slim":  "oraclelinux:9-slim",
	"azurelinux":   "mcr.microsoft.com/azurelinux/base/core:3.0",
	// Scientific
	"scientific":  "scientificlinux/sl:latest",
	"neurodebian": "neurodebian:bookworm",
	// Specialized
	"clearlinux": "clearlinux:latest",
	"photon":     "photon:5.0", // VMware Photon OS 5.0
	// Raspberry Pi / ARM
	"raspberrypi": "balenalib/raspberry-pi-debian:bookworm",
	// macOS (VM-based) - CUA Lumier image (https://cua.ai/docs/lume/guide/advanced/lumier/docker)
	"macos":        "ghcr.io/trycua/macos-sequoia-cua:latest", // macOS Sequoia (CUA)
	"macos-legacy": "sickcodes/docker-osx:big-sur",            // Big Sur (legacy)
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
		{Name: "debian-11", DisplayName: "Debian 11 (Bullseye)", Description: "Previous stable Debian release", Category: "debian", Tags: []string{"oldstable"}, Popular: false},
		{Name: "debian-12", DisplayName: "Debian 12 (Bookworm)", Description: "Current stable Debian release", Category: "debian", Tags: []string{"stable"}, Popular: false},
		{Name: "debian", DisplayName: "Debian 12 (Bookworm)", Description: "Rock-solid stability with extensive packages", Category: "debian", Tags: []string{"stable", "server"}, Popular: true},
		{Name: "devuan", DisplayName: "Devuan 5 (Excalibur)", Description: "Debian without systemd - init freedom", Category: "debian", Tags: []string{"init-freedom", "advanced"}, Popular: false},
		{Name: "elementary", DisplayName: "elementary OS", Description: "Beautiful and privacy-focused desktop OS", Category: "debian", Tags: []string{"desktop", "beautiful"}, Popular: false},
		{Name: "mint", DisplayName: "Linux Mint 22", Description: "User-friendly Ubuntu derivative with Cinnamon", Category: "debian", Tags: []string{"desktop", "beginner-friendly"}, Popular: true},
		{Name: "ubuntu-20", DisplayName: "Ubuntu 20.04 LTS", Description: "Focal Fossa - Long-term support until 2025", Category: "debian", Tags: []string{"lts", "legacy"}, Popular: false},
		{Name: "ubuntu-22", DisplayName: "Ubuntu 22.04 LTS", Description: "Jammy Jellyfish - Stable and well-tested", Category: "debian", Tags: []string{"lts", "stable"}, Popular: false},
		{Name: "ubuntu", DisplayName: "Ubuntu 24.04 LTS", Description: "Latest Ubuntu LTS with best-in-class security", Category: "debian", Tags: []string{"lts", "popular", "beginner-friendly"}, Popular: true},
		{Name: "ubuntu-24", DisplayName: "Ubuntu 24.04 LTS", Description: "Noble Numbat - Latest Ubuntu LTS release", Category: "debian", Tags: []string{"lts", "latest"}, Popular: true},

		// Security & Penetration Testing
		{Name: "blackarch", DisplayName: "BlackArch Linux", Description: "Arch-based with 2900+ security tools", Category: "security", Tags: []string{"security", "pentest", "arch"}, Popular: false},
		{Name: "kali", DisplayName: "Kali Linux", Description: "Industry-standard penetration testing distro", Category: "security", Tags: []string{"security", "pentest", "hacking"}, Popular: true},
		{Name: "parrot", DisplayName: "Parrot OS", Description: "Security-focused with privacy tools", Category: "security", Tags: []string{"security", "privacy", "pentest"}, Popular: true},
		{Name: "parrot-security", DisplayName: "Parrot Security", Description: "Full Parrot security edition with all tools", Category: "security", Tags: []string{"security", "pentest", "full"}, Popular: false},

		// Red Hat-based - Updated Dec 2025
		{Name: "alma-8", DisplayName: "AlmaLinux 8.10", Description: "AlmaLinux 8 branch - stable enterprise", Category: "redhat", Tags: []string{"enterprise", "stable"}, Popular: false},
		{Name: "alma", DisplayName: "AlmaLinux 9.5", Description: "Community-driven RHEL fork with long support", Category: "redhat", Tags: []string{"enterprise", "rhel-compatible"}, Popular: true},
		{Name: "centos-stream", DisplayName: "CentOS Stream 9", Description: "Rolling preview of future RHEL", Category: "redhat", Tags: []string{"enterprise", "rolling"}, Popular: false},
		{Name: "centos", DisplayName: "CentOS Stream 9", Description: "Upstream for RHEL, community-driven", Category: "redhat", Tags: []string{"enterprise", "rhel-upstream"}, Popular: true},
		{Name: "fedora-39", DisplayName: "Fedora 39", Description: "Older Fedora release", Category: "redhat", Tags: []string{"stable"}, Popular: false},
		{Name: "fedora-40", DisplayName: "Fedora 40", Description: "Previous stable Fedora release", Category: "redhat", Tags: []string{"stable"}, Popular: false},
		{Name: "fedora", DisplayName: "Fedora 41", Description: "Latest Fedora with cutting-edge features", Category: "redhat", Tags: []string{"modern", "rhel-upstream", "latest"}, Popular: true},
		{Name: "openeuler", DisplayName: "openEuler 24.03 LTS", Description: "Enterprise Linux from Huawei", Category: "redhat", Tags: []string{"enterprise", "lts"}, Popular: false},
		{Name: "oracle", DisplayName: "Oracle Linux 9", Description: "Oracle's enterprise Linux with Ksplice", Category: "redhat", Tags: []string{"enterprise", "oracle"}, Popular: false},
		{Name: "rhel", DisplayName: "Red Hat UBI 9", Description: "Official Red Hat Universal Base Image", Category: "redhat", Tags: []string{"enterprise", "commercial"}, Popular: true},
		{Name: "rocky-8", DisplayName: "Rocky Linux 8.10", Description: "Rocky Linux 8 branch - stable and tested", Category: "redhat", Tags: []string{"enterprise", "stable"}, Popular: false},
		{Name: "rocky", DisplayName: "Rocky Linux 9.5", Description: "Enterprise-grade 1:1 RHEL-compatible", Category: "redhat", Tags: []string{"enterprise", "rhel-compatible"}, Popular: true},

		// Arch-based
		{Name: "archlinux", DisplayName: "Arch Linux", Description: "Rolling release with latest packages and AUR", Category: "arch", Tags: []string{"rolling", "bleeding-edge", "aur"}, Popular: true},
		{Name: "artix", DisplayName: "Artix Linux", Description: "Arch without systemd (OpenRC)", Category: "arch", Tags: []string{"rolling", "init-freedom"}, Popular: false},
		{Name: "manjaro", DisplayName: "Manjaro", Description: "User-friendly Arch with curated updates", Category: "arch", Tags: []string{"rolling", "beginner-friendly"}, Popular: true},

		// SUSE-based - Updated Dec 2025
		{Name: "mageia", DisplayName: "Mageia 9", Description: "Community-driven Mandriva fork", Category: "suse", Tags: []string{"rpm", "desktop", "stable"}, Popular: false},
		{Name: "opensuse", DisplayName: "openSUSE Leap 15.6", Description: "Stable enterprise-grade openSUSE", Category: "suse", Tags: []string{"enterprise", "stable", "zypper"}, Popular: true},
		{Name: "tumbleweed", DisplayName: "openSUSE Tumbleweed", Description: "Rolling release with tested updates", Category: "suse", Tags: []string{"rolling", "tested"}, Popular: false},

		// Independent Distributions
		{Name: "crux", DisplayName: "CRUX 3.7", Description: "Lightweight, BSD-style init scripts", Category: "independent", Tags: []string{"lightweight", "bsd-style", "simple"}, Popular: false},
		{Name: "gentoo", DisplayName: "Gentoo Linux", Description: "Source-based with extreme customization", Category: "independent", Tags: []string{"source-based", "advanced", "performance"}, Popular: false},
		{Name: "guix", DisplayName: "Guix System", Description: "Advanced transactional package manager", Category: "independent", Tags: []string{"functional", "gnu", "scheme"}, Popular: false},
		{Name: "nixos", DisplayName: "NixOS", Description: "Declarative configuration and reproducible builds", Category: "independent", Tags: []string{"declarative", "nix", "reproducible"}, Popular: false},
		{Name: "slackware", DisplayName: "Slackware 15.0", Description: "Oldest maintained Linux distro, Unix-like", Category: "independent", Tags: []string{"classic", "stable", "unix-like"}, Popular: false},
		{Name: "void", DisplayName: "Void Linux", Description: "Independent distro with runit init system", Category: "independent", Tags: []string{"independent", "runit", "rolling"}, Popular: false},

		// Minimal / Embedded - Updated Dec 2025
		{Name: "alpine-3.18", DisplayName: "Alpine 3.18", Description: "Older stable Alpine release", Category: "minimal", Tags: []string{"minimal", "legacy"}, Popular: false},
		{Name: "alpine-3.20", DisplayName: "Alpine 3.20", Description: "Previous stable Alpine release", Category: "minimal", Tags: []string{"minimal", "stable"}, Popular: false},
		{Name: "alpine", DisplayName: "Alpine 3.21", Description: "Lightweight and security-oriented (6MB)", Category: "minimal", Tags: []string{"minimal", "docker", "security"}, Popular: true},
		{Name: "busybox", DisplayName: "BusyBox 1.37", Description: "Ultra-minimal Unix utilities (~2MB)", Category: "minimal", Tags: []string{"minimal", "embedded"}, Popular: false},
		{Name: "openwrt", DisplayName: "OpenWrt 23.05", Description: "Embedded operating system for routers", Category: "minimal", Tags: []string{"network", "embedded", "router"}, Popular: false},
		{Name: "tinycore", DisplayName: "TinyCore Linux", Description: "The smallest subset of Linux (~16MB)", Category: "minimal", Tags: []string{"micro", "fast", "ram-only"}, Popular: false},

		// Container / Cloud Optimized
		{Name: "rancheros", DisplayName: "RancherOS (Alpine)", Description: "Lightweight container-optimized OS (Alpine-based)", Category: "container", Tags: []string{"containers", "docker", "minimal"}, Popular: false},

		// Cloud Provider Specific - Updated Dec 2025
		{Name: "amazonlinux", DisplayName: "Amazon Linux 2023", Description: "Latest Amazon Linux optimized for AWS", Category: "cloud", Tags: []string{"aws", "cloud", "enterprise"}, Popular: true},
		{Name: "amazonlinux2", DisplayName: "Amazon Linux 2", Description: "LTS Amazon Linux (EOL Jun 2025)", Category: "cloud", Tags: []string{"aws", "cloud", "legacy"}, Popular: false},
		{Name: "azurelinux", DisplayName: "Azure Linux 3.0", Description: "Microsoft's OS for Azure Kubernetes Service", Category: "cloud", Tags: []string{"azure", "microsoft", "cloud"}, Popular: false},
		{Name: "oracle-slim", DisplayName: "Oracle Linux 9 Slim", Description: "Lightweight Oracle Linux", Category: "cloud", Tags: []string{"oracle", "cloud", "minimal"}, Popular: false},

		// Scientific
		{Name: "neurodebian", DisplayName: "NeuroDebian", Description: "Neuroscience-oriented Debian", Category: "developer", Tags: []string{"science", "brain", "research"}, Popular: false},
		{Name: "scientific", DisplayName: "Scientific Linux", Description: "For scientific computing and research", Category: "developer", Tags: []string{"scientific", "research", "rhel"}, Popular: false},

		// Specialized
		{Name: "clearlinux", DisplayName: "Clear Linux", Description: "Intel-optimized for maximum performance", Category: "specialized", Tags: []string{"performance", "intel", "cloud"}, Popular: false},
		{Name: "photon", DisplayName: "VMware Photon OS 5.0", Description: "Optimized for VMware and containers", Category: "specialized", Tags: []string{"vmware", "container", "minimal"}, Popular: false},

		// Raspberry Pi / ARM
		{Name: "raspberrypi", DisplayName: "Raspberry Pi OS", Description: "Debian-based OS for Raspberry Pi/ARM", Category: "embedded", Tags: []string{"raspberry-pi", "arm", "iot"}, Popular: false},

		// macOS - CUA Lumier image
		{Name: "macos", DisplayName: "macOS (Sequoia)", Description: "Apple macOS Sequoia (CUA Lumier, VM-based)", Category: "macos", Tags: []string{"macos", "apple", "vm", "sequoia"}, Popular: true},
		{Name: "macos-legacy", DisplayName: "macOS (Big Sur)", Description: "Apple macOS Big Sur (VM-based)", Category: "macos", Tags: []string{"macos", "apple", "vm", "legacy"}, Popular: false},
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
	"macos":        "/bin/bash",
	"macos-legacy": "/bin/bash",
}

// ImageFallbackShells provides fallback shells to try if the primary fails
var ImageFallbackShells = []string{"/bin/sh", "/bin/bash", "/bin/ash"}

// ContainerConfig holds configuration for creating a new container
type ContainerConfig struct {
	UserID        string
	ContainerName string            // User-provided name for the container
	ImageType     string            // ubuntu, debian, arch, alpine, or "custom:imagename"
	CustomImage   string            // Custom Docker image name (if ImageType is "custom")
	Role          string            // Role ID (e.g. "standard", "node", "python")
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
	client           client.CommonAPIClient
	containers       map[string]*ContainerInfo // dockerID -> container info
	userIndex        map[string][]string       // userID -> list of dockerIDs
	mu               sync.RWMutex
	volumePath       string // base path for user volumes
	diskQuotaEnabled bool   // whether disk quota is available
	diskQuotaChecked bool   // whether we've checked for disk quota support
	diskQuotaCheckMu sync.Once

	// Stats broadcasting
	activeStatsStreams map[string]*StatsBroadcaster
	statsMu            sync.Mutex
}

// StatsBroadcaster manages a single stats stream from Docker and broadcasts to multiple subscribers
type StatsBroadcaster struct {
	containerID string
	manager     *Manager
	subscribers map[chan ContainerResourceStats]struct{}
	mu          sync.Mutex
	done        chan struct{}
	cancel      context.CancelFunc // To stop the Docker stats stream
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
		client:             cli,
		containers:         make(map[string]*ContainerInfo),
		userIndex:          make(map[string][]string),
		volumePath:         volumePath,
		diskQuotaEnabled:   false,
		diskQuotaChecked:   false,
		activeStatsStreams: make(map[string]*StatsBroadcaster),
	}

	// Check disk quota availability asynchronously
	go mgr.checkDiskQuotaSupport()

	// Ensure isolated network exists
	if err := mgr.ensureIsolatedNetwork(); err != nil {
		log.Printf("[Container] WARNING: Failed to create isolated network: %v", err)
	}

	return mgr, nil
}

// ensureIsolatedNetwork ensures the isolated network exists with ICC disabled
func (m *Manager) ensureIsolatedNetwork() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := m.client.NetworkInspect(ctx, IsolatedNetworkName, network.InspectOptions{})
	if err == nil {
		return nil // Network exists
	}

	if !client.IsErrNotFound(err) {
		return fmt.Errorf("failed to inspect network: %w", err)
	}

	// Create network with Inter-Container Communication (ICC) disabled
	_, err = m.client.NetworkCreate(ctx, IsolatedNetworkName, network.CreateOptions{
		Driver: "bridge",
		Options: map[string]string{
			"com.docker.network.bridge.enable_icc": "false",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create isolated network: %w", err)
	}

	log.Printf("[Container] Created isolated network: %s", IsolatedNetworkName)
	return nil
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

// CheckImageExists checks if an image exists in Docker
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
	// Consistent volume name for data persistence
	volumeName := containerName

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
			"rexec.role":           cfg.Role,
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
		minMemory := int64(4096 * 1024 * 1024)    // 4GB
		minCPU := int64(2000)                     // 2 vCPU
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
	cpuPeriod := int64(100000)                    // 100ms in microseconds
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
			PidsLimit:  &[]int64{512}[0], // Limit number of processes (512 allows AI tools like opencode)
		},
		// Storage options for disk quota (requires overlay2 on XFS with pquota mount option)
		StorageOpt: storageOpts,
		// Security options - prevent privilege escalation
		SecurityOpt: []string{
			"no-new-privileges:true",
			// Use default Docker seccomp profile (more secure than unconfined)
			// This blocks dangerous syscalls like reboot, mount, ptrace, etc.
		},
		// Prevent container from gaining new privileges
		Privileged: false,
		// Note: ReadonlyRootfs is NOT used to allow package installation during role setup
		// Security is maintained via other measures (seccomp, capabilities, etc.)
		ReadonlyRootfs: false,
		// Tmpfs mounts for writable directories
		// These directories MUST be writable for package management and tools
		Tmpfs: map[string]string{
			"/tmp":                    "rw,exec,nosuid,size=256m", // exec required for opencode TUI library
			"/var/tmp":                "rw,noexec,nosuid,size=50m",
			"/run":                    "rw,noexec,nosuid,size=50m",
			"/var/run":                "rw,noexec,nosuid,size=50m",
			"/var/log":                "rw,noexec,nosuid,size=50m",
			"/var/lib/apt/lists":      "rw,noexec,nosuid,size=100m", // apt package lists (always writable)
			"/var/cache/apt/archives": "rw,noexec,nosuid,size=500m", // apt package cache
			// NOTE: /root and /home/user are NOT tmpfs - they use the overlay filesystem
			// This ensures installed tools (opencode, tgpt, etc.) persist across container restarts
		},
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
			"/sys/kernel",
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
			"CHOWN",            // Change file ownership
			"DAC_OVERRIDE",     // Bypass file permission checks (needed for sudo)
			"FOWNER",           // Bypass permission checks on file owner
			"SETGID",           // Set group ID
			"SETUID",           // Set user ID (needed for su/sudo)
			"KILL",             // Send signals
			"NET_BIND_SERVICE", // Bind to ports < 1024 (useful for web dev)
			"SYS_PTRACE",       // Needed for TUI apps like opencode, debugging, strace
		},
		// Sysctls - restrict kernel parameters the container can modify
		Sysctls: map[string]string{
			"net.ipv4.ip_unprivileged_port_start": "0", // Allow binding to low ports
		},
		// Restart policy
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		// Port bindings (optional, for future use)
		PortBindings: nat.PortMap{},
	}

	// Special handling for macOS (CUA Lumier / docker-osx style VM images)
	// Check for "macos" or "osx" in image type (case-insensitive)
	isMacOS := strings.Contains(strings.ToLower(cfg.ImageType), "macos") || strings.Contains(strings.ToLower(cfg.ImageType), "osx")

	if isMacOS {
		// Do NOT override Entrypoint - let the VM boot script run
		// Do NOT use /home/user working dir - use default
		log.Printf("[Container] Configuring macOS container (privileged, kvm, headless)")

		// VM environment variables for headless VNC mode (CUA Lumier / docker-osx compatible)
		// GENERATE_UNIQUE=true generates a unique serial/MLB for each container
		// DEVICE_MODEL and SERIAL are optional customizations
		// CPU throttling options to reduce idle CPU usage and prevent disconnections:
		// - cpu-pm=on: Enable CPU power management (sleep states)
		// - hv-time: Hyper-V timer for better guest idle
		// - +invtsc: Invariant TSC for stable timing
		containerConfig.Env = append(containerConfig.Env,
			"GENERATE_UNIQUE=true",
			"DISPLAY=:99",             // Use virtual display (Xvfb)
			"LIBGL_ALWAYS_SOFTWARE=1", // Software rendering
			"NOGRAPHIC=true",          // Disable SDL/GTK display
			"CPU_STRING=host,+invtsc,vmware-cpuid-freq=on,cpu-pm=on",                                                      // CPU with power management
			"EXTRA=-display none -vnc 0.0.0.0:0,websocket=on -global ICH9-LPC.disable_s3=1 -global ICH9-LPC.disable_s4=1", // VNC + disable deep sleep states that cause issues
		)

		hostConfig.Privileged = true
		hostConfig.SecurityOpt = nil      // Clear security opts to allow KVM
		hostConfig.CapDrop = nil          // Don't drop caps
		hostConfig.CapAdd = nil           // Allow all caps
		hostConfig.ReadonlyRootfs = false // macOS needs writable root
		hostConfig.Tmpfs = nil            // Clear tmpfs for macOS

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
		// Mount persistent volume for user data
		hostConfig.Mounts = []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/home/user",
			},
		}

		// Remove /home/user from tmpfs since we use a volume mount
		delete(hostConfig.Tmpfs, "/home/user")

		// Ensure permissions on volume mount and configure tmux for session persistence
		// The tmux session is created by the terminal handler on first connect
		// This just sets up the config and keeps the container alive
		tmuxSetup := fmt.Sprintf(`mkdir -p /home/user && (chmod 777 /home/user || true) &&
mkdir -p /home/user/.tmux &&
cat > /home/user/.tmux.conf << 'TMUXCONF'
# Rexec tmux config for session persistence
set -g default-terminal "xterm-256color"
set -ga terminal-overrides ",xterm-256color:Tc"
set -g history-limit 50000
# Mouse mode: enable for scrolling but configure for better copy behavior
# Users can hold Shift to bypass tmux and use browser selection
set -g mouse on
# When selecting with mouse, automatically copy to tmux buffer
# This works with tmux's internal clipboard
bind-key -T copy-mode-vi MouseDragEnd1Pane send-keys -X copy-selection-no-clear
bind-key -T copy-mode MouseDragEnd1Pane send-keys -X copy-selection-no-clear
set -g status off
set -g set-titles on
set -g set-titles-string "#{pane_title}"
# Zero escape time for responsive Ctrl+C and other key combos
set -g escape-time 0
set -sg escape-time 0
set -g focus-events on
# Proper terminal size handling for TUI apps
set -g aggressive-resize on
setw -g aggressive-resize on
# Keep sessions alive when client detaches
set -g destroy-unattached off
set -g detach-on-destroy off
# Recreate session if it dies
set -g remain-on-exit off
# Default shell
set -g default-shell %s
set -g default-command %s
# Pass through Ctrl+C and other control keys without interception
set -g xterm-keys on
# Ensure UTF-8 is handled properly
setw -g mode-keys vi
# Fix for terminal reset on reconnect - clear alternate screen issues
set -ga terminal-overrides ',*:Ss=\E[%%p1%%d q:Se=\E[2 q'
# Clipboard passthrough for OSC 52 (requires tmux 3.3+, silently ignored on older)
set -ga terminal-overrides ',xterm*:Ms=\E]52;c;%%p2%%s\007'
TMUXCONF

# Apply allow-passthrough only if tmux supports it (3.3+)
if tmux -V 2>/dev/null | grep -qE 'tmux ([3-9]\.[3-9]|[4-9]\.)'; then
    echo 'set -g allow-passthrough on' >> /home/user/.tmux.conf
fi
export HOME=/home/user &&
cd /home/user &&
# Keep container running indefinitely
# Terminal sessions are managed via docker exec + tmux
exec tail -f /dev/null`, shell, shell)
		containerConfig.Entrypoint = []string{"/bin/sh", "-c", tmuxSetup}
		containerConfig.WorkingDir = "/home/user"
	}

	// Network configuration
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			IsolatedNetworkName: {},
		},
	}

	// Clean up any existing container with the same name (from failed previous attempts)
	// This prevents "container name already in use" errors
	// IMPORTANT: Only remove if it belongs to the same user to prevent accidental deletion
	existingContainers, err := m.client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", "^/"+containerName+"$")),
	})
	if err == nil && len(existingContainers) > 0 {
		for _, existing := range existingContainers {
			// Check if this container belongs to the same user
			ownerUserID := existing.Labels["rexec.user_id"]
			if ownerUserID == "" || ownerUserID == cfg.UserID {
				log.Printf("[Container] Removing stale container with same name: %s (%s) owned by user %s", containerName, existing.ID[:12], ownerUserID)
				_ = m.client.ContainerStop(ctx, existing.ID, container.StopOptions{})
				_ = m.client.ContainerRemove(ctx, existing.ID, container.RemoveOptions{Force: true})
			} else {
				// Container belongs to a different user - this should never happen but log it
				log.Printf("[Container] WARNING: Container name conflict! %s (%s) belongs to user %s, not %s", containerName, existing.ID[:12], ownerUserID, cfg.UserID)
				return nil, fmt.Errorf("container name conflict: name already in use by another user")
			}
		}
	}

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
		"rexec.role":           cfg.Role,
		"rexec.managed":        "true",
	}, cfg.Labels)

	// Get IP address (handle custom network)
	ipAddress := inspect.NetworkSettings.IPAddress
	if ipAddress == "" && inspect.NetworkSettings.Networks != nil {
		if net, ok := inspect.NetworkSettings.Networks[IsolatedNetworkName]; ok {
			ipAddress = net.IPAddress
		} else {
			// Fallback to any network
			for _, net := range inspect.NetworkSettings.Networks {
				ipAddress = net.IPAddress
				break
			}
		}
	}

	info := &ContainerInfo{
		ID:            resp.ID,
		UserID:        cfg.UserID,
		ContainerName: cfg.ContainerName,
		ImageType:     imageType,
		Status:        "configuring", // Set to configuring initially so UI waits
		CreatedAt:     now,
		LastUsedAt:    now,
		IPAddress:     ipAddress,
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

	// Fallback: search by ID prefix (for short IDs like first 12 chars)
	if len(idOrName) >= 12 {
		for dockerID, info := range m.containers {
			if strings.HasPrefix(dockerID, idOrName) {
				return info, true
			}
			if len(dockerID) >= 12 && strings.HasPrefix(idOrName, dockerID[:12]) {
				return info, true
			}
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
func (m *Manager) GetClient() client.CommonAPIClient {
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
	cpuPeriod := int64(100000)                     // 100ms in microseconds
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
// Only returns guest containers - authenticated users don't have idle timeout
func (m *Manager) GetIdleContainers(threshold time.Duration) []*ContainerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	result := make([]*ContainerInfo, 0)

	for _, info := range m.containers {
		if info.Status != "running" {
			continue
		}
		if now.Sub(info.LastUsedAt) <= threshold {
			continue
		}

		// Only apply idle timeout to guest containers
		// Authenticated users don't have idle timeout - their containers run until they stop them
		isGuest := false
		if info.Labels != nil {
			if tier, ok := info.Labels["rexec.tier"]; ok && tier == "guest" {
				isGuest = true
			}
			if _, ok := info.Labels["rexec.guest"]; ok {
				isGuest = true
			}
		}

		if isGuest {
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

	// Rebuild indexes to avoid duplicate entries on repeated loads.
	m.containers = make(map[string]*ContainerInfo)
	m.userIndex = make(map[string][]string)

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
			Labels:        c.Labels,
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
	case "guest":
		return 1
	case "trial", "free": // Unified 60-day trial experience
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
	Role          string // Role to restore
	OldDockerID   string
	Tier          string
	// Optional custom resource limits (if set, these override tier defaults)
	MemoryMB      int64 // Memory in MB (0 = use tier default)
	CPUMillicores int64 // CPU in millicores (0 = use tier default)
	DiskMB        int64 // Disk quota in MB (0 = use tier default)
	// Shell options
	UseTmux *bool // Whether to use tmux for session persistence (nil = inherit from old container)
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
	if cfg.Role != "" {
		labels["rexec.role"] = cfg.Role
	}
	// Preserve tmux preference
	if cfg.UseTmux != nil {
		if *cfg.UseTmux {
			labels["rexec.use_tmux"] = "true"
		} else {
			labels["rexec.use_tmux"] = "false"
		}
	}

	// Create new container using existing method
	containerCfg := ContainerConfig{
		UserID:        cfg.UserID,
		ContainerName: cfg.ContainerName,
		ImageType:     imageType,
		CustomImage:   customImage,
		Role:          cfg.Role,
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
			containerCfg.MemoryLimit = 2048 * 1024 * 1024    // 2GB
			containerCfg.CPULimit = 2000                     // 2 CPUs
			containerCfg.DiskQuota = 20 * 1024 * 1024 * 1024 // 20GB
		case "enterprise":
			containerCfg.MemoryLimit = 4096 * 1024 * 1024    // 4GB
			containerCfg.CPULimit = 4000                     // 4 CPUs
			containerCfg.DiskQuota = 50 * 1024 * 1024 * 1024 // 50GB
		default: // free/guest
			containerCfg.MemoryLimit = 512 * 1024 * 1024    // 512MB
			containerCfg.CPULimit = 500                     // 0.5 CPU
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
	DiskUsage   float64 `json:"disk_usage"`   // bytes stored (rw layer)
	DiskLimit   float64 `json:"disk_limit"`   // bytes (from storage-opt size)
	NetRx       float64 `json:"net_rx"`       // bytes received
	NetTx       float64 `json:"net_tx"`       // bytes transmitted
}

// StreamContainerStats streams container stats to the provided channel
func (m *Manager) StreamContainerStats(ctx context.Context, containerID string, statsCh chan<- ContainerResourceStats) error {
	// Get or create a broadcaster for this container
	broadcaster := m.getOrCreateStatsBroadcaster(containerID)

	// Subscribe to the broadcaster
	subCh, unsubscribe := broadcaster.Subscribe()
	defer unsubscribe()

	// Forward stats from subscription to the caller
	for {
		select {
		case <-ctx.Done():
			return nil
		case stats, ok := <-subCh:
			if !ok {
				return nil // Broadcaster stopped
			}
			select {
			case statsCh <- stats:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

// getOrCreateStatsBroadcaster gets an existing broadcaster or creates a new one
func (m *Manager) getOrCreateStatsBroadcaster(containerID string) *StatsBroadcaster {
	m.statsMu.Lock()
	defer m.statsMu.Unlock()

	if sb, exists := m.activeStatsStreams[containerID]; exists {
		return sb
	}

	sb := &StatsBroadcaster{
		containerID: containerID,
		manager:     m,
		subscribers: make(map[chan ContainerResourceStats]struct{}),
		done:        make(chan struct{}),
	}
	m.activeStatsStreams[containerID] = sb

	// Start the broadcast loop in background
	go sb.start()

	return sb
}

// Subscribe adds a subscriber to the broadcaster
func (sb *StatsBroadcaster) Subscribe() (chan ContainerResourceStats, func()) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	ch := make(chan ContainerResourceStats, 10) // Buffered channel to prevent blocking
	sb.subscribers[ch] = struct{}{}

	return ch, func() {
		sb.mu.Lock()
		defer sb.mu.Unlock()

		delete(sb.subscribers, ch)
		close(ch)

		// If no subscribers left, stop the broadcaster
		if len(sb.subscribers) == 0 {
			sb.manager.statsMu.Lock()
			// Check if we are still the active broadcaster (race protection)
			if sb.manager.activeStatsStreams[sb.containerID] == sb {
				delete(sb.manager.activeStatsStreams, sb.containerID)
				if sb.cancel != nil {
					sb.cancel()
				}
			}
			sb.manager.statsMu.Unlock()
		}
	}
}

// start begins the Docker stats stream and broadcasts updates
func (sb *StatsBroadcaster) start() {
	ctx, cancel := context.WithCancel(context.Background())
	sb.cancel = cancel

	// Helper to broadcast stats
	broadcast := func(stats ContainerResourceStats) {
		sb.mu.Lock()
		defer sb.mu.Unlock()
		for ch := range sb.subscribers {
			select {
			case ch <- stats:
			default:
				// Skip if channel is full (slow consumer)
			}
		}
	}

	// 1. Get configured limits (same logic as before)
	var configuredMemoryLimit int64
	var configuredDiskLimit int64

	inspectInfo, err := sb.manager.client.ContainerInspect(ctx, sb.containerID)
	if err != nil {
		// Log error and remove self
		sb.manager.statsMu.Lock()
		delete(sb.manager.activeStatsStreams, sb.containerID)
		sb.manager.statsMu.Unlock()
		return
	}

	if inspectInfo.HostConfig != nil {
		if inspectInfo.HostConfig.Memory > 0 {
			configuredMemoryLimit = inspectInfo.HostConfig.Memory
		}
		if sizeStr, ok := inspectInfo.HostConfig.StorageOpt["size"]; ok {
			configuredDiskLimit = parseSizeString(sizeStr)
		}
	}

	if inspectInfo.Config != nil && inspectInfo.Config.Labels != nil {
		if configuredMemoryLimit == 0 {
			if memLimitStr, ok := inspectInfo.Config.Labels["rexec.memory_limit"]; ok {
				if memLimit, err := strconv.ParseInt(memLimitStr, 10, 64); err == nil && memLimit > 0 {
					configuredMemoryLimit = memLimit
				}
			}
		}
		if configuredDiskLimit == 0 {
			if diskLimitStr, ok := inspectInfo.Config.Labels["rexec.disk_quota"]; ok {
				if diskLimit, err := strconv.ParseInt(diskLimitStr, 10, 64); err == nil && diskLimit > 0 {
					configuredDiskLimit = diskLimit
				}
			}
		}
		if configuredMemoryLimit == 0 {
			tier := inspectInfo.Config.Labels["rexec.tier"]
			switch tier {
			case "pro":
				configuredMemoryLimit = 2048 * 1024 * 1024
			case "enterprise":
				configuredMemoryLimit = 4096 * 1024 * 1024
			default:
				configuredMemoryLimit = 512 * 1024 * 1024
			}
		}
	}

	// 2. Start Docker stats stream
	stats, err := sb.manager.client.ContainerStats(ctx, sb.containerID, true)
	if err != nil {
		// Log error and remove self
		sb.manager.statsMu.Lock()
		delete(sb.manager.activeStatsStreams, sb.containerID)
		sb.manager.statsMu.Unlock()
		return
	}
	defer stats.Body.Close()

	decoder := json.NewDecoder(stats.Body)
	var previousCPU uint64
	var previousSystem uint64
	var diskUsage float64
	var ticks int

	for {
		select {
		case <-ctx.Done():
			return
		default:
			var v *container.StatsResponse
			if err := decoder.Decode(&v); err != nil {
				if err != io.EOF && !strings.Contains(err.Error(), "closed") {
					// Log error
				}
				return
			}

			ticks++
			// Update disk usage every 10 seconds (approx)
			if ticks%10 == 0 {
				// Use a short timeout context for disk check
				diskCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				usage := sb.manager.getContainerDiskUsage(diskCtx, sb.containerID)
				cancel()
				if usage > 0 {
					diskUsage = usage
				}
			}

			// Calculate CPU percent
			var cpuPercent = 0.0
			prevCPUVal := previousCPU
			prevSysVal := previousSystem

			if prevCPUVal == 0 {
				prevCPUVal = v.PreCPUStats.CPUUsage.TotalUsage
				prevSysVal = v.PreCPUStats.SystemUsage
			}

			if prevCPUVal > 0 && prevSysVal > 0 {
				cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(prevCPUVal)
				systemDelta := float64(v.CPUStats.SystemUsage) - float64(prevSysVal)

				if systemDelta > 0.0 && cpuDelta > 0.0 {
					numCPUs := float64(len(v.CPUStats.CPUUsage.PercpuUsage))
					if numCPUs == 0 && v.CPUStats.OnlineCPUs > 0 {
						numCPUs = float64(v.CPUStats.OnlineCPUs)
					}
					if numCPUs == 0 {
						numCPUs = float64(runtime.NumCPU())
					}
					cpuPercent = (cpuDelta / systemDelta) * numCPUs * 100.0
				}
			}

			previousCPU = v.CPUStats.CPUUsage.TotalUsage
			previousSystem = v.CPUStats.SystemUsage

			// Calculate Memory usage
			memUsage := float64(v.MemoryStats.Usage)
			if v.MemoryStats.Stats != nil {
				if cache, ok := v.MemoryStats.Stats["cache"]; ok {
					memUsage -= float64(cache)
				} else if inactiveFile, ok := v.MemoryStats.Stats["inactive_file"]; ok {
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

			// Memory limit logic
			memLimit := float64(v.MemoryStats.Limit)
			if configuredMemoryLimit > 0 && (memLimit == 0 || memLimit > float64(configuredMemoryLimit)*2) {
				memLimit = float64(configuredMemoryLimit)
			} else if configuredMemoryLimit == 0 && memLimit > 2*1024*1024*1024 {
				memLimit = 512 * 1024 * 1024
			}

			broadcast(ContainerResourceStats{
				CPUPercent:  cpuPercent,
				Memory:      memUsage,
				MemoryLimit: memLimit,
				DiskRead:    diskRead,
				DiskWrite:   diskWrite,
				DiskUsage:   diskUsage,
				DiskLimit:   float64(configuredDiskLimit),
				NetRx:       netRx,
				NetTx:       netTx,
			})
		}
	}
}

// getContainerDiskUsage calculates disk usage of /home/user inside the container
func (m *Manager) getContainerDiskUsage(ctx context.Context, containerID string) float64 {
	execConfig := container.ExecOptions{
		Cmd:          []string{"du", "-sk", "/home/user"},
		AttachStdout: true,
		AttachStderr: false,
	}

	execResp, err := m.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return 0
	}

	attachResp, err := m.client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return 0
	}
	defer attachResp.Close()

	// Read output
	var output strings.Builder
	buf := make([]byte, 1024)
	for {
		n, err := attachResp.Reader.Read(buf)
		if n > 0 {
			output.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	// Parse output "12345   /home/user"
	fields := strings.Fields(output.String())
	if len(fields) > 0 {
		if sizeKB, err := strconv.ParseFloat(fields[0], 64); err == nil {
			return sizeKB * 1024 // Convert KB to Bytes
		}
	}
	return 0
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
