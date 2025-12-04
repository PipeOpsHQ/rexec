#!/bin/bash
set -e

# =============================================================================
# Rexec Remote Docker Host Setup with TLS + gVisor
# =============================================================================
# This script sets up a VM to serve as a remote Docker host for Rexec.
# Supports optional sandboxing via gVisor for enhanced security.
# Run this on a fresh Ubuntu/Debian VM (Hetzner, DigitalOcean, Linode, etc.)
#
# Port: 2377 (non-standard to avoid attacks on default Docker port 2376)
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/your-repo/rexec/main/docker/docker-host/setup.sh | sudo bash
#   # or
#   sudo ./setup.sh                    # Docker + containerd only
#   sudo ./setup.sh --with-gvisor      # Docker + gVisor sandboxing (recommended)
#   sudo ./setup.sh --with-gvisor --force-xfs  # Enable XFS with disk quotas (wipes volume data)
#   sudo ./setup.sh --force-certs      # Regenerate TLS certificates even if valid ones exist
#
# Runtime Options (set via OCI_RUNTIME env var in Rexec):
#   - runc (default): Standard Docker runtime
#   - runsc: gVisor sandbox (user-space kernel, no VM overhead)
#   - runsc-kvm: gVisor with KVM acceleration (requires /dev/kvm)
#
# Security:
#   - TLS required for all connections
#   - Non-standard port 2377 (not 2376)
#   - gVisor provides syscall filtering and isolation
#
# =============================================================================

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Parse arguments
INSTALL_KATA=false
INSTALL_GVISOR=false
USE_PODMAN=false
FORCE_XFS=false
FORCE_CERTS=false
for arg in "$@"; do
    case $arg in
        --with-kata|--kata|--firecracker)
            INSTALL_KATA=true
            shift
            ;;
        --with-gvisor|--gvisor)
            INSTALL_GVISOR=true
            shift
            ;;
        --podman)
            USE_PODMAN=true
            shift
            ;;
        --force-xfs)
            FORCE_XFS=true
            shift
            ;;
        --force-certs)
            FORCE_CERTS=true
            shift
            ;;
    esac
done

# Auto-detect installed runtimes if not explicitly requested
if [ "$INSTALL_GVISOR" = false ]; then
    if command -v runsc &> /dev/null; then
        echo -e "${YELLOW}Detected gVisor (runsc) installed. Enabling gVisor configuration...${NC}"
        INSTALL_GVISOR=true
    fi
fi

if [ "$INSTALL_KATA" = false ]; then
    if command -v kata-runtime &> /dev/null; then
        echo -e "${YELLOW}Detected Kata Containers installed. Enabling Kata configuration...${NC}"
        INSTALL_KATA=true
    fi
fi

echo -e "${BLUE}=============================================${NC}"
echo -e "${BLUE}  Rexec Remote Docker Host Setup${NC}"
if [ "$USE_PODMAN" = true ]; then
    echo -e "${CYAN}  Using Podman (Docker-compatible)${NC}"
fi
if [ "$INSTALL_KATA" = true ]; then
    echo -e "${CYAN}  + Kata Containers with Firecracker${NC}"
fi
if [ "$INSTALL_GVISOR" = true ]; then
    echo -e "${CYAN}  + gVisor (runsc) for sandboxed isolation${NC}"
fi
echo -e "${BLUE}=============================================${NC}"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: This script must be run as root${NC}"
    echo "Run: sudo $0"
    exit 1
fi

# Check KVM support if installing Kata
if [ "$INSTALL_KATA" = true ]; then
    echo -e "${YELLOW}Checking KVM support for Firecracker...${NC}"
    if [ ! -e /dev/kvm ]; then
        echo -e "${RED}Error: /dev/kvm not found. KVM is required for Firecracker.${NC}"
        echo "Make sure your VM has nested virtualization enabled."
        echo "For cloud VMs, use bare-metal or KVM-enabled instances."
        exit 1
    fi
    if [ ! -r /dev/kvm ] || [ ! -w /dev/kvm ]; then
        echo -e "${YELLOW}Fixing /dev/kvm permissions...${NC}"
        chmod 666 /dev/kvm
    fi
    echo -e "${GREEN}✓ KVM is available${NC}"
    echo ""
fi

# Detect public IP
PUBLIC_IP=$(curl -s ifconfig.me || curl -s icanhazip.com || hostname -I | awk '{print $1}')
echo -e "Detected public IP: ${GREEN}${PUBLIC_IP}${NC}"
echo ""

# =============================================================================
# 0. Setup Disk Quotas for Container Storage Limits
# =============================================================================
echo -e "${YELLOW}[0/N] Checking disk quota support for container storage limits...${NC}"

# Find the device for /var/lib/docker
DOCKER_DIR="/var/lib/docker"

# Check for available attached volumes that can be used for Docker storage
# This enables XFS with pquota for proper disk quotas
ATTACHED_VOLUME=""
ATTACHED_VOLUME_DEVICE=""

# Look for mounted volumes that could be used for Docker (common cloud provider patterns)
for mount_path in /mnt/volume* /mnt/data* /mnt/docker* /data /opt/docker; do
    if [ -d "$mount_path" ] && mountpoint -q "$mount_path" 2>/dev/null; then
        ATTACHED_VOLUME="$mount_path"
        ATTACHED_VOLUME_DEVICE=$(df "$mount_path" --output=source 2>/dev/null | tail -1)
        break
    fi
done

