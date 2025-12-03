#!/bin/bash
#
# Podman Remote Host Setup Script with TLS
# More secure than Docker: rootless, daemonless, no exposed socket
#
# Port: 2378 (non-standard to avoid attacks on default Docker port 2376)
#
# Usage: ./setup.sh [OPTIONS]
#   --certs-only    Only generate certificates
#   --no-certs      Skip certificate generation (reuse existing)
#   --rootless      Run Podman in rootless mode (more secure, recommended)
#   --user=NAME     Specify user for rootless mode (default: rexec)
#
# Security advantages over Docker:
#   - Rootless containers by default
#   - No always-running privileged daemon  
#   - Fork/exec model - better process isolation
#   - Native user namespace support
#   - Non-standard port 2378 (not 2376)
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() { echo -e "${GREEN}✓${NC} $1"; }
log_warn() { echo -e "${YELLOW}⚠${NC} $1"; }
log_error() { echo -e "${RED}✗${NC} $1"; }
log_step() { echo -e "${CYAN}▸${NC} $1"; }

# Configuration
CERT_DIR="/etc/podman/certs"
CLIENT_CERT_DIR="$HOME/.podman-certs"
VALIDITY_DAYS=3650
HOST_IP=$(curl -s ifconfig.me || hostname -I | awk '{print $1}')

# Parse arguments
CERTS_ONLY=false
NO_CERTS=false
ROOTLESS=false
PODMAN_USER="rexec"

for arg in "$@"; do
    case $arg in
        --certs-only) CERTS_ONLY=true ;;
        --no-certs) NO_CERTS=true ;;
        --rootless) ROOTLESS=true ;;
        --user=*) PODMAN_USER="${arg#*=}" ;;
    esac
done

echo -e "${CYAN}"
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║           Podman Remote Host Setup with TLS                 ║"
echo "║                                                              ║"
echo "║  Security advantages over Docker:                           ║"
echo "║  • Rootless containers by default                           ║"
echo "║  • No always-running privileged daemon                      ║"
echo "║  • Fork/exec model - better process isolation               ║"
echo "║  • Native user namespace support                            ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

if [ "$ROOTLESS" = true ]; then
    echo -e "${GREEN}Running in ROOTLESS mode (recommended for security)${NC}"
    echo -e "${YELLOW}User: $PODMAN_USER${NC}"
    echo ""
fi

# ============================================================================
# STEP 1: Install Podman
# ============================================================================
install_podman() {
    log_step "Installing Podman..."
    
    if command -v podman &> /dev/null; then
        log_info "Podman already installed: $(podman --version)"
    else
        # Detect OS
        if [ -f /etc/os-release ]; then
            . /etc/os-release
            OS=$ID
            VERSION=$VERSION_ID
        fi
        
        case $OS in
            ubuntu|debian)
                apt-get update -qq
                apt-get install -y -qq podman podman-docker uidmap slirp4netns fuse-overlayfs
                ;;
            fedora|centos|rhel|rocky|almalinux)
                dnf install -y podman podman-docker uidmap slirp4netns fuse-overlayfs
                ;;
            *)
                log_error "Unsupported OS: $OS"
                exit 1
                ;;
        esac
        
        log_info "Podman installed: $(podman --version)"
    fi
    
    # Setup rootless user if requested
    if [ "$ROOTLESS" = true ]; then
        setup_rootless_user
    fi
}

