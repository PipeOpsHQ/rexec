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
#
# Runtime Options (set via CONTAINER_RUNTIME env var in Rexec):
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
    esac
done

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
# 1. Install Container Runtime (Docker or Podman)
# =============================================================================
STEP_NUM=1
TOTAL_STEPS=5
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
    
    # Install dependencies
    apt-get update
    apt-get install -y curl gnupg2 apt-transport-https ca-certificates
    
    # Use Kata 2.5.x which has OCI runtime CLI support for Docker
    # Kata 3.x removed OCI CLI support and only works as containerd shim
    KATA_VERSION="2.5.2"
    KATA_URL="https://github.com/kata-containers/kata-containers/releases/download/${KATA_VERSION}"
    
    # Download and extract Kata
    echo -e "${CYAN}Downloading Kata Containers ${KATA_VERSION}...${NC}"
    mkdir -p /opt/kata
    cd /opt/kata
    
    # Download kata-static bundle (includes Firecracker)
    curl -LO "${KATA_URL}/kata-static-${KATA_VERSION}-${KATA_ARCH}.tar.xz"
    tar -xf "kata-static-${KATA_VERSION}-${KATA_ARCH}.tar.xz" --strip-components=2
    
    # Clean up archive
    rm -f "kata-static-${KATA_VERSION}-${KATA_ARCH}.tar.xz"
    
    # Verify extracted files exist
    if [ ! -f /opt/kata/bin/kata-runtime ]; then
        echo -e "${RED}Kata binaries not found after extraction${NC}"
        echo "Checking extracted contents..."
        ls -la /opt/kata/
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
    # Docker/containerd looks for containerd-shim-<runtime>-v2 in PATH
    ln -sf /opt/kata/bin/containerd-shim-kata-v2 /usr/local/bin/containerd-shim-kata-fc-v2
    ln -sf /opt/kata/bin/containerd-shim-kata-v2 /usr/local/bin/containerd-shim-kata-qemu-v2
    ln -sf /opt/kata/bin/containerd-shim-kata-v2 /usr/local/bin/containerd-shim-kata-clh-v2
    
    echo -e "${GREEN}✓ Kata shim binaries linked${NC}"
    ls -la /usr/local/bin/containerd-shim-kata*
    
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
        exit 1
    fi
    
    # Check Firecracker
    if /opt/kata/bin/firecracker --version > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Firecracker available${NC}"
        /opt/kata/bin/firecracker --version
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
    
    # Configure gVisor for Docker
    # gVisor supports ptrace (default) and KVM platforms
    # KVM provides better performance but requires /dev/kvm
    GVISOR_PLATFORM="systrap"
    if [ -e /dev/kvm ] && [ -r /dev/kvm ]; then
        GVISOR_PLATFORM="kvm"
        echo -e "${CYAN}KVM available - using KVM platform for better performance${NC}"
    fi
    
    # Add gVisor to containerd config if containerd is used
    if [ -f /etc/containerd/config.toml ]; then
        # Check if gVisor config already exists
        if ! grep -q "containerd.runtimes.runsc" /etc/containerd/config.toml; then
            cat >> /etc/containerd/config.toml <<GVISOREOF

# gVisor (runsc) sandboxed runtime
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runsc]
  runtime_type = "io.containerd.runsc.v1"
  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runsc.options]
    TypeUrl = "io.containerd.runsc.v1.options"
    ConfigPath = "/etc/gvisor/runsc.toml"
GVISOREOF
        fi
        
        # Create gVisor config
        mkdir -p /etc/gvisor
        cat > /etc/gvisor/runsc.toml <<EOF
# gVisor runsc configuration
platform = "${GVISOR_PLATFORM}"
network = "sandbox"
EOF
        
        systemctl restart containerd
        echo -e "${GREEN}✓ gVisor added to containerd${NC}"
    fi
    
    echo -e "${GREEN}✓ gVisor (runsc) installed and configured${NC}"
    echo ""
fi

# =============================================================================
# Next Step: Generate TLS Certificates
# =============================================================================
STEP_NUM=$((STEP_NUM + 1))
echo -e "${YELLOW}[$STEP_NUM/$TOTAL_STEPS] Generating TLS certificates...${NC}"

CERT_DIR="/etc/docker/certs"
DAYS_VALID=3650