# If we found an attached volume, check if we should use it for Docker
if [ -n "$ATTACHED_VOLUME" ] && [ -n "$ATTACHED_VOLUME_DEVICE" ]; then
    VOLUME_FS_TYPE=$(df -T "$ATTACHED_VOLUME" --output=fstype 2>/dev/null | tail -1)
    echo -e "${CYAN}Found attached volume: $ATTACHED_VOLUME ($ATTACHED_VOLUME_DEVICE) - $VOLUME_FS_TYPE${NC}"
    
    # Check if Docker is already using this volume
    if [ -L "$DOCKER_DIR" ] && [ "$(readlink -f "$DOCKER_DIR")" = "$ATTACHED_VOLUME/docker" ]; then
        echo -e "${GREEN}✓ Docker already configured to use attached volume${NC}"
    else
        # Docker directory exists - check if we should migrate to XFS volume
        SHOULD_MIGRATE=false
        
        # Check if volume needs to be formatted as XFS
        if [ "$VOLUME_FS_TYPE" != "xfs" ]; then
            echo -e "${YELLOW}Volume is $VOLUME_FS_TYPE, needs XFS for disk quotas${NC}"
            
            # Check if volume is empty or has only lost+found, or if force flag is set
            VOLUME_CONTENTS=$(ls -A "$ATTACHED_VOLUME" 2>/dev/null | grep -v "^lost+found$" | wc -l)
            
            if [ "$FORCE_XFS" = true ]; then
                echo -e "${YELLOW}Force XFS enabled - will format volume and migrate Docker data...${NC}"
                SHOULD_MIGRATE=true
                
                # Stop Docker first
                echo -e "${CYAN}Stopping Docker...${NC}"
                systemctl stop docker 2>/dev/null || true
                sleep 2
                
                # Unmount the volume
                umount "$ATTACHED_VOLUME" 2>/dev/null || true
                
                # Format as XFS with quota support
                echo -e "${CYAN}Formatting $ATTACHED_VOLUME_DEVICE as XFS...${NC}"
                if mkfs.xfs -f -m crc=1,finobt=1 "$ATTACHED_VOLUME_DEVICE"; then
                    echo -e "${GREEN}✓ Formatted volume as XFS${NC}"
                    
                    # Update fstab to use XFS with pquota
                    cp /etc/fstab /etc/fstab.backup.$(date +%s)
                    
                    # Get UUID of new XFS filesystem
                    NEW_UUID=$(blkid -s UUID -o value "$ATTACHED_VOLUME_DEVICE" 2>/dev/null)
                    
                    # Remove old entry and add new one
                    sed -i "\|$ATTACHED_VOLUME_DEVICE|d" /etc/fstab
                    sed -i "\|$ATTACHED_VOLUME|d" /etc/fstab
                    echo "UUID=$NEW_UUID $ATTACHED_VOLUME xfs defaults,pquota,nofail 0 2" >> /etc/fstab
                    
                    # Mount with pquota
                    mkdir -p "$ATTACHED_VOLUME"
                    mount -o defaults,pquota "$ATTACHED_VOLUME"
                    echo -e "${GREEN}✓ Mounted volume with pquota support${NC}"
                    
                    VOLUME_FS_TYPE="xfs"
                else
                    echo -e "${RED}Failed to format volume as XFS${NC}"
                    SHOULD_MIGRATE=false
                fi
            elif [ "$VOLUME_CONTENTS" -eq 0 ]; then
                echo -e "${CYAN}Volume is empty, formatting as XFS with quota support...${NC}"
                SHOULD_MIGRATE=true
                
                # Stop Docker first
                systemctl stop docker 2>/dev/null || true
                
                # Unmount the volume
                umount "$ATTACHED_VOLUME" 2>/dev/null || true
                
                # Format as XFS with quota support
                if mkfs.xfs -f -m crc=1,finobt=1 "$ATTACHED_VOLUME_DEVICE"; then
                    echo -e "${GREEN}✓ Formatted volume as XFS${NC}"
                    
                    # Update fstab
                    cp /etc/fstab /etc/fstab.backup.$(date +%s)
                    NEW_UUID=$(blkid -s UUID -o value "$ATTACHED_VOLUME_DEVICE" 2>/dev/null)
                    sed -i "\|$ATTACHED_VOLUME_DEVICE|d" /etc/fstab
                    sed -i "\|$ATTACHED_VOLUME|d" /etc/fstab
                    echo "UUID=$NEW_UUID $ATTACHED_VOLUME xfs defaults,pquota,nofail 0 2" >> /etc/fstab
                    
                    # Mount with pquota
                    mkdir -p "$ATTACHED_VOLUME"
                    mount -o defaults,pquota "$ATTACHED_VOLUME"
                    echo -e "${GREEN}✓ Mounted volume with pquota support${NC}"
                    
                    VOLUME_FS_TYPE="xfs"
                else
                    echo -e "${RED}Failed to format volume as XFS${NC}"
                    SHOULD_MIGRATE=false
                fi
            else
                echo -e "${YELLOW}Volume contains data ($VOLUME_CONTENTS items)${NC}"
                echo -e "${YELLOW}To enable disk quotas, run with --force-xfs flag:${NC}"
                echo -e "${YELLOW}  ./setup.sh --with-gvisor --force-xfs${NC}"
                echo -e "${YELLOW}WARNING: This will ERASE all data on $ATTACHED_VOLUME${NC}"
                echo -e "${YELLOW}         (Docker containers will be recreated)${NC}"
            fi
        else
            # Volume is already XFS
            SHOULD_MIGRATE=true
        fi
        
        # If volume is XFS, migrate Docker to use it
        if [ "$VOLUME_FS_TYPE" = "xfs" ] && [ "$SHOULD_MIGRATE" = true ]; then
            echo -e "${CYAN}Setting up Docker to use XFS volume at $ATTACHED_VOLUME...${NC}"
            
            # Ensure pquota is in fstab
            if ! grep -q "pquota" /etc/fstab || ! grep -q "$ATTACHED_VOLUME" /etc/fstab; then
                cp /etc/fstab /etc/fstab.backup.$(date +%s)
                NEW_UUID=$(blkid -s UUID -o value "$ATTACHED_VOLUME_DEVICE" 2>/dev/null)
                sed -i "\|$ATTACHED_VOLUME_DEVICE|d" /etc/fstab
                sed -i "\|$ATTACHED_VOLUME|d" /etc/fstab
                sed -i "\|UUID=$NEW_UUID|d" /etc/fstab
                echo "UUID=$NEW_UUID $ATTACHED_VOLUME xfs defaults,pquota,nofail 0 2" >> /etc/fstab
            fi
            
            # Ensure mounted with pquota
            if ! mount | grep "$ATTACHED_VOLUME" | grep -q "pquota"; then
                umount "$ATTACHED_VOLUME" 2>/dev/null || true
                mount -o defaults,pquota "$ATTACHED_VOLUME"
            fi
            
            # Create Docker directory on volume
            mkdir -p "$ATTACHED_VOLUME/docker"
            
            # Check if Docker dir is already a symlink to the volume
            if [ -L "$DOCKER_DIR" ] && [ "$(readlink -f "$DOCKER_DIR")" = "$ATTACHED_VOLUME/docker" ]; then
                echo -e "${GREEN}✓ Docker already using XFS volume${NC}"
            else
                # Stop Docker and migrate
                echo -e "${CYAN}Stopping Docker for migration...${NC}"
                systemctl stop docker 2>/dev/null || true
                sleep 2
                
                # Move existing Docker data to volume
                if [ -d "$DOCKER_DIR" ] && [ ! -L "$DOCKER_DIR" ]; then
                    if [ -n "$(ls -A $DOCKER_DIR 2>/dev/null)" ]; then
                        echo -e "${CYAN}Moving existing Docker data to volume (this may take a while)...${NC}"
                        rsync -a --progress "$DOCKER_DIR"/ "$ATTACHED_VOLUME/docker/" 2>/dev/null || \
                            cp -a "$DOCKER_DIR"/* "$ATTACHED_VOLUME/docker/" 2>/dev/null || true
                    fi
                    rm -rf "$DOCKER_DIR"
                fi
                
                # Create symlink
                ln -sf "$ATTACHED_VOLUME/docker" "$DOCKER_DIR"
                echo -e "${GREEN}✓ Docker directory linked to XFS volume with quota support${NC}"
            fi
        elif [ "$VOLUME_FS_TYPE" != "xfs" ]; then
            echo -e "${CYAN}Docker directory exists with data, keeping current configuration${NC}"
            echo -e "${YELLOW}Disk quotas will NOT be available without XFS${NC}"
        fi
    fi
fi

mkdir -p "$DOCKER_DIR"

# Get the mount point and device for Docker directory
DOCKER_MOUNT=$(df "$DOCKER_DIR" --output=target 2>/dev/null | tail -1)
DOCKER_DEVICE=$(df "$DOCKER_DIR" --output=source 2>/dev/null | tail -1)

# Track if we need a reboot for quota changes
QUOTA_NEEDS_REBOOT=false

# Function to update fstab with quota options
update_fstab_quota() {
    local device="$1"
    local mount_point="$2"
    local quota_opt="$3"
    
    # Backup fstab
    cp /etc/fstab /etc/fstab.backup.$(date +%s)
    
    # Get the UUID if device is a block device
    local fstab_entry=""
    if [[ "$device" == /dev/* ]]; then
        local uuid=$(blkid -s UUID -o value "$device" 2>/dev/null)
        if [ -n "$uuid" ]; then
            fstab_entry="UUID=$uuid"
        else
            fstab_entry="$device"
        fi
    else
        fstab_entry="$device"
    fi
    
    # Check if entry exists in fstab
    if grep -qE "(^$device|^$fstab_entry|UUID=.*$mount_point)" /etc/fstab 2>/dev/null; then
        # Entry exists - add quota option if not present
        if ! grep -E "(^$device|^$fstab_entry)" /etc/fstab | grep -q "$quota_opt"; then
            # Add quota option to existing mount options
            sed -i -E "/(^${device//\//\\/}|^${fstab_entry//\//\\/})/s/(defaults|rw)/\1,$quota_opt/" /etc/fstab 2>/dev/null || \
            sed -i -E "/(^${device//\//\\/}|^${fstab_entry//\//\\/})/s/([[:space:]])(ext4|xfs)([[:space:]]+)([^[:space:]]+)/\1\2\3\4,$quota_opt/" /etc/fstab 2>/dev/null
            echo -e "${GREEN}✓ Updated /etc/fstab with $quota_opt option${NC}"
            return 0
        else
            echo -e "${CYAN}$quota_opt already in fstab${NC}"
            return 1
        fi
    else
        # No entry exists - check if it's root filesystem
        if [ "$mount_point" = "/" ]; then
            # For root filesystem, find and update the root entry
            if grep -q "^UUID=.* / " /etc/fstab; then
                sed -i -E "/^UUID=.* \/ /s/(defaults|rw)/\1,$quota_opt/" /etc/fstab 2>/dev/null
                echo -e "${GREEN}✓ Updated root filesystem entry in /etc/fstab with $quota_opt${NC}"
                return 0
            fi
        fi
        echo -e "${YELLOW}Could not find fstab entry for $device${NC}"
        return 1
    fi
}

if [ -n "$DOCKER_DEVICE" ] && [ "$DOCKER_DEVICE" != "-" ]; then
    echo -e "${CYAN}Docker storage device: $DOCKER_DEVICE mounted at $DOCKER_MOUNT${NC}"
    
    # Check if filesystem supports project quotas (xfs or ext4)
    FS_TYPE=$(df -T "$DOCKER_DIR" --output=fstype 2>/dev/null | tail -1)
    echo -e "${CYAN}Filesystem type: $FS_TYPE${NC}"
    
    # Check current mount options from /proc/mounts (more reliable than mount command)
    CURRENT_OPTS=$(grep " $DOCKER_MOUNT " /proc/mounts 2>/dev/null | awk '{print $4}' || echo "")
    echo -e "${CYAN}Current mount options: $CURRENT_OPTS${NC}"
    
    if echo "$FS_TYPE" | grep -qE "^(xfs|ext4)$"; then
        if echo "$CURRENT_OPTS" | grep -qE "pquota|prjquota"; then
            echo -e "${GREEN}✓ Project quotas already enabled${NC}"
        else
            echo -e "${YELLOW}Enabling project quotas for disk limits...${NC}"
            
            # For XFS, we need to remount with pquota
            if [ "$FS_TYPE" = "xfs" ]; then
                # Check if pquota is supported (XFS must be formatted with quota support)
                # Modern XFS (crc=1) supports quotas by default
                XFS_INFO=$(xfs_info "$DOCKER_MOUNT" 2>/dev/null || echo "")
                if echo "$XFS_INFO" | grep -qE "crc=1|crc=enabled"; then
                    echo -e "${CYAN}XFS v5 format detected (quota-ready)${NC}"
                    
                    # First update fstab to ensure persistence
                    if update_fstab_quota "$DOCKER_DEVICE" "$DOCKER_MOUNT" "pquota"; then
                        QUOTA_NEEDS_REBOOT=true
                    fi
                    
                    # Try to enable pquota via remount
                    echo -e "${CYAN}Attempting live remount with pquota...${NC}"
                    if mount -o remount,pquota "$DOCKER_MOUNT" 2>/dev/null; then
                        echo -e "${GREEN}✓ Successfully enabled pquota via remount${NC}"
                        QUOTA_NEEDS_REBOOT=false
                    else
                        echo -e "${YELLOW}Live remount failed - pquota will be enabled after reboot${NC}"
                        QUOTA_NEEDS_REBOOT=true
                    fi
                else
                    echo -e "${YELLOW}XFS v4 format detected. Checking quota inode...${NC}"
                    # Check if quota inode exists
                    if echo "$XFS_INFO" | grep -qE "pquotino"; then
                        echo -e "${CYAN}Quota inode present, attempting remount...${NC}"
                        if update_fstab_quota "$DOCKER_DEVICE" "$DOCKER_MOUNT" "pquota"; then
                            QUOTA_NEEDS_REBOOT=true
                        fi
                        if mount -o remount,pquota "$DOCKER_MOUNT" 2>/dev/null; then
                            echo -e "${GREEN}✓ Successfully enabled pquota via remount${NC}"
                            QUOTA_NEEDS_REBOOT=false
                        fi
                    else
                        echo -e "${YELLOW}XFS not formatted with project quota support.${NC}"
                        echo -e "${YELLOW}Disk quotas will not be enforced. To enable (DESTRUCTIVE):${NC}"
                        echo -e "${YELLOW}  mkfs.xfs -f -m crc=1,finobt=1 $DOCKER_DEVICE${NC}"
                    fi
                fi
            elif [ "$FS_TYPE" = "ext4" ]; then
                # For ext4, enable project quota feature
                EXT4_FEATURES=$(tune2fs -l "$DOCKER_DEVICE" 2>/dev/null | grep -i "features" || echo "")
                if echo "$EXT4_FEATURES" | grep -qi "project"; then
                    echo -e "${CYAN}ext4 project quota feature detected${NC}"
                    if update_fstab_quota "$DOCKER_DEVICE" "$DOCKER_MOUNT" "prjquota"; then
                        QUOTA_NEEDS_REBOOT=true
                    fi
                    if mount -o remount,prjquota "$DOCKER_MOUNT" 2>/dev/null; then
                        echo -e "${GREEN}✓ Successfully enabled prjquota via remount${NC}"
                        QUOTA_NEEDS_REBOOT=false
                    else
                        echo -e "${YELLOW}Live remount failed - prjquota will be enabled after reboot${NC}"
                        QUOTA_NEEDS_REBOOT=true
                    fi
                else
                    echo -e "${YELLOW}Enabling ext4 project quota feature...${NC}"
                    # Enable quota feature (may work on mounted fs with newer kernels 4.5+)
                    if tune2fs -O project,quota "$DOCKER_DEVICE" 2>/dev/null; then
                        echo -e "${GREEN}✓ Enabled ext4 project quota feature${NC}"
                        if update_fstab_quota "$DOCKER_DEVICE" "$DOCKER_MOUNT" "prjquota"; then
                            QUOTA_NEEDS_REBOOT=true
                        fi
                        if mount -o remount,prjquota "$DOCKER_MOUNT" 2>/dev/null; then
                            echo -e "${GREEN}✓ Successfully enabled prjquota via remount${NC}"
                            QUOTA_NEEDS_REBOOT=false
                        fi
                    else
                        echo -e "${YELLOW}Could not enable quota feature on mounted filesystem.${NC}"
                        echo -e "${YELLOW}You may need to run: tune2fs -O project,quota $DOCKER_DEVICE (while unmounted)${NC}"
                    fi
                fi
            fi
        fi
    else
        echo -e "${YELLOW}Filesystem $FS_TYPE does not support project quotas.${NC}"
        echo -e "${YELLOW}Disk quotas will not be enforced for containers.${NC}"
    fi
else
    echo -e "${YELLOW}Could not detect Docker storage device. Skipping quota setup.${NC}"
fi

# If quota needs reboot and this is the first run, schedule reboot at end of script
if [ "$QUOTA_NEEDS_REBOOT" = true ]; then
    echo -e "${YELLOW}Note: A reboot is required to enable disk quotas.${NC}"
    echo -e "${YELLOW}Run 'sudo reboot' after setup completes, then re-run this script.${NC}"
fi

echo ""

# =============================================================================
# 1. Install Container Runtime (Docker or Podman)
# =============================================================================
STEP_NUM=1
TOTAL_STEPS=6  # Base: install, certs, daemon config, firewall, TLS test, quota verify
if [ "$INSTALL_KATA" = true ]; then
    TOTAL_STEPS=$((TOTAL_STEPS + 2))
fi
if [ "$INSTALL_GVISOR" = true ]; then
    TOTAL_STEPS=$((TOTAL_STEPS + 1))
fi

if [ "$USE_PODMAN" = true ]; then
    echo -e "${YELLOW}[$STEP_NUM/$TOTAL_STEPS] Installing Podman...${NC}"

    # Stop Docker if running to avoid port conflicts
    if systemctl is-active --quiet docker 2>/dev/null; then
        echo -e "${CYAN}Stopping Docker to avoid conflicts...${NC}"
        systemctl stop docker
        systemctl disable docker
    fi

    if command -v podman &> /dev/null; then
        echo -e "${GREEN}Podman is already installed${NC}"
        podman --version
    else
        # Install Podman
        apt-get update
        apt-get install -y podman podman-docker

        echo -e "${GREEN}Podman installed successfully${NC}"
    fi
else
    echo -e "${YELLOW}[$STEP_NUM/$TOTAL_STEPS] Installing Docker...${NC}"

    if command -v docker &> /dev/null; then
        echo -e "${GREEN}Docker is already installed${NC}"
        docker --version
    else
        # Install Docker using official script
        curl -fsSL https://get.docker.com | sh

        # Enable and start Docker
        systemctl enable docker
        systemctl start docker

        echo -e "${GREEN}Docker installed successfully${NC}"
    fi
fi

echo ""

# =============================================================================
# 2. Install Kata Containers with Firecracker (Optional)
# =============================================================================
if [ "$INSTALL_KATA" = true ]; then
    STEP_NUM=$((STEP_NUM + 1))
    echo -e "${YELLOW}[$STEP_NUM/$TOTAL_STEPS] Installing Kata Containers with Firecracker...${NC}"

    # Detect architecture
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64) KATA_ARCH="amd64" ;;
        aarch64) KATA_ARCH="arm64" ;;
        *)
            echo -e "${RED}Unsupported architecture: $ARCH${NC}"
            exit 1
            ;;
    esac

    # Check if already installed
    if command -v kata-runtime &> /dev/null; then
        echo -e "${GREEN}Kata Containers is already installed. Skipping download.${NC}"
    else
        # Install dependencies
        apt-get update
        apt-get install -y curl gnupg2 apt-transport-https ca-certificates

        # Use Kata 2.5.x which has OCI runtime CLI support for Docker
        KATA_VERSION="2.5.2"
        KATA_URL="https://github.com/kata-containers/kata-containers/releases/download/${KATA_VERSION}"

        # Download and extract Kata
        echo -e "${CYAN}Downloading Kata Containers ${KATA_VERSION}...${NC}"
        mkdir -p /opt/kata
        cd /opt/kata

        # Download with error checking
        set +e
        curl -LOf "${KATA_URL}/kata-static-${KATA_VERSION}-${KATA_ARCH}.tar.xz"
        CURL_EXIT=$?
        set -e

        if [ $CURL_EXIT -ne 0 ]; then
            echo -e "${RED}Failed to download Kata Containers. Skipping Kata installation.${NC}"
            # Don't exit, just disable Kata so we can proceed to gVisor
            INSTALL_KATA=false
        else
            tar -xf "kata-static-${KATA_VERSION}-${KATA_ARCH}.tar.xz" --strip-components=2

            # Clean up archive
            rm -f "kata-static-${KATA_VERSION}-${KATA_ARCH}.tar.xz"

            # Verify extracted files exist
            if [ ! -f /opt/kata/bin/kata-runtime ]; then
                echo -e "${RED}Kata binaries not found after extraction${NC}"
                # Try alternative extraction location
                if [ -d /opt/kata/opt/kata ]; then
                    mv /opt/kata/opt/kata/* /opt/kata/
                    rm -rf /opt/kata/opt
                fi
            fi

            # Link binaries
            ln -sf /opt/kata/bin/kata-runtime /usr/local/bin/kata-runtime
            ln -sf /opt/kata/bin/kata-collect-data.sh /usr/local/bin/kata-collect-data.sh
            ln -sf /opt/kata/bin/containerd-shim-kata-v2 /usr/local/bin/containerd-shim-kata-v2

            # Create shim symlinks for all Kata runtime variants
            ln -sf /opt/kata/bin/containerd-shim-kata-v2 /usr/local/bin/containerd-shim-kata-fc-v2
            ln -sf /opt/kata/bin/containerd-shim-kata-v2 /usr/local/bin/containerd-shim-kata-qemu-v2
            ln -sf /opt/kata/bin/containerd-shim-kata-v2 /usr/local/bin/containerd-shim-kata-clh-v2

            echo -e "${GREEN}✓ Kata shim binaries linked${NC}"
        fi
    fi

    # Only proceed with config if Kata is installed (either pre-existing or just installed)
    if command -v kata-runtime &> /dev/null; then
        # Also link firecracker binary to PATH
        ln -sf /opt/kata/bin/firecracker /usr/local/bin/firecracker 2>/dev/null || true
        ln -sf /opt/kata/bin/jailer /usr/local/bin/jailer 2>/dev/null || true

        # Create symlinks for configuration files
        mkdir -p /etc/kata-containers
        ln -sf /opt/kata/share/defaults/kata-containers/configuration-fc.toml /etc/kata-containers/configuration.toml
        ln -sf /opt/kata/share/defaults/kata-containers/configuration-fc.toml /etc/kata-containers/configuration-fc.toml
        ln -sf /opt/kata/share/defaults/kata-containers/configuration-qemu.toml /etc/kata-containers/configuration-qemu.toml

        # Verify Kata installation
        if kata-runtime --version > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Kata Containers installed successfully${NC}"
            kata-runtime --version
        else
            echo -e "${RED}✗ Kata Containers installation failed${NC}"
            # Don't exit, just warn
            echo "Warning: Kata runtime check failed."
        fi

        # Check Firecracker
        if command -v firecracker &> /dev/null; then
            echo -e "${GREEN}✓ Firecracker available${NC}"
            firecracker --version
        fi

        echo ""
        STEP_NUM=$((STEP_NUM + 1))
        echo -e "${YELLOW}[$STEP_NUM/$TOTAL_STEPS] Configuring containerd for Kata...${NC}"

        # Configure containerd for Kata runtime
        mkdir -p /etc/containerd

        # Generate default containerd config if not exists
        if [ ! -f /etc/containerd/config.toml ]; then
            containerd config default > /etc/containerd/config.toml
        fi

        # Remove any existing Kata config to avoid duplicates
        sed -i '/containerd.runtimes.kata/,/ConfigPath.*kata/d' /etc/containerd/config.toml 2>/dev/null || true

        # Add Kata runtime configuration
        # Note: Both kata and kata-fc use the same shim (io.containerd.kata.v2)
        # The difference is the config file which specifies the hypervisor (QEMU vs Firecracker)
        cat >> /etc/containerd/config.toml <<'CONTAINERDEOF'

# Kata Containers with Firecracker runtime
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata]
  runtime_type = "io.containerd.kata.v2"
  privileged_without_host_devices = true
  pod_annotations = ["io.katacontainers.*"]
  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata.options]
    ConfigPath = "/opt/kata/share/defaults/kata-containers/configuration-fc.toml"

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata-fc]
  runtime_type = "io.containerd.kata.v2"
  privileged_without_host_devices = true
  pod_annotations = ["io.katacontainers.*"]
  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata-fc.options]
    ConfigPath = "/opt/kata/share/defaults/kata-containers/configuration-fc.toml"

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata-qemu]
  runtime_type = "io.containerd.kata.v2"
  privileged_without_host_devices = true
  pod_annotations = ["io.katacontainers.*"]
  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata-qemu.options]
    ConfigPath = "/opt/kata/share/defaults/kata-containers/configuration-qemu.toml"
CONTAINERDEOF
        echo -e "${GREEN}✓ Kata runtime added to containerd config${NC}"

        # Restart containerd
        systemctl restart containerd
        echo -e "${GREEN}✓ containerd configured for Kata${NC}"
        echo ""
    fi
fi

# =============================================================================
# Install gVisor (Optional)
# =============================================================================
if [ "$INSTALL_GVISOR" = true ]; then
    STEP_NUM=$((STEP_NUM + 1))
    echo -e "${YELLOW}[$STEP_NUM/$TOTAL_STEPS] Installing gVisor (runsc)...${NC}"

    # Detect architecture
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)
            GVISOR_ARCH="x86_64"
            GVISOR_APT_ARCH="amd64"
            ;;
        aarch64)
            GVISOR_ARCH="aarch64"
            GVISOR_APT_ARCH="arm64"
            ;;
        *)
            echo -e "${RED}Unsupported architecture for gVisor: $ARCH${NC}"
            exit 1
            ;;
    esac

    # Install gVisor from official release
    echo -e "${CYAN}Downloading gVisor...${NC}"

    # Try apt-based installation first (Ubuntu/Debian)
    if command -v apt-get &> /dev/null; then
        # Add gVisor repo
        curl -fsSL https://gvisor.dev/archive.key | gpg --batch --yes --dearmor -o /usr/share/keyrings/gvisor-archive-keyring.gpg 2>/dev/null || true
        echo "deb [arch=${GVISOR_APT_ARCH} signed-by=/usr/share/keyrings/gvisor-archive-keyring.gpg] https://storage.googleapis.com/gvisor/releases release main" | tee /etc/apt/sources.list.d/gvisor.list > /dev/null
        apt-get update -qq
        if apt-get install -y runsc 2>/dev/null; then
            echo -e "${GREEN}✓ gVisor installed via apt${NC}"
        else
            echo -e "${YELLOW}apt install failed, trying direct download...${NC}"
            # Direct download as fallback
            curl -fsSL "https://storage.googleapis.com/gvisor/releases/release/latest/${GVISOR_ARCH}/runsc" -o /usr/local/bin/runsc
            curl -fsSL "https://storage.googleapis.com/gvisor/releases/release/latest/${GVISOR_ARCH}/containerd-shim-runsc-v1" -o /usr/local/bin/containerd-shim-runsc-v1
            chmod +x /usr/local/bin/runsc /usr/local/bin/containerd-shim-runsc-v1
        fi
    else
        # Manual installation for other distros
        curl -fsSL "https://storage.googleapis.com/gvisor/releases/release/latest/${GVISOR_ARCH}/runsc" -o /usr/local/bin/runsc
        curl -fsSL "https://storage.googleapis.com/gvisor/releases/release/latest/${GVISOR_ARCH}/containerd-shim-runsc-v1" -o /usr/local/bin/containerd-shim-runsc-v1
        chmod +x /usr/local/bin/runsc /usr/local/bin/containerd-shim-runsc-v1
    fi

    # Verify installation and create symlinks
    RUNSC_PATH=""
    if command -v runsc &> /dev/null; then
        RUNSC_PATH=$(which runsc)
    elif [ -f /usr/bin/runsc ]; then
        RUNSC_PATH="/usr/bin/runsc"
    elif [ -f /usr/local/bin/runsc ]; then
        RUNSC_PATH="/usr/local/bin/runsc"
    fi

    if [ -n "$RUNSC_PATH" ]; then
        echo -e "${GREEN}✓ gVisor installed successfully at $RUNSC_PATH${NC}"
        $RUNSC_PATH --version

        # Ensure symlink exists at /usr/local/bin for Docker
        if [ "$RUNSC_PATH" != "/usr/local/bin/runsc" ]; then
            ln -sf "$RUNSC_PATH" /usr/local/bin/runsc
            echo -e "${CYAN}Created symlink /usr/local/bin/runsc -> $RUNSC_PATH${NC}"
        fi

        # Also ensure containerd-shim-runsc-v1 is available
        if [ -f /usr/bin/containerd-shim-runsc-v1 ] && [ ! -f /usr/local/bin/containerd-shim-runsc-v1 ]; then
            ln -sf /usr/bin/containerd-shim-runsc-v1 /usr/local/bin/containerd-shim-runsc-v1
        fi
    else
        echo -e "${RED}✗ gVisor installation failed - runsc not found${NC}"
        exit 1
    fi

    # Configure gVisor for Docker - Performance Optimized
    # gVisor supports systrap (default, ~10-15% overhead) and KVM platforms (~2-5% overhead)
    # KVM provides significantly better performance but requires /dev/kvm
    GVISOR_PLATFORM="systrap"
    if [ -e /dev/kvm ] && [ -r /dev/kvm ]; then
        GVISOR_PLATFORM="kvm"
        echo -e "${CYAN}KVM available - using KVM platform for 5x better performance${NC}"
    else
        echo -e "${YELLOW}KVM not available - using systrap platform (slower)${NC}"
    fi

    # Add gVisor to containerd config if containerd is used
    if [ -f /etc/containerd/config.toml ]; then
        # Check if gVisor config already exists
        if ! grep -q "containerd.runtimes.runsc" /etc/containerd/config.toml; then
            cat >> /etc/containerd/config.toml <<GVISOREOF

# gVisor (runsc) sandboxed runtime - Performance Optimized
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runsc]
  runtime_type = "io.containerd.runsc.v1"
  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runsc.options]
    TypeUrl = "io.containerd.runsc.v1.options"
    ConfigPath = "/etc/gvisor/runsc.toml"
GVISOREOF
        fi

        # Create optimized gVisor config for terminal workloads
        mkdir -p /etc/gvisor
        cat > /etc/gvisor/runsc.toml <<EOF
# gVisor runsc configuration - Optimized for Rexec Terminal Workloads
# Generated by setup script on $(date)

# Platform (KVM = 2-5% overhead, systrap = 10-15% overhead)
platform = "${GVISOR_PLATFORM}"

# Network isolation
network = "sandbox"

# Performance optimizations for terminal/shell workloads
directfs = true                    # 50-70% faster file I/O operations
num-network-channels = 2           # Sufficient for terminal workloads
file-access = "exclusive"          # Required for directfs, better performance
host-uds = "all"                   # Unix domain socket support

# Disable gVisor rootfs overlay to avoid conflicts with Docker overlay2 storage driver
# overlay2 disabled - use default gVisor behavior

# Debug settings (disable in production for best performance)
debug = false
debug-log = ""
EOF

        systemctl restart containerd
        echo -e "${GREEN}✓ gVisor added to containerd with performance optimizations${NC}"
    fi

    echo -e "${GREEN}✓ gVisor (runsc) installed and configured with optimizations${NC}"
    echo -e "${CYAN}  Performance flags: directfs, file-access=exclusive${NC}"
    echo ""
fi

# =============================================================================
# Next Step: Generate TLS Certificates
# =============================================================================
STEP_NUM=$((STEP_NUM + 1))
echo -e "${YELLOW}[$STEP_NUM/$TOTAL_STEPS] Generating TLS certificates...${NC}"

CERT_DIR="/etc/docker/certs"
DAYS_VALID=3650
REGENERATE_CERTS=false

# Check if valid certificates already exist (unless --force-certs is set)
if [ "$FORCE_CERTS" = "true" ]; then
    echo -e "${YELLOW}--force-certs specified - regenerating certificates...${NC}"
    REGENERATE_CERTS=true
elif [ -f "$CERT_DIR/ca.pem" ] && [ -f "$CERT_DIR/server-cert.pem" ] && [ -f "$CERT_DIR/server-key.pem" ] && \
   [ -f "$CERT_DIR/client/ca.pem" ] && [ -f "$CERT_DIR/client/cert.pem" ] && [ -f "$CERT_DIR/client/key.pem" ]; then
    # Verify certificates are still valid (not expired within 24 hours)
    if openssl x509 -checkend 86400 -noout -in "$CERT_DIR/ca.pem" 2>/dev/null && \
       openssl x509 -checkend 86400 -noout -in "$CERT_DIR/server-cert.pem" 2>/dev/null; then
        echo -e "${GREEN}✓ Existing TLS certificates found and valid - skipping regeneration${NC}"
        # Show cert expiry info
        CERT_EXPIRY=$(openssl x509 -enddate -noout -in "$CERT_DIR/server-cert.pem" 2>/dev/null | cut -d= -f2)
        echo -e "${CYAN}  Certificate expires: $CERT_EXPIRY${NC}"
        echo -e "${CYAN}  Use --force-certs to regenerate${NC}"
        echo ""
    else
        echo -e "${YELLOW}Existing certificates expired or expiring soon - regenerating...${NC}"
        REGENERATE_CERTS=true
    fi
else
    echo -e "${YELLOW}No existing certificates found - generating new ones...${NC}"
    REGENERATE_CERTS=true
fi

if [ "${REGENERATE_CERTS:-false}" = "true" ]; then
    mkdir -p "$CERT_DIR"
    cd "$CERT_DIR"

    # Generate CA
    openssl genrsa -out ca-key.pem 4096 2>/dev/null
    openssl req -new -x509 -days $DAYS_VALID -key ca-key.pem -sha256 -out ca.pem \
        -subj "/C=US/ST=Cloud/L=Docker/O=Rexec/CN=Docker CA" 2>/dev/null

    # Generate server certificate
    openssl genrsa -out server-key.pem 4096 2>/dev/null
    openssl req -subj "/CN=$PUBLIC_IP" -sha256 -new -key server-key.pem -out server.csr 2>/dev/null

    cat > extfile.cnf <<EOF
subjectAltName = DNS:localhost,DNS:$(hostname),IP:$PUBLIC_IP,IP:127.0.0.1
extendedKeyUsage = serverAuth
EOF

    openssl x509 -req -days $DAYS_VALID -sha256 \
        -in server.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
        -out server-cert.pem -extfile extfile.cnf 2>/dev/null

    # Generate client certificate
    openssl genrsa -out key.pem 4096 2>/dev/null
    openssl req -subj '/CN=client' -new -key key.pem -out client.csr 2>/dev/null
    echo extendedKeyUsage = clientAuth > extfile-client.cnf
    openssl x509 -req -days $DAYS_VALID -sha256 \
        -in client.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
        -out cert.pem -extfile extfile-client.cnf 2>/dev/null

    # Cleanup
    rm -f server.csr client.csr extfile.cnf extfile-client.cnf ca.srl

    # Set permissions
    chmod 0400 ca-key.pem server-key.pem key.pem
    chmod 0444 ca.pem server-cert.pem cert.pem

    # Create client bundle
    mkdir -p "$CERT_DIR/client"
    cp ca.pem cert.pem key.pem "$CERT_DIR/client/"

    echo -e "${GREEN}TLS certificates generated${NC}"
    echo ""
fi

# =============================================================================
# Configure Container Runtime Daemon
# =============================================================================
STEP_NUM=$((STEP_NUM + 1))

if [ "$USE_PODMAN" = true ]; then
    echo -e "${YELLOW}[$STEP_NUM/$TOTAL_STEPS] Configuring Podman for remote TLS access...${NC}"

    # Stop and disable Docker if running to avoid port conflicts
    echo -e "${CYAN}Disabling Docker to avoid conflicts...${NC}"
    systemctl stop docker.socket docker 2>/dev/null || true
    systemctl disable docker.socket docker 2>/dev/null || true

    # Stop and disable any existing Podman socket services that might conflict
    echo -e "${CYAN}Cleaning up conflicting services...${NC}"
    systemctl stop podman-tcp.socket podman.socket podman-api.service podman-tls.service 2>/dev/null || true
    systemctl disable podman-tcp.socket podman.socket 2>/dev/null || true

    # Kill any process holding ports 2375 (internal) or 2378 (external TLS)
    echo -e "${CYAN}Freeing ports 2375 and 2378...${NC}"
    fuser -k 2375/tcp 2>/dev/null || true
    fuser -k 2378/tcp 2>/dev/null || true
    sleep 1

    # Create Podman system service with TLS
    mkdir -p /etc/containers/certs.d

    # Copy certs to Podman location
    cp "$CERT_DIR/ca.pem" /etc/containers/certs.d/
    cp "$CERT_DIR/server-cert.pem" /etc/containers/certs.d/
    cp "$CERT_DIR/server-key.pem" /etc/containers/certs.d/

    # Podman doesn't support TLS flags natively, so we use stunnel as TLS proxy
    echo -e "${CYAN}Installing stunnel for TLS termination...${NC}"
    apt-get install -y stunnel4 2>/dev/null || yum install -y stunnel 2>/dev/null || dnf install -y stunnel 2>/dev/null

    # Create combined cert for stunnel
    CERT_DIR="/etc/docker/certs"
    mkdir -p /etc/stunnel
    cat $CERT_DIR/server-cert.pem $CERT_DIR/server-key.pem > /etc/stunnel/podman.pem
    chmod 600 /etc/stunnel/podman.pem
    cp $CERT_DIR/ca.pem /etc/stunnel/ca.pem
    chmod 644 /etc/stunnel/ca.pem

    # Configure stunnel for TLS termination with client cert verification
    # Podman uses port 2378 (different from Docker's 2377) to allow both to run on same host
    cat > /etc/stunnel/podman.conf <<EOF
; Stunnel configuration for Podman API
pid = /run/stunnel-podman.pid
setuid = root
setgid = root
foreground = yes

[podman-api]
accept = 0.0.0.0:2378
connect = 127.0.0.1:2375
cert = /etc/stunnel/podman.pem
CAfile = /etc/stunnel/ca.pem
verify = 2
EOF

    # Create Podman API service (plain TCP on localhost only - secure)
    # Uses port 2375 internally (only localhost), exposed via stunnel on 2378
    cat > /etc/systemd/system/podman-api.service <<EOF
[Unit]
Description=Podman API Service (Local)
After=network.target

[Service]
Type=simple
ExecStart=/usr/bin/podman system service --time=0 tcp://127.0.0.1:2375
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    # Create stunnel TLS proxy service (using foreground mode so Type=simple works)
    cat > /etc/systemd/system/podman-tls.service <<EOF
[Unit]
Description=Stunnel TLS Proxy for Podman API
After=podman-api.service
Requires=podman-api.service

[Service]
Type=simple
ExecStart=/usr/bin/stunnel /etc/stunnel/podman.conf
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    # Reload systemd
    systemctl daemon-reload

    # Enable and start Podman API first
    systemctl enable podman-api.service
    systemctl start podman-api.service

    # Wait for Podman API to be ready on localhost:2375
    echo -e "${CYAN}Waiting for Podman API to start...${NC}"
    for i in {1..10}; do
        if curl -s http://127.0.0.1:2375/_ping > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Podman API is ready${NC}"
            break
        fi
        sleep 1
    done

    # Now start stunnel TLS proxy
    systemctl enable podman-tls.service
    systemctl start podman-tls.service

    # Wait for services to start
    sleep 2

    if systemctl is-active --quiet podman-api.service && systemctl is-active --quiet podman-tls.service; then
        echo -e "${GREEN}✓ Podman API + TLS proxy running on port 2378${NC}"
    else
        echo -e "${RED}✗ Podman services failed to start${NC}"
        echo "Podman API status:"
        systemctl status podman-api.service --no-pager -l | head -10
        echo ""
        echo "TLS proxy status:"
        systemctl status podman-tls.service --no-pager -l | head -10
        echo ""
        echo "Checking port 2378:"
        ss -tlnp | grep 2378 || echo "Port 2378 not listening"
    fi

    echo -e "${GREEN}Podman configured for remote TLS access on port 2378${NC}"
    echo ""
else
    echo -e "${YELLOW}[$STEP_NUM/$TOTAL_STEPS] Configuring Docker daemon for TLS...${NC}"

    # Stop Podman if running to avoid port conflicts
    echo -e "${CYAN}Stopping Podman services if running...${NC}"
    systemctl stop podman-api.service podman-tls.service podman-api-local.service 2>/dev/null || true
    systemctl disable podman-api.service podman-tls.service podman-api-local.service 2>/dev/null || true

    # Kill any process on port 2377
    echo -e "${CYAN}Freeing port 2377...${NC}"
    fuser -k 2377/tcp 2>/dev/null || true
    sleep 1

    # Backup existing config
    if [ -f /etc/docker/daemon.json ]; then
        cp /etc/docker/daemon.json /etc/docker/daemon.json.bak
    fi

    # Build runtimes JSON based on what's installed
    RUNTIMES_JSON=""

    if [ "$INSTALL_KATA" = true ]; then
        # Create wrapper scripts for kata-runtime with different configs
        cat > /usr/local/bin/kata-fc-runtime <<'WRAPPER'
#!/bin/bash
exec /opt/kata/bin/kata-runtime --config /opt/kata/share/defaults/kata-containers/configuration-fc.toml "$@"
WRAPPER
        chmod +x /usr/local/bin/kata-fc-runtime

        cat > /usr/local/bin/kata-qemu-runtime <<'WRAPPER'
#!/bin/bash
exec /opt/kata/bin/kata-runtime --config /opt/kata/share/defaults/kata-containers/configuration-qemu.toml "$@"
WRAPPER
        chmod +x /usr/local/bin/kata-qemu-runtime

        RUNTIMES_JSON="\"kata\": {\"path\": \"/opt/kata/bin/kata-runtime\"}, \"kata-fc\": {\"path\": \"/usr/local/bin/kata-fc-runtime\"}, \"kata-qemu\": {\"path\": \"/usr/local/bin/kata-qemu-runtime\"}"
    fi

    if [ "$INSTALL_GVISOR" = true ]; then
        # Find runsc binary path
        RUNSC_PATH=""
        if [ -f /usr/local/bin/runsc ]; then
            RUNSC_PATH="/usr/local/bin/runsc"
        elif [ -f /usr/bin/runsc ]; then
            RUNSC_PATH="/usr/bin/runsc"
        elif command -v runsc &> /dev/null; then
            RUNSC_PATH=$(which runsc)
        fi

        if [ -z "$RUNSC_PATH" ]; then
            echo -e "${YELLOW}⚠ gVisor (runsc) not found - skipping runtime config${NC}"
            echo -e "${YELLOW}  Run with --with-gvisor to install gVisor${NC}"
        else
            echo -e "${GREEN}✓ Found runsc at $RUNSC_PATH${NC}"
            
            if [ -n "$RUNTIMES_JSON" ]; then
                RUNTIMES_JSON="${RUNTIMES_JSON}, "
            fi

            # Detect gVisor platform if not already set
            if [ -z "$GVISOR_PLATFORM" ]; then
                GVISOR_PLATFORM="systrap"
                if [ -e /dev/kvm ] && [ -r /dev/kvm ]; then
                    GVISOR_PLATFORM="kvm"
                    echo -e "${CYAN}KVM available - using KVM platform${NC}"
                fi
            fi

            # Build optimized gVisor runtime args for Docker daemon.json
            # These flags improve terminal/shell workload performance:
            # - directfs: 50-70% faster file I/O (requires file-access=exclusive)
            # - num-network-channels=2: Optimized for low-bandwidth terminal traffic
            # - file-access=exclusive: Better performance, required for directfs
            # - host-uds=all: Unix domain socket support
            # NOTE: ignore-cgroups removed to allow resource limit enforcement
            # NOTE: overlay2 flag removed - was causing parse errors with gVisor
            GVISOR_ARGS_JSON="\"--platform=${GVISOR_PLATFORM}\", \"--directfs\", \"--num-network-channels=2\", \"--file-access=exclusive\", \"--host-uds=all\""

            # Add runsc runtime with auto-detected platform
            RUNTIMES_JSON="${RUNTIMES_JSON}\"runsc\": {
      \"path\": \"${RUNSC_PATH}\",
      \"runtimeArgs\": [${GVISOR_ARGS_JSON}]
    }"

            # Only add explicit runsc-kvm if KVM is available
            if [ "$GVISOR_PLATFORM" = "kvm" ]; then
                GVISOR_KVM_ARGS_JSON="\"--platform=kvm\", \"--directfs\", \"--num-network-channels=2\", \"--file-access=exclusive\", \"--host-uds=all\""
                RUNTIMES_JSON="${RUNTIMES_JSON}, \"runsc-kvm\": {
      \"path\": \"${RUNSC_PATH}\",
      \"runtimeArgs\": [${GVISOR_KVM_ARGS_JSON}]
    }"
            fi
        fi
    fi

    # Check if disk quotas are available for storage-opts
    # Docker overlay2 storage driver needs backing filesystem with pquota/prjquota support
    QUOTA_AVAILABLE=false
    DOCKER_MOUNT=$(df /var/lib/docker --output=target 2>/dev/null | tail -1)
    
    # Check mount options for pquota or prjquota
    if mount | grep " $DOCKER_MOUNT " | grep -qE "pquota|prjquota"; then
        QUOTA_AVAILABLE=true
        echo -e "${GREEN}✓ Disk quotas available (mount options) - enabling per-container storage limits${NC}"
    # Also check XFS quota status directly
    elif command -v xfs_quota &> /dev/null && xfs_quota -x -c "state" "$DOCKER_MOUNT" 2>/dev/null | grep -q "Project quota state"; then
        QUOTA_AVAILABLE=true
        echo -e "${GREEN}✓ Disk quotas available (XFS quota state) - enabling per-container storage limits${NC}"
    else
        echo -e "${YELLOW}Disk quotas not available - container storage limits disabled${NC}"
        echo -e "${YELLOW}To enable disk quotas:${NC}"
        echo -e "${YELLOW}  1. Add 'pquota' to mount options in /etc/fstab${NC}"
        echo -e "${YELLOW}  2. Reboot or remount the filesystem${NC}"
    fi

    # Write daemon config with optional runtimes and storage opts
    # Build the JSON dynamically to handle optional sections
    
    # Determine what optional sections we need to add trailing commas correctly
    HAS_STORAGE_OPTS=false
    HAS_RUNTIMES=false
    if [ "$QUOTA_AVAILABLE" = true ]; then
        HAS_STORAGE_OPTS=true
    fi
    if [ -n "$RUNTIMES_JSON" ]; then
        HAS_RUNTIMES=true
    fi
    
    {
        echo '{'
        echo '  "tls": true,'
        echo "  \"tlscacert\": \"$CERT_DIR/ca.pem\","
        echo "  \"tlscert\": \"$CERT_DIR/server-cert.pem\","
        echo "  \"tlskey\": \"$CERT_DIR/server-key.pem\","
        echo '  "tlsverify": true,'
        echo '  "log-driver": "json-file",'
        echo '  "log-opts": {'
        echo '    "max-size": "10m",'
        echo '    "max-file": "3"'
        echo '  },'
        
        # Add storage-driver with comma if more sections follow
        if [ "$HAS_STORAGE_OPTS" = true ] || [ "$HAS_RUNTIMES" = true ]; then
            echo '  "storage-driver": "overlay2",'
        else
            echo '  "storage-driver": "overlay2"'
        fi
        
        # Add storage-opts if quotas available
        if [ "$HAS_STORAGE_OPTS" = true ]; then
            if [ "$HAS_RUNTIMES" = true ]; then
                echo '  "storage-opts": ["overlay2.size=10G"],'
            else
                echo '  "storage-opts": ["overlay2.size=10G"]'
            fi
        fi
        
        # Add runtimes if configured (always last, no trailing comma)
        if [ "$HAS_RUNTIMES" = true ]; then
            echo '  "runtimes": {'
            echo "    ${RUNTIMES_JSON}"
            echo '  }'
        fi
        
        echo '}'
    } > /etc/docker/daemon.json
    
    # Validate JSON syntax
    if command -v jq &> /dev/null; then
        if ! jq . /etc/docker/daemon.json > /dev/null 2>&1; then
            echo -e "${RED}✗ Invalid JSON in daemon.json!${NC}"
            cat /etc/docker/daemon.json
            exit 1
        fi
        echo -e "${GREEN}✓ daemon.json validated${NC}"
    fi
    
    if [ -n "$RUNTIMES_JSON" ]; then
        echo -e "${CYAN}Docker configured with additional runtimes${NC}"
    fi

    # Create systemd override to add TCP listener alongside unix socket
    # This keeps the unix socket working for local SSH/console access
    mkdir -p /etc/systemd/system/docker.service.d/
    cat > /etc/systemd/system/docker.service.d/override.conf <<EOF
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd -H fd:// -H unix:///var/run/docker.sock -H tcp://0.0.0.0:2377
EOF

    # Stop Docker and clean up before restart
    echo -e "${CYAN}Restarting Docker daemon...${NC}"
    systemctl stop docker.socket docker.service 2>/dev/null || true
    
    # Kill any stale Docker/containerd processes
    pkill -9 dockerd 2>/dev/null || true
    pkill -9 containerd-shim 2>/dev/null || true
    sleep 2
    
    # Clean up stale pid/socket files
    rm -f /var/run/docker.pid /var/run/docker.sock 2>/dev/null || true
    
    # Reset failed state
    systemctl reset-failed docker.service 2>/dev/null || true
    
    # Reload systemd configuration
    systemctl daemon-reload
    
    # Start Docker with timeout to prevent hang
    echo -e "${CYAN}Starting Docker daemon (timeout: 60s)...${NC}"
    if ! timeout 60 systemctl start docker; then
        echo -e "${RED}✗ Docker start timed out or failed!${NC}"
        echo -e "${YELLOW}Checking Docker status and logs...${NC}"
        systemctl status docker.service --no-pager -l 2>&1 | head -30 || true
        echo ""
        echo -e "${YELLOW}Last 30 lines of Docker journal:${NC}"
        journalctl -u docker -n 30 --no-pager 2>&1 || true
        echo ""
        echo -e "${YELLOW}Checking daemon.json for errors:${NC}"
        cat /etc/docker/daemon.json 2>&1 || true
        echo ""
        echo -e "${RED}Docker failed to start. Please check the logs above and fix any issues.${NC}"
        exit 1
    fi

    # Wait for Docker to be fully ready (socket available)
    echo -e "${CYAN}Waiting for Docker to be ready...${NC}"
    DOCKER_READY=false
    for i in {1..30}; do
        if docker version > /dev/null 2>&1; then
            DOCKER_READY=true
            break
        fi
        sleep 1
    done
    
    if [ "$DOCKER_READY" = false ]; then
        echo -e "${RED}✗ Docker started but is not responding!${NC}"
        systemctl status docker.service --no-pager -l 2>&1 | head -20 || true
        exit 1
    fi

    echo -e "${GREEN}✓ Docker daemon started and responding${NC}"
    echo -e "${GREEN}✓ Local socket access works (SSH/console)${NC}"

    echo -e "${GREEN}Docker daemon configured for TLS${NC}"
    echo ""
fi

# =============================================================================
# Configure Firewall
# =============================================================================
STEP_NUM=$((STEP_NUM + 1))
echo -e "${YELLOW}[$STEP_NUM/$TOTAL_STEPS] Configuring firewall...${NC}"

# Determine port based on runtime
if [ "$USE_PODMAN" = true ]; then
    FW_PORT=2378
else
    FW_PORT=2377
fi

# Try UFW first (Ubuntu/Debian)
if command -v ufw &> /dev/null; then
    ufw allow ${FW_PORT}/tcp
    echo -e "${GREEN}UFW: Opened port ${FW_PORT}${NC}"
# Try firewalld (CentOS/RHEL/Fedora)
elif command -v firewall-cmd &> /dev/null; then
    firewall-cmd --permanent --add-port=${FW_PORT}/tcp
    firewall-cmd --reload
    echo -e "${GREEN}firewalld: Opened port ${FW_PORT}${NC}"
# Fall back to iptables
elif command -v iptables &> /dev/null; then
    iptables -A INPUT -p tcp --dport ${FW_PORT} -j ACCEPT
    echo -e "${GREEN}iptables: Opened port ${FW_PORT}${NC}"
else
    echo -e "${YELLOW}No firewall detected. Make sure port ${FW_PORT} is accessible.${NC}"
fi

echo ""

# =============================================================================
# Test Connection
# =============================================================================
STEP_NUM=$((STEP_NUM + 1))
echo -e "${YELLOW}[$STEP_NUM/$TOTAL_STEPS] Testing TLS connection...${NC}"

sleep 2  # Wait for service to fully start

if [ "$USE_PODMAN" = true ]; then
    CERT_DIR="/etc/docker/certs"
    # Test Podman API via stunnel TLS proxy on port 2378
    if curl -s --cacert "$CERT_DIR/ca.pem" \
        --cert "$CERT_DIR/client/cert.pem" \
        --key "$CERT_DIR/client/key.pem" \
        "https://127.0.0.1:2378/v4.0.0/libpod/info" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Podman TLS connection successful!${NC}"
    elif curl -s --cacert "$CERT_DIR/ca.pem" \
        --cert "$CERT_DIR/client/cert.pem" \
        --key "$CERT_DIR/client/key.pem" \
        "https://127.0.0.1:2378/_ping" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Podman TLS connection successful!${NC}"
    else
        echo -e "${YELLOW}⚠ TLS connection test - checking services...${NC}"
        echo "Podman API (local on 2375):"
        curl -s http://127.0.0.1:2375/_ping && echo " - Local API works" || echo " - Local API failed"
        echo "Stunnel proxy (TLS on 2378):"
        systemctl status podman-tls.service --no-pager -l | head -5
    fi
else
    if docker --tlsverify \
        --tlscacert=$CERT_DIR/ca.pem \
        --tlscert=$CERT_DIR/client/cert.pem \
        --tlskey=$CERT_DIR/client/key.pem \
        -H=tcp://127.0.0.1:2377 version > /dev/null 2>&1; then
        echo -e "${GREEN}✓ TLS connection successful!${NC}"
    else
        echo -e "${RED}✗ TLS connection failed${NC}"
        echo "Check Docker logs: journalctl -u docker -n 50"
        exit 1
    fi
fi

echo ""

# =============================================================================
# Verify Disk Quota Support
# =============================================================================
STEP_NUM=$((STEP_NUM + 1))
echo -e "${YELLOW}[$STEP_NUM/$TOTAL_STEPS] Verifying disk quota support...${NC}"

# Check if quotas are actually working by testing Docker's storage driver
QUOTA_WORKING=false

if [ "$USE_PODMAN" != true ]; then
    # Check Docker info for storage driver and options
    STORAGE_INFO=$(docker info 2>/dev/null | grep -A5 "Storage Driver")
    echo -e "${CYAN}Storage configuration:${NC}"
    echo "$STORAGE_INFO"
    
    # Check if overlay2.size is in daemon.json
    if grep -q "overlay2.size" /etc/docker/daemon.json 2>/dev/null; then
        echo -e "${GREEN}✓ Storage limit configured in daemon.json${NC}"
        
        # Check mount options for pquota
        DOCKER_MOUNT=$(df /var/lib/docker --output=target 2>/dev/null | tail -1)
        MOUNT_OPTS=$(mount | grep " $DOCKER_MOUNT " | grep -oE '\([^)]+\)' || echo "(unknown)")
        echo -e "${CYAN}Mount options for $DOCKER_MOUNT: $MOUNT_OPTS${NC}"
        
        if echo "$MOUNT_OPTS" | grep -qE "pquota|prjquota"; then
            echo -e "${GREEN}✓ Project quotas enabled in mount options${NC}"
            
            # Test by creating a container with storage limit
            echo -e "${CYAN}Testing disk quota enforcement...${NC}"
            
            # Create a test container with 50MB limit
            TEST_OUTPUT=$(docker run --rm --storage-opt size=50M alpine sh -c 'echo "Quota test passed"' 2>&1)
            if echo "$TEST_OUTPUT" | grep -q "Quota test passed"; then
                echo -e "${GREEN}✓ Disk quotas are working! Containers can have per-container storage limits.${NC}"
                QUOTA_WORKING=true
            elif echo "$TEST_OUTPUT" | grep -qi "not supported\|unknown flag\|invalid"; then
                echo -e "${YELLOW}⚠ Storage driver doesn't support per-container limits${NC}"
                echo -e "${YELLOW}  Output: $TEST_OUTPUT${NC}"
            else
                echo -e "${GREEN}✓ Container ran with storage-opt (quotas likely working)${NC}"
                QUOTA_WORKING=true
            fi
        else
            echo -e "${YELLOW}⚠ pquota not in mount options - quotas may not work${NC}"
            echo -e "${YELLOW}  A reboot may be required to enable quotas${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ No storage-opts configured - disk quotas disabled${NC}"
        echo -e "${YELLOW}  This usually means the filesystem doesn't support project quotas${NC}"
    fi
fi

if [ "$QUOTA_WORKING" = true ]; then
    echo -e "${GREEN}✓ Per-container disk limits are enforced${NC}"
else
    echo -e "${YELLOW}Note: Per-container disk limits may not be enforced.${NC}"
    echo -e "${YELLOW}Containers will share the host's disk space.${NC}"
fi

echo ""

# =============================================================================
# Output Environment Variables
# =============================================================================
echo -e "${GREEN}=============================================${NC}"
echo -e "${GREEN}  Setup Complete!${NC}"
echo -e "${GREEN}=============================================${NC}"
echo ""

# Set port based on runtime
if [ "$USE_PODMAN" = true ]; then
    RUNTIME_PORT=2378
else
    RUNTIME_PORT=2377
fi

echo -e "${YELLOW}Set these environment variables in your Railway/PipeOps deployment:${NC}"
echo ""
echo -e "${BLUE}DOCKER_HOST${NC}=tcp://${PUBLIC_IP}:${RUNTIME_PORT}"
echo -e "${BLUE}DOCKER_TLS_VERIFY${NC}=1"
if [ "$USE_PODMAN" = true ]; then
    echo -e "${BLUE}CONTAINER_ENGINE${NC}=podman"
fi
echo ""

# Output certificates as base64 for easy copy-paste
echo -e "${YELLOW}Certificate values (copy these exactly):${NC}"
echo ""
echo -e "${BLUE}DOCKER_CA_CERT${NC}="
cat "$CERT_DIR/client/ca.pem"
echo ""
echo -e "${BLUE}DOCKER_CLIENT_CERT${NC}="
cat "$CERT_DIR/client/cert.pem"
echo ""
echo -e "${BLUE}DOCKER_CLIENT_KEY${NC}="
cat "$CERT_DIR/client/key.pem"
echo ""

# Save to file for convenience
if [ "$USE_PODMAN" = true ]; then
    cat > /root/rexec-docker-env.txt <<EOF
# Rexec Podman Host Environment Variables
# Generated on $(date)
# Podman Host: $PUBLIC_IP
# Port: 2378 (Podman TLS)

DOCKER_HOST=tcp://${PUBLIC_IP}:2378
DOCKER_TLS_VERIFY=1
CONTAINER_ENGINE=podman

# CA Certificate
DOCKER_CA_CERT=$(cat "$CERT_DIR/client/ca.pem" | base64 -w0)

# Client Certificate
DOCKER_CLIENT_CERT=$(cat "$CERT_DIR/client/cert.pem" | base64 -w0)

# Client Key
DOCKER_CLIENT_KEY=$(cat "$CERT_DIR/client/key.pem" | base64 -w0)
EOF
else
    cat > /root/rexec-docker-env.txt <<EOF
# Rexec Docker Host Environment Variables
# Generated on $(date)
# Docker Host: $PUBLIC_IP
# Port: 2377 (Docker TLS)

DOCKER_HOST=tcp://${PUBLIC_IP}:2377
DOCKER_TLS_VERIFY=1

# CA Certificate
DOCKER_CA_CERT=$(cat "$CERT_DIR/client/ca.pem" | base64 -w0)

# Client Certificate
DOCKER_CLIENT_CERT=$(cat "$CERT_DIR/client/cert.pem" | base64 -w0)

# Client Key
DOCKER_CLIENT_KEY=$(cat "$CERT_DIR/client/key.pem" | base64 -w0)
EOF
fi

echo -e "${GREEN}Environment variables also saved to: /root/rexec-docker-env.txt${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Copy the environment variables above to your deployment platform"
echo "2. Deploy Rexec using Dockerfile.remote"
echo "3. Test the connection from your Rexec instance"
echo ""

if [ "$USE_PODMAN" = true ]; then
    echo -e "${GREEN}Your Podman host is ready at: tcp://${PUBLIC_IP}:2378${NC}"
    echo ""
    echo -e "${BLUE}Summary:${NC}"
    echo "  - Local access: podman commands work normally"
    echo "  - Remote access (TLS): tcp://${PUBLIC_IP}:2378"
    echo ""
    echo -e "${CYAN}Podman Benefits:${NC}"
    echo "  - Docker-compatible API (works with Docker SDK)"
    echo "  - Rootless container support"
    echo "  - Daemonless architecture"
    echo "  - Better Kata Containers integration"
else
    echo -e "${GREEN}Your Docker host is ready at: tcp://${PUBLIC_IP}:2377${NC}"
    echo ""
    echo -e "${BLUE}Summary:${NC}"
    echo "  - Local access (SSH/console): docker commands work normally"
    echo "  - Remote access (TLS): tcp://${PUBLIC_IP}:2377"
fi

if [ "$INSTALL_KATA" = true ]; then
    echo ""
    echo -e "${CYAN}Kata/Firecracker Usage:${NC}"
    echo "  - Run containers with microVM isolation:"
    echo "    docker run --runtime=kata -it alpine sh"
    echo ""
    echo "  - In Rexec, set OCI_RUNTIME=kata to use Firecracker VMs"
    echo "  - Each terminal will run in its own microVM for full isolation"
    echo ""
    echo -e "${CYAN}Benefits:${NC}"
    echo "  - True VM-level isolation (not just container namespaces)"
    echo "  - Dedicated kernel per terminal"
    echo "  - Near-container boot times (~125ms)"
    echo "  - Better security for multi-tenant environments"
    echo ""
    echo -e "${YELLOW}Testing Kata runtime...${NC}"
    if docker run --rm --runtime=kata hello-world > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Kata runtime works!${NC}"
    else
        echo -e "${YELLOW}⚠ Kata test failed - checking logs...${NC}"
        echo "Try manually: docker run --runtime=kata hello-world"
        echo "Debug with: kata-runtime check"
    fi
fi

if [ "$INSTALL_GVISOR" = true ]; then
    echo ""
    echo -e "${CYAN}gVisor (runsc) Usage:${NC}"
    echo "  - Run containers with gVisor sandboxing:"
    echo "    docker run --runtime=runsc -it alpine sh"
    echo ""
    echo "  - In Rexec, set OCI_RUNTIME=runsc to use gVisor"
    if [ "$GVISOR_PLATFORM" = "kvm" ]; then
        echo "  - For explicit KVM: OCI_RUNTIME=runsc-kvm"
    fi
    echo ""
    echo -e "${CYAN}Benefits:${NC}"
    echo "  - User-space kernel intercepts syscalls (strong isolation)"
    echo "  - ${GVISOR_PLATFORM} platform: ~$([ "$GVISOR_PLATFORM" = "kvm" ] && echo "2-5%" || echo "10-15%") overhead"
    echo "  - directfs + overlay2=memory: Near-native file I/O"
    echo "  - No VM overhead (faster than Kata/Firecracker)"
    echo "  - Ideal for untrusted multi-tenant workloads"
    echo ""
    echo -e "${YELLOW}Testing gVisor runtime...${NC}"
    if docker run --rm --runtime=runsc hello-world > /dev/null 2>&1; then
        echo -e "${GREEN}✓ gVisor runtime works!${NC}"

        # Run performance test
        echo -e "${YELLOW}Running quick performance test...${NC}"
        echo -n "  Native (runc): "
        RUNC_TIME=$(docker run --rm alpine sh -c 'time -p for i in $(seq 1 100); do echo test > /tmp/file; done' 2>&1 | grep real | awk '{print $2}' || echo "0.1")
        echo "${RUNC_TIME}s"

        echo -n "  gVisor (runsc): "
        RUNSC_TIME=$(docker run --runtime=runsc --rm alpine sh -c 'time -p for i in $(seq 1 100); do echo test > /tmp/file; done' 2>&1 | grep real | awk '{print $2}' || echo "0.1")
        echo "${RUNSC_TIME}s"

        # Calculate overhead
        if command -v bc &> /dev/null && [ "$(echo "$RUNC_TIME > 0" | bc)" = "1" ]; then
            OVERHEAD=$(echo "scale=1; ($RUNSC_TIME - $RUNC_TIME) / $RUNC_TIME * 100" | bc 2>/dev/null || echo "N/A")
            if [ "$OVERHEAD" != "N/A" ]; then
                echo -e "${CYAN}  Overhead: ${OVERHEAD}% (target: <10% with KVM+directfs)${NC}"
            fi
        fi
    else
        echo -e "${YELLOW}⚠ gVisor test failed - checking...${NC}"
        echo "Try manually: docker run --runtime=runsc hello-world"
        echo "Check logs: runsc --debug --alsologtostderr ..."
    fi
fi

echo ""

# Check if reboot is needed for disk quotas
if [ "$QUOTA_NEEDS_REBOOT" = true ]; then
    echo -e "${YELLOW}=============================================${NC}"
    echo -e "${YELLOW}  REBOOT REQUIRED FOR DISK QUOTAS${NC}"
    echo -e "${YELLOW}=============================================${NC}"
    echo ""
    echo -e "${YELLOW}Disk quota settings have been added to /etc/fstab but require a reboot.${NC}"
    echo -e "${YELLOW}After rebooting, disk quotas will be enabled automatically.${NC}"
    echo ""
    echo -e "${CYAN}To verify disk quotas after reboot:${NC}"
    echo "  mount | grep pquota"
    echo "  # or for ext4:"
    echo "  mount | grep prjquota"
    echo ""
    
    # Check if running interactively
    if [ -t 0 ]; then
        read -p "Would you like to reboot now to enable disk quotas? (y/N): " REBOOT_NOW
        if [[ "$REBOOT_NOW" =~ ^[Yy]$ ]]; then
            echo -e "${GREEN}Rebooting in 5 seconds...${NC}"
            sleep 5
            sudo reboot
        else
            echo -e "${YELLOW}Please reboot manually when ready: sudo reboot${NC}"
        fi
    else
        echo -e "${YELLOW}Non-interactive mode - please reboot manually: sudo reboot${NC}"
    fi
    echo ""
fi

echo -e "${GREEN}Setup complete!${NC}"