# ============================================================================
# STEP 1b: Setup Rootless User
# ============================================================================
setup_rootless_user() {
    log_step "Setting up rootless user: $PODMAN_USER..."
    
    # Create user if doesn't exist
    if ! id "$PODMAN_USER" &>/dev/null; then
        useradd -m -s /bin/bash "$PODMAN_USER"
        log_info "Created user: $PODMAN_USER"
    else
        log_info "User $PODMAN_USER already exists"
    fi
    
    # Configure subuid/subgid for user namespaces
    if ! grep -q "^$PODMAN_USER:" /etc/subuid; then
        echo "$PODMAN_USER:100000:65536" >> /etc/subuid
        echo "$PODMAN_USER:100000:65536" >> /etc/subgid
        log_info "Configured subuid/subgid for $PODMAN_USER"
    fi
    
    # Enable lingering so user services run without login
    loginctl enable-linger "$PODMAN_USER"
    log_info "Enabled lingering for $PODMAN_USER"
    
    # Create XDG_RUNTIME_DIR if needed
    PODMAN_USER_UID=$(id -u "$PODMAN_USER")
    RUNTIME_DIR="/run/user/$PODMAN_USER_UID"
    if [ ! -d "$RUNTIME_DIR" ]; then
        mkdir -p "$RUNTIME_DIR"
        chown "$PODMAN_USER:$PODMAN_USER" "$RUNTIME_DIR"
        chmod 700 "$RUNTIME_DIR"
    fi
    
    # Setup podman storage for rootless
    PODMAN_USER_HOME=$(eval echo ~$PODMAN_USER)
    mkdir -p "$PODMAN_USER_HOME/.config/containers"
    cat > "$PODMAN_USER_HOME/.config/containers/storage.conf" <<EOF
[storage]
driver = "overlay"
runroot = "/run/user/$PODMAN_USER_UID/containers"
graphroot = "$PODMAN_USER_HOME/.local/share/containers/storage"

[storage.options.overlay]
mount_program = "/usr/bin/fuse-overlayfs"
EOF
    chown -R "$PODMAN_USER:$PODMAN_USER" "$PODMAN_USER_HOME/.config"
    
    log_info "Rootless user $PODMAN_USER configured"
}

# ============================================================================
# STEP 2: Generate TLS Certificates
# ============================================================================
generate_certificates() {
    if [ "$NO_CERTS" = true ] && [ -f "$CERT_DIR/server-cert.pem" ]; then
        log_info "Reusing existing certificates"
        return 0
    fi
    
    log_step "Generating TLS certificates..."
    
    mkdir -p "$CERT_DIR" "$CLIENT_CERT_DIR"
    chmod 700 "$CERT_DIR" "$CLIENT_CERT_DIR"
    cd "$CERT_DIR"
    
    # Generate CA
    log_step "Creating Certificate Authority..."
    openssl genrsa -out ca-key.pem 4096 2>/dev/null
    openssl req -new -x509 -days $VALIDITY_DAYS -key ca-key.pem -sha256 -out ca.pem \
        -subj "/C=US/ST=Cloud/L=Podman/O=Rexec/CN=Podman CA" 2>/dev/null
    
    # Generate Server Certificate
    log_step "Creating server certificate..."
    openssl genrsa -out server-key.pem 4096 2>/dev/null
    openssl req -new -key server-key.pem -out server.csr \
        -subj "/CN=$HOST_IP" 2>/dev/null
    
    cat > extfile.cnf <<EOF
subjectAltName = DNS:localhost,IP:$HOST_IP,IP:127.0.0.1
extendedKeyUsage = serverAuth
EOF
    
    openssl x509 -req -days $VALIDITY_DAYS -sha256 \
        -in server.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
        -out server-cert.pem -extfile extfile.cnf 2>/dev/null
    
    # Generate Client Certificate
    log_step "Creating client certificate..."
    openssl genrsa -out client-key.pem 4096 2>/dev/null
    openssl req -new -key client-key.pem -out client.csr \
        -subj "/CN=client" 2>/dev/null
    
    cat > client-extfile.cnf <<EOF
extendedKeyUsage = clientAuth
EOF
    
    openssl x509 -req -days $VALIDITY_DAYS -sha256 \
        -in client.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
        -out client-cert.pem -extfile client-extfile.cnf 2>/dev/null
    
    # Set permissions
    chmod 644 ca.pem server-cert.pem client-cert.pem
    chmod 600 ca-key.pem server-key.pem client-key.pem
    
    # Copy client certs
    cp ca.pem client-cert.pem client-key.pem "$CLIENT_CERT_DIR/"
    
    # Cleanup
    rm -f *.csr *.cnf *.srl
    
    log_info "Certificates generated in $CERT_DIR"
}

