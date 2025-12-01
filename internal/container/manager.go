package container

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
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
	"ubuntu":     "ubuntu:22.04",
	"ubuntu-24":  "ubuntu:24.04",
	"ubuntu-20":  "ubuntu:20.04",
	"debian":     "debian:bookworm",
	"debian-11":  "debian:bullseye",
	"kali":       "kalilinux/kali-rolling:latest",
	"parrot":     "parrotsec/core:latest",
	"mint":       "linuxmintd/mint21-amd64:latest",
	"popos":      "pop-os/pop:22.04",
	"elementary": "elementary/docker:latest",
	"zorin":      "zorinos/zorin:latest",
	"mxlinux":    "mxlinux/mx:latest",
	"devuan":     "devuan/devuan:daedalus",
	"antix":      "antix/antix:latest",
	// Security & Penetration Testing
	"blackarch":   "blackarchlinux/blackarch:latest",
	"backbox":     "backbox/backbox:latest",
	"dracos":      "dracos/dracos:latest",
	"pentoo":      "pentoo/pentoo:latest",
	"samurai":     "samurai/samurai-wtf:latest",
	"kali-purple": "kalilinux/kali-purple:latest",
	// Red Hat-based
	"fedora":     "fedora:latest",
	"fedora-39":  "fedora:39",
	"centos":     "centos:stream9",
	"rocky":      "rockylinux:9",
	"alma":       "almalinux:9",
	"oracle":     "oraclelinux:9",
	"rhel":       "redhat/ubi9:latest",
	"openeuler":  "openeuler/openeuler:latest",
	"springdale": "springdale/springdale:latest",
	"navy":       "navylinux/navy:latest",
	// Arch-based
	"archlinux":   "archlinux:latest",
	"manjaro":     "manjarolinux/base:latest",
	"endeavouros": "endeavouros/endeavouros:latest",
	"garuda":      "garudalinux/garuda:latest",
	"arcolinux":   "arcolinux/arcolinux:latest",
	"artix":       "artixlinux/artix:latest",
	// SUSE-based
	"opensuse":   "opensuse/leap:latest",
	"tumbleweed": "opensuse/tumbleweed:latest",
	"sles":       "registry.suse.com/suse/sles:latest",
	// Independent Distributions
	"gentoo":    "gentoo/stage3:latest",
	"void":      "voidlinux/voidlinux:latest",
	"nixos":     "nixos/nix:latest",
	"slackware": "vbatts/slackware:latest",
	"solus":     "solus/solus:latest",
	"pclinuxos": "pclinuxos/pclinuxos:latest",
	// Minimal / Embedded
	"alpine":      "alpine:latest",
	"alpine-3.18": "alpine:3.18",
	"tinycore":    "tinycore/tinycore:latest",
	"puppy":       "puppylinux/puppy:latest",
	"dsl":         "damnsmalllinux/dsl:latest",
	"busybox":     "busybox:latest",
	// Container / Cloud Optimized
	"flatcar":      "flatcar/flatcar:latest",
	"rancheros":    "rancher/os:latest",
	"bottlerocket": "bottlerocket/bottlerocket:latest",
	"talos":        "ghcr.io/siderolabs/talos:latest",
	"k3os":         "rancher/k3os:latest",
	// BSD Systems
	"freebsd":      "freebsd/freebsd:latest",
	"openbsd":      "openbsd/openbsd:latest",
	"netbsd":       "netbsd/netbsd:latest",
	"dragonflybsd": "dragonflybsd/dragonfly:latest",
	// Special Purpose
	"qubes":      "qubes/qubes:latest",
	"tails":      "tails/tails:latest",
	"whonix":     "whonix/whonix:latest",
	"raspbian":   "raspbian/raspbian:latest",
	"ubuntucore": "ubuntu:core22",
	// Gaming / Desktop
	"steamos":   "steamos/steamos:latest",
	"chimeraos": "chimeraos/chimeraos:latest",
	// Developer / Scientific
	"ubuntustudio": "ubuntustudio/ubuntustudio:latest",
	"scientific":   "scientificlinux/sl:latest",
	// Cloud Provider Specific
	"amazonlinux":      "amazonlinux:2023",
	"amazonlinux2":     "amazonlinux:2",
	"amazonlinux2022":  "amazonlinux:2022",
	"cos":              "gcr.io/cos-cloud/cos-stable:latest",
	"azurelinux":       "mcr.microsoft.com/cbl-mariner/base/core:2.0",
	"oracleautonomous": "oraclelinux:8-slim",
	"alibabacloud":     "registry.cn-hangzhou.aliyuncs.com/alinux/aliyunlinux3",
	"ibmcloud":         "icr.io/ibm/ibmcloud-cli:latest",
	"digitalocean":     "ubuntu:22.04",
	// Specialized
	"clearlinux": "clearlinux:latest",
	"photon":     "photon:latest",
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
		{Name: "mint", DisplayName: "Linux Mint", Description: "User-friendly Ubuntu derivative with Cinnamon desktop", Category: "debian", Tags: []string{"desktop", "beginner-friendly"}, Popular: true},
		{Name: "popos", DisplayName: "Pop!_OS", Description: "System76's developer-focused Ubuntu-based distro", Category: "debian", Tags: []string{"desktop", "developer"}, Popular: true},
		{Name: "elementary", DisplayName: "elementary OS", Description: "Beautiful and privacy-focused desktop OS", Category: "debian", Tags: []string{"desktop", "beautiful"}, Popular: false},
		{Name: "zorin", DisplayName: "Zorin OS", Description: "Windows-like interface for easy transition", Category: "debian", Tags: []string{"desktop", "windows-like"}, Popular: false},
		{Name: "mxlinux", DisplayName: "MX Linux", Description: "Efficient Debian-stable based distro", Category: "debian", Tags: []string{"stable", "efficient"}, Popular: true},
		{Name: "devuan", DisplayName: "Devuan", Description: "Debian without systemd", Category: "debian", Tags: []string{"init-freedom", "advanced"}, Popular: false},
		{Name: "antix", DisplayName: "antiX", Description: "Fast, lightweight Debian-based for old computers", Category: "debian", Tags: []string{"lightweight", "fast"}, Popular: false},

		// Security & Penetration Testing
		{Name: "kali", DisplayName: "Kali Linux", Description: "Penetration testing and security auditing platform", Category: "security", Tags: []string{"pentesting", "security", "hacking"}, Popular: true},
		{Name: "parrot", DisplayName: "Parrot Security OS", Description: "Security and privacy focused distribution", Category: "security", Tags: []string{"security", "privacy", "pentesting"}, Popular: true},
		{Name: "blackarch", DisplayName: "BlackArch Linux", Description: "Arch-based pentesting distro with 2500+ tools", Category: "security", Tags: []string{"pentesting", "arch", "advanced"}, Popular: true},
		{Name: "backbox", DisplayName: "BackBox", Description: "Ubuntu-based security assessment toolkit", Category: "security", Tags: []string{"pentesting", "ubuntu"}, Popular: false},
		{Name: "dracos", DisplayName: "Dracos Linux", Description: "Arch-based penetration testing OS", Category: "security", Tags: []string{"pentesting", "arch"}, Popular: false},
		{Name: "pentoo", DisplayName: "Pentoo", Description: "Gentoo-based security-focused distro", Category: "security", Tags: []string{"pentesting", "gentoo"}, Popular: false},
		{Name: "samurai", DisplayName: "Samurai WTF", Description: "Web penetration testing framework", Category: "security", Tags: []string{"pentesting", "web"}, Popular: false},
		{Name: "kali-purple", DisplayName: "Kali Purple", Description: "Defensive security operations platform", Category: "security", Tags: []string{"security", "defensive", "soc"}, Popular: false},

		// Red Hat-based
		{Name: "fedora", DisplayName: "Fedora (Latest)", Description: "Cutting-edge features and technologies", Category: "redhat", Tags: []string{"latest", "developer"}, Popular: true},
		{Name: "fedora-39", DisplayName: "Fedora 39", Description: "Stable Fedora release", Category: "redhat", Tags: []string{"stable"}, Popular: false},
		{Name: "centos", DisplayName: "CentOS Stream 9", Description: "Enterprise Linux for development", Category: "redhat", Tags: []string{"enterprise", "rhel"}, Popular: false},
		{Name: "rocky", DisplayName: "Rocky Linux 9", Description: "Enterprise-grade RHEL-compatible OS", Category: "redhat", Tags: []string{"enterprise", "rhel-compatible"}, Popular: true},
		{Name: "alma", DisplayName: "AlmaLinux 9", Description: "Community-driven RHEL fork", Category: "redhat", Tags: []string{"enterprise", "rhel-compatible"}, Popular: true},
		{Name: "oracle", DisplayName: "Oracle Linux 9", Description: "Oracle's enterprise Linux", Category: "redhat", Tags: []string{"enterprise", "oracle"}, Popular: false},
		{Name: "rhel", DisplayName: "Red Hat Enterprise Linux (UBI)", Description: "Industry-leading enterprise Linux", Category: "redhat", Tags: []string{"enterprise", "commercial"}, Popular: true},
		{Name: "openeuler", DisplayName: "openEuler", Description: "Enterprise Linux by Huawei", Category: "redhat", Tags: []string{"enterprise", "china"}, Popular: false},
		{Name: "springdale", DisplayName: "Springdale Linux", Description: "RHEL rebuild from Princeton", Category: "redhat", Tags: []string{"enterprise", "rhel-compatible"}, Popular: false},
		{Name: "navy", DisplayName: "Navy Linux", Description: "RHEL-based derivative", Category: "redhat", Tags: []string{"enterprise", "rhel-compatible"}, Popular: false},

		// Arch-based
		{Name: "archlinux", DisplayName: "Arch Linux", Description: "Rolling release with latest packages and AUR", Category: "arch", Tags: []string{"rolling", "bleeding-edge", "aur"}, Popular: true},
		{Name: "manjaro", DisplayName: "Manjaro", Description: "User-friendly Arch with easy installation", Category: "arch", Tags: []string{"rolling", "beginner-friendly"}, Popular: true},
		{Name: "endeavouros", DisplayName: "EndeavourOS", Description: "Near-vanilla Arch with graphical installer", Category: "arch", Tags: []string{"rolling", "arch-like"}, Popular: true},
		{Name: "garuda", DisplayName: "Garuda Linux", Description: "Gaming-focused Arch with performance tweaks", Category: "arch", Tags: []string{"gaming", "performance", "rolling"}, Popular: false},
		{Name: "arcolinux", DisplayName: "ArcoLinux", Description: "Educational Arch-based learning platform", Category: "arch", Tags: []string{"educational", "rolling"}, Popular: false},
		{Name: "artix", DisplayName: "Artix Linux", Description: "Arch without systemd, init freedom", Category: "arch", Tags: []string{"rolling", "init-freedom"}, Popular: false},

		// SUSE-based
		{Name: "opensuse", DisplayName: "openSUSE Leap", Description: "Stable enterprise-grade openSUSE", Category: "suse", Tags: []string{"enterprise", "stable", "zypper"}, Popular: true},
		{Name: "tumbleweed", DisplayName: "openSUSE Tumbleweed", Description: "Rolling release openSUSE", Category: "suse", Tags: []string{"rolling", "testing"}, Popular: false},
		{Name: "sles", DisplayName: "SUSE Linux Enterprise", Description: "Commercial enterprise Linux", Category: "suse", Tags: []string{"enterprise", "commercial"}, Popular: false},

		// Independent Distributions
		{Name: "gentoo", DisplayName: "Gentoo Linux", Description: "Source-based with extreme customization", Category: "independent", Tags: []string{"source-based", "advanced", "performance"}, Popular: false},
		{Name: "void", DisplayName: "Void Linux", Description: "Independent distro with runit init system", Category: "independent", Tags: []string{"independent", "runit", "rolling"}, Popular: false},
		{Name: "nixos", DisplayName: "NixOS", Description: "Declarative configuration and reproducible builds", Category: "independent", Tags: []string{"declarative", "nix", "reproducible"}, Popular: false},
		{Name: "slackware", DisplayName: "Slackware", Description: "Oldest maintained Linux distro, Unix-like", Category: "independent", Tags: []string{"classic", "stable", "unix-like"}, Popular: false},
		{Name: "solus", DisplayName: "Solus", Description: "Independent desktop-focused distro", Category: "independent", Tags: []string{"desktop", "modern"}, Popular: false},
		{Name: "pclinuxos", DisplayName: "PCLinuxOS", Description: "Independent user-friendly distro", Category: "independent", Tags: []string{"desktop", "kde"}, Popular: false},

		// Minimal / Embedded
		{Name: "alpine", DisplayName: "Alpine Linux", Description: "Lightweight and security-oriented for containers", Category: "minimal", Tags: []string{"minimal", "docker", "security"}, Popular: true},
		{Name: "alpine-3.18", DisplayName: "Alpine 3.18", Description: "Stable Alpine release", Category: "minimal", Tags: []string{"minimal", "stable"}, Popular: false},
		{Name: "tinycore", DisplayName: "Tiny Core Linux", Description: "Ultra-minimal distro (11-16MB)", Category: "minimal", Tags: []string{"minimal", "tiny", "embedded"}, Popular: false},
		{Name: "puppy", DisplayName: "Puppy Linux", Description: "Lightweight, runs entirely in RAM", Category: "minimal", Tags: []string{"lightweight", "ram", "old-hardware"}, Popular: false},
		{Name: "dsl", DisplayName: "Damn Small Linux", Description: "Extremely small distro for old hardware", Category: "minimal", Tags: []string{"minimal", "tiny", "old-hardware"}, Popular: false},
		{Name: "busybox", DisplayName: "BusyBox", Description: "Ultra-minimal Unix utilities in single binary", Category: "minimal", Tags: []string{"minimal", "embedded"}, Popular: false},

		// Container / Cloud Optimized
		{Name: "flatcar", DisplayName: "Flatcar Container Linux", Description: "Container-optimized OS, CoreOS successor", Category: "container", Tags: []string{"containers", "immutable", "cloud"}, Popular: true},
		{Name: "rancheros", DisplayName: "RancherOS", Description: "Entire OS as Docker containers", Category: "container", Tags: []string{"containers", "docker", "minimal"}, Popular: false},
		{Name: "bottlerocket", DisplayName: "Bottlerocket", Description: "AWS's minimal container-focused OS", Category: "container", Tags: []string{"containers", "aws", "minimal"}, Popular: true},
		{Name: "talos", DisplayName: "Talos Linux", Description: "Kubernetes-native, API-managed OS", Category: "container", Tags: []string{"kubernetes", "immutable", "api"}, Popular: true},
		{Name: "k3os", DisplayName: "k3OS", Description: "Lightweight Kubernetes OS", Category: "container", Tags: []string{"kubernetes", "lightweight"}, Popular: false},

		// BSD Systems
		{Name: "freebsd", DisplayName: "FreeBSD", Description: "Most popular BSD Unix-like system", Category: "bsd", Tags: []string{"unix", "bsd", "stable"}, Popular: true},
		{Name: "openbsd", DisplayName: "OpenBSD", Description: "Security-focused BSD system", Category: "bsd", Tags: []string{"unix", "bsd", "security"}, Popular: true},
		{Name: "netbsd", DisplayName: "NetBSD", Description: "Highly portable BSD system", Category: "bsd", Tags: []string{"unix", "bsd", "portable"}, Popular: false},
		{Name: "dragonflybsd", DisplayName: "DragonFly BSD", Description: "Fork of FreeBSD focused on SMP", Category: "bsd", Tags: []string{"unix", "bsd", "performance"}, Popular: false},

		// Special Purpose
		{Name: "qubes", DisplayName: "Qubes OS", Description: "Security-focused with VM isolation", Category: "special", Tags: []string{"security", "isolation", "privacy"}, Popular: false},
		{Name: "tails", DisplayName: "Tails", Description: "Privacy and anonymity via Tor", Category: "special", Tags: []string{"privacy", "tor", "anonymity"}, Popular: false},
		{Name: "whonix", DisplayName: "Whonix", Description: "Tor-based anonymous operating system", Category: "special", Tags: []string{"privacy", "tor", "anonymity"}, Popular: false},
		{Name: "raspbian", DisplayName: "Raspberry Pi OS", Description: "Official OS for Raspberry Pi", Category: "special", Tags: []string{"raspberry-pi", "arm", "iot"}, Popular: false},
		{Name: "ubuntucore", DisplayName: "Ubuntu Core", Description: "IoT-focused Ubuntu with snaps", Category: "special", Tags: []string{"iot", "snaps", "embedded"}, Popular: false},

		// Gaming / Desktop
		{Name: "steamos", DisplayName: "SteamOS", Description: "Valve's gaming-focused Linux", Category: "gaming", Tags: []string{"gaming", "steam", "desktop"}, Popular: false},
		{Name: "chimeraos", DisplayName: "ChimeraOS", Description: "Gaming-focused couch experience", Category: "gaming", Tags: []string{"gaming", "steam", "desktop"}, Popular: false},

		// Developer / Scientific
		{Name: "ubuntustudio", DisplayName: "Ubuntu Studio", Description: "Multimedia content creation platform", Category: "developer", Tags: []string{"multimedia", "creative", "audio"}, Popular: false},
		{Name: "scientific", DisplayName: "Scientific Linux", Description: "For scientific computing and research", Category: "developer", Tags: []string{"scientific", "research", "rhel"}, Popular: false},

		// Cloud Provider Specific
		{Name: "amazonlinux", DisplayName: "Amazon Linux 2023", Description: "Optimized for AWS cloud", Category: "cloud", Tags: []string{"aws", "cloud", "enterprise"}, Popular: true},
		{Name: "amazonlinux2", DisplayName: "Amazon Linux 2", Description: "Long-term support release for AWS", Category: "cloud", Tags: []string{"aws", "cloud", "lts"}, Popular: true},
		{Name: "amazonlinux2022", DisplayName: "Amazon Linux 2022", Description: "Previous AWS Linux version", Category: "cloud", Tags: []string{"aws", "cloud"}, Popular: false},
		{Name: "cos", DisplayName: "Container-Optimized OS", Description: "Google Cloud's optimized OS", Category: "cloud", Tags: []string{"gcp", "google", "container"}, Popular: true},
		{Name: "azurelinux", DisplayName: "Azure Linux (Mariner)", Description: "Microsoft's internal Linux distro", Category: "cloud", Tags: []string{"azure", "microsoft", "cloud"}, Popular: true},
		{Name: "oracleautonomous", DisplayName: "Oracle Autonomous Linux", Description: "Autonomous management for Oracle Cloud", Category: "cloud", Tags: []string{"oracle", "cloud", "autonomous"}, Popular: false},
		{Name: "alibabacloud", DisplayName: "Alibaba Cloud Linux", Description: "Optimized for Alibaba Cloud", Category: "cloud", Tags: []string{"alibaba", "cloud", "china"}, Popular: false},
		{Name: "ibmcloud", DisplayName: "IBM Cloud CLI", Description: "IBM Cloud tools environment", Category: "cloud", Tags: []string{"ibm", "cloud", "cli"}, Popular: false},
		{Name: "digitalocean", DisplayName: "DigitalOcean Droplet", Description: "Optimized Ubuntu for DigitalOcean", Category: "cloud", Tags: []string{"digitalocean", "cloud", "ubuntu"}, Popular: true},

		// Specialized
		{Name: "clearlinux", DisplayName: "Clear Linux", Description: "Intel-optimized for performance", Category: "specialized", Tags: []string{"performance", "intel", "cloud"}, Popular: false},
		{Name: "photon", DisplayName: "VMware Photon OS", Description: "Optimized for VMware and containers", Category: "specialized", Tags: []string{"vmware", "container", "minimal"}, Popular: false},
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
	"ubuntu-20":  "/bin/bash",
	"debian":     "/bin/bash",
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
	"blackarch":   "/bin/bash",
	"backbox":     "/bin/bash",
	"dracos":      "/bin/bash",
	"pentoo":      "/bin/bash",
	"samurai":     "/bin/bash",
	"kali-purple": "/bin/bash",
	// Red Hat-based
	"fedora":     "/bin/bash",
	"fedora-39":  "/bin/bash",
	"centos":     "/bin/bash",
	"rocky":      "/bin/bash",
	"alma":       "/bin/bash",
	"oracle":     "/bin/bash",
	"rhel":       "/bin/bash",
	"openeuler":  "/bin/bash",
	"springdale": "/bin/bash",
	"navy":       "/bin/bash",
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
	// Independent
	"gentoo":    "/bin/bash",
	"void":      "/bin/bash",
	"nixos":     "/bin/bash",
	"slackware": "/bin/bash",
	"solus":     "/bin/bash",
	"pclinuxos": "/bin/bash",
	// Minimal (use sh)
	"alpine":      "/bin/sh",
	"alpine-3.18": "/bin/sh",
	"tinycore":    "/bin/sh",
	"puppy":       "/bin/sh",
	"dsl":         "/bin/sh",
	"busybox":     "/bin/sh",
	// Container optimized
	"flatcar":      "/bin/bash",
	"rancheros":    "/bin/bash",
	"bottlerocket": "/bin/bash",
	"talos":        "/bin/sh",
	"k3os":         "/bin/sh",
	// BSD (use sh as default)
	"freebsd":      "/bin/sh",
	"openbsd":      "/bin/sh",
	"netbsd":       "/bin/sh",
	"dragonflybsd": "/bin/sh",
	// Special Purpose
	"qubes":      "/bin/bash",
	"tails":      "/bin/bash",
	"whonix":     "/bin/bash",
	"raspbian":   "/bin/bash",
	"ubuntucore": "/bin/bash",
	// Gaming
	"steamos":   "/bin/bash",
	"chimeraos": "/bin/bash",
	// Developer
	"ubuntustudio": "/bin/bash",
	"scientific":   "/bin/bash",
	// Cloud Provider Specific
	"amazonlinux":      "/bin/bash",
	"amazonlinux2":     "/bin/bash",
	"amazonlinux2022":  "/bin/bash",
	"cos":              "/bin/bash",
	"azurelinux":       "/bin/bash",
	"oracleautonomous": "/bin/bash",
	"alibabacloud":     "/bin/bash",
	"ibmcloud":         "/bin/bash",
	"digitalocean":     "/bin/bash",
	// Specialized
	"clearlinux": "/bin/bash",
	"photon":     "/bin/bash",
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
// Supports local Docker socket or remote Docker host via DOCKER_HOST environment variable.
// For remote connections, set:
//   - DOCKER_HOST=tcp://host:2376 (TLS) or tcp://host:2375 (no TLS)
//   - DOCKER_TLS_VERIFY=1 (for TLS connections)
//   - DOCKER_CERT_PATH=/path/to/certs (for TLS connections)
func NewManager(volumePaths ...string) (*Manager, error) {
	dockerHost := os.Getenv("DOCKER_HOST")

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
			// Provide helpful error message for remote Docker host issues
			errMsg := fmt.Sprintf("failed to connect to remote Docker daemon at %s", dockerHost)
			if strings.Contains(dockerHost, ":2376") {
				errMsg += " (TLS enabled - check DOCKER_TLS_VERIFY and DOCKER_CERT_PATH)"
			} else if strings.Contains(dockerHost, "ssh://") {
				errMsg += " (SSH connection - check SSH_PRIVATE_KEY and host accessibility)"
			}
			return nil, fmt.Errorf("%s: %w", errMsg, err)
		}
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

	// Apply tier-based resource limits
	switch cfg.Tier {
	case "pro":
		containerCfg.MemoryLimit = 2048 * 1024 * 1024 // 2GB
		containerCfg.CPULimit = 200000                // 2 CPUs
	case "enterprise":
		containerCfg.MemoryLimit = 4096 * 1024 * 1024 // 4GB
		containerCfg.CPULimit = 400000                // 4 CPUs
	default: // free/guest
		containerCfg.MemoryLimit = 512 * 1024 * 1024 // 512MB
		containerCfg.CPULimit = 100000               // 1 CPU
	}

	return m.CreateContainer(ctx, containerCfg)
}