# Check if valid certificates already exist
if [ -f "$CERT_DIR/ca.pem" ] && [ -f "$CERT_DIR/server-cert.pem" ] && [ -f "$CERT_DIR/server-key.pem" ] && \
   [ -f "$CERT_DIR/client/ca.pem" ] && [ -f "$CERT_DIR/client/cert.pem" ] && [ -f "$CERT_DIR/client/key.pem" ]; then
    # Verify certificates are still valid (not expired)
    if openssl x509 -checkend 86400 -noout -in "$CERT_DIR/ca.pem" 2>/dev/null && \
       openssl x509 -checkend 86400 -noout -in "$CERT_DIR/server-cert.pem" 2>/dev/null; then
        echo -e "${GREEN}✓ Existing TLS certificates found and valid - reusing${NC}"
        echo ""
    else
        echo -e "${YELLOW}Existing certificates expired - regenerating...${NC}"
        REGENERATE_CERTS=true
    fi
else
    echo -e "${YELLOW}No existing certificates found - generating new ones...${NC}"
    REGENERATE_CERTS=true
fi

if [ "${REGENERATE_CERTS:-true}" = "true" ]; then
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
        if [ -n "$RUNTIMES_JSON" ]; then
            RUNTIMES_JSON="${RUNTIMES_JSON}, "
        fi
        RUNTIMES_JSON="${RUNTIMES_JSON}\"runsc\": {\"path\": \"/usr/local/bin/runsc\"}, \"runsc-kvm\": {\"path\": \"/usr/local/bin/runsc\", \"runtimeArgs\": [\"--platform=kvm\"]}"
    fi
    
    # Write daemon config with optional runtimes
    if [ -n "$RUNTIMES_JSON" ]; then
        cat > /etc/docker/daemon.json <<EOF
{
  "tls": true,
  "tlscacert": "$CERT_DIR/ca.pem",
  "tlscert": "$CERT_DIR/server-cert.pem",
  "tlskey": "$CERT_DIR/server-key.pem",
  "tlsverify": true,
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  },
  "storage-driver": "overlay2",
  "runtimes": {
    ${RUNTIMES_JSON}
  }
}
EOF
        echo -e "${CYAN}Docker configured with additional runtimes${NC}"
    else
        cat > /etc/docker/daemon.json <<EOF
{
  "tls": true,
  "tlscacert": "$CERT_DIR/ca.pem",
  "tlscert": "$CERT_DIR/server-cert.pem",
  "tlskey": "$CERT_DIR/server-key.pem",
  "tlsverify": true,
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  },
  "storage-driver": "overlay2"
}
EOF
    fi

    # Create systemd override to add TCP listener alongside unix socket
    # This keeps the unix socket working for local SSH/console access
    mkdir -p /etc/systemd/system/docker.service.d/
    cat > /etc/systemd/system/docker.service.d/override.conf <<EOF
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd -H fd:// -H unix:///var/run/docker.sock -H tcp://0.0.0.0:2377
EOF

    # Reload and restart Docker
    systemctl daemon-reload
    systemctl restart docker

    # Wait for Docker to be ready
    sleep 2

    # Verify Docker is running and local socket works
    if ! systemctl is-active --quiet docker; then
        echo -e "${RED}✗ Docker failed to start!${NC}"
        journalctl -u docker -n 20 --no-pager
        exit 1
    fi

    # Verify local socket still works (important for SSH access)
    if docker version > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Local socket access works (SSH/console)${NC}"
    else
        echo -e "${RED}✗ Local socket access failed!${NC}"
    fi

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
    echo "  - In Rexec, set CONTAINER_RUNTIME=kata to use Firecracker VMs"
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
    echo "  - In Rexec, set CONTAINER_RUNTIME=runsc to use gVisor"
    echo "  - For KVM acceleration: CONTAINER_RUNTIME=runsc-kvm"
    echo ""
    echo -e "${CYAN}Benefits:${NC}"
    echo "  - User-space kernel intercepts syscalls"
    echo "  - No VM overhead (faster than Kata/Firecracker)"
    echo "  - Strong security boundary without hardware virtualization"
    echo "  - Ideal for untrusted workloads"
    echo ""
    echo -e "${YELLOW}Testing gVisor runtime...${NC}"
    if docker run --rm --runtime=runsc hello-world > /dev/null 2>&1; then
        echo -e "${GREEN}✓ gVisor runtime works!${NC}"
    else
        echo -e "${YELLOW}⚠ gVisor test failed - checking...${NC}"
        echo "Try manually: docker run --runtime=runsc hello-world"
        echo "Check logs: runsc --debug --alsologtostderr ..."
    fi
fi

echo ""
echo -e "${GREEN}Setup complete!${NC}"