# ============================================================================
# STEP 3: Configure Podman Service with TLS
# ============================================================================
configure_podman_service() {
    log_step "Configuring Podman API service with TLS..."
    
    if [ "$ROOTLESS" = true ]; then
        configure_rootless_service
    else
        configure_rootful_service
    fi
}

# ============================================================================
# STEP 3a: Configure Rootless Podman Service
# ============================================================================
configure_rootless_service() {
    log_step "Configuring rootless Podman service for $PODMAN_USER..."
    
    PODMAN_USER_UID=$(id -u "$PODMAN_USER")
    PODMAN_USER_HOME=$(eval echo ~$PODMAN_USER)
    
    # Create user systemd directory
    mkdir -p "$PODMAN_USER_HOME/.config/systemd/user"
    
    # Create rootless Podman API service (runs as user)
    # Uses port 2375 internally (localhost only)
    cat > "$PODMAN_USER_HOME/.config/systemd/user/podman-api.service" <<EOF
[Unit]
Description=Podman API Service (Rootless)
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/podman system service --time=0 tcp://127.0.0.1:2375
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
EOF
    
    chown -R "$PODMAN_USER:$PODMAN_USER" "$PODMAN_USER_HOME/.config/systemd"
    
    # Enable and start the user service
    sudo -u "$PODMAN_USER" XDG_RUNTIME_DIR="/run/user/$PODMAN_USER_UID" \
        systemctl --user daemon-reload
    sudo -u "$PODMAN_USER" XDG_RUNTIME_DIR="/run/user/$PODMAN_USER_UID" \
        systemctl --user enable --now podman-api.service
    
    log_info "Rootless Podman API service started"
    
    # Setup nginx TLS proxy (runs as root to bind to privileged port)
    setup_nginx_proxy
}

# ============================================================================
# STEP 3b: Configure Rootful Podman Service  
# ============================================================================
configure_rootful_service() {
    log_step "Configuring rootful Podman service..."
    
    # Create Podman local service (no TLS, only localhost on port 2375)
    cat > /etc/systemd/system/podman-api-local.service <<EOF
[Unit]
Description=Podman API Service (Local only)
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/podman system service --time=0 tcp://127.0.0.1:2375
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
    
    # Reload and start services
    systemctl daemon-reload
    systemctl enable --now podman-api-local
    
    log_info "Rootful Podman API service started on localhost:2375"
    
    # Setup nginx TLS proxy (exposes on port 2378)
    setup_nginx_proxy
}

# ============================================================================
# STEP 3c: Setup Nginx TLS Proxy
# ============================================================================
setup_nginx_proxy() {
    log_step "Setting up TLS proxy..."
    
    if ! command -v nginx &> /dev/null; then
        apt-get install -y -qq nginx || dnf install -y nginx
    fi
    
    # Create nginx config for TLS proxy
    # Proxies external port 2378 (TLS) to internal port 2375 (plain)
    mkdir -p /etc/nginx/sites-available /etc/nginx/sites-enabled
    
    cat > /etc/nginx/sites-available/podman-tls <<EOF
upstream podman {
    server 127.0.0.1:2375;
}

server {
    listen 2378 ssl;
    
    ssl_certificate $CERT_DIR/server-cert.pem;
    ssl_certificate_key $CERT_DIR/server-key.pem;
    ssl_client_certificate $CERT_DIR/ca.pem;
    ssl_verify_client on;
    
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    
    location / {
        proxy_pass http://podman;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_read_timeout 900s;
        proxy_send_timeout 900s;
        proxy_connect_timeout 60s;
        
        # For streaming/attach
        proxy_buffering off;
        chunked_transfer_encoding on;
    }
}
EOF
    
    # Enable site
    ln -sf /etc/nginx/sites-available/podman-tls /etc/nginx/sites-enabled/
    rm -f /etc/nginx/sites-enabled/default 2>/dev/null || true
    
    # Start nginx
    nginx -t && systemctl enable --now nginx
    
    log_info "TLS proxy configured on port 2378"
}

# ============================================================================
# STEP 4: Configure Firewall
# ============================================================================
configure_firewall() {
    log_step "Configuring firewall..."
    
    if command -v ufw &> /dev/null; then
        ufw allow 2378/tcp comment "Podman API TLS"
        log_info "UFW rule added for port 2378"
    elif command -v firewall-cmd &> /dev/null; then
        firewall-cmd --permanent --add-port=2378/tcp
        firewall-cmd --reload
        log_info "Firewalld rule added for port 2378"
    fi
}

# ============================================================================
# STEP 5: Test Connection
# ============================================================================
test_connection() {
    log_step "Testing TLS connection..."
    
    sleep 2
    
    # Test with curl
    if curl -s --cacert "$CERT_DIR/ca.pem" \
            --cert "$CERT_DIR/client-cert.pem" \
            --key "$CERT_DIR/client-key.pem" \
            --max-time 10 \
            "https://$HOST_IP:2378/_ping" | grep -q "OK"; then
        log_info "TLS connection successful!"
    else
        log_warn "TLS test inconclusive - service may still be starting"
    fi
}

# ============================================================================
# STEP 6: Output Client Configuration
# ============================================================================
output_client_config() {
    echo ""
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}Podman Remote Host Setup Complete!${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
    
    if [ "$ROOTLESS" = true ]; then
        echo -e "${GREEN}ROOTLESS MODE ENABLED${NC}"
        echo "Containers run as unprivileged user: $PODMAN_USER"
        echo ""
        echo "Security benefits:"
        echo "  • No root daemon running"
        echo "  • Containers isolated via user namespaces"
        echo "  • UID mapping: container root → host UID 100000+"
        echo "  • Limited blast radius if container escapes"
        echo ""
    fi
    
    echo -e "${YELLOW}Client certificates are in: $CLIENT_CERT_DIR${NC}"
    echo ""
    echo "Copy these files to your Rexec server:"
    echo "  • ca.pem"
    echo "  • client-cert.pem (rename to cert.pem)"
    echo "  • client-key.pem (rename to key.pem)"
    echo ""
    echo "Environment variables for Rexec:"
    echo "  DOCKER_HOST=tcp://$HOST_IP:2378"
    echo "  DOCKER_TLS_VERIFY=1"
    echo "  DOCKER_CERT_PATH=/path/to/certs"
    echo ""
    echo "Or use with podman-remote:"
    echo "  podman --remote --url tcp://$HOST_IP:2378 \\"
    echo "    --identity $CLIENT_CERT_DIR/client-key.pem info"
    echo ""
    echo "Test connection:"
    echo "  curl --cacert ca.pem --cert client-cert.pem --key client-key.pem \\"
    echo "    https://$HOST_IP:2378/_ping"
    echo ""
    
    if [ "$ROOTLESS" = true ]; then
        echo "Verify rootless mode:"
        echo "  sudo -u $PODMAN_USER podman info | grep rootless"
        echo "  sudo -u $PODMAN_USER podman run --rm alpine cat /proc/self/uid_map"
        echo ""
    fi
}

# ============================================================================
# MAIN
# ============================================================================
main() {
    if [ "$EUID" -ne 0 ]; then
        log_error "Please run as root"
        exit 1
    fi
    
    if [ "$CERTS_ONLY" = true ]; then
        generate_certificates
        output_client_config
        exit 0
    fi
    
    install_podman
    generate_certificates
    configure_podman_service
    configure_firewall
    test_connection
    output_client_config
}

main "$@"
