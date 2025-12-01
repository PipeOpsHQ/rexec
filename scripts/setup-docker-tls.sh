#!/bin/bash
set -e

# =============================================================================
# Docker TLS Certificate Generator
# =============================================================================
# This script generates TLS certificates for securing Docker daemon connections.
# Run this on your Docker host (VM) to enable secure remote access.
#
# Usage:
#   ./setup-docker-tls.sh [DOCKER_HOST_IP] [CERT_DIR]
#
# Examples:
#   ./setup-docker-tls.sh 123.45.67.89
#   ./setup-docker-tls.sh my-docker-host.example.com /opt/docker-certs
#
# After running this script:
#   1. Configure Docker daemon to use TLS (see instructions at end)
#   2. Copy client certificates to your Rexec deployment
#   3. Set environment variables in Railway/PipeOps
#
# =============================================================================

# Configuration
DOCKER_HOST_IP="${1:-$(hostname -I | awk '{print $1}')}"
CERT_DIR="${2:-/etc/docker/certs}"
DAYS_VALID=3650  # 10 years

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=============================================${NC}"
echo -e "${BLUE}  Docker TLS Certificate Generator${NC}"
echo -e "${BLUE}=============================================${NC}"
echo ""
echo -e "Docker Host: ${GREEN}${DOCKER_HOST_IP}${NC}"
echo -e "Cert Directory: ${GREEN}${CERT_DIR}${NC}"
echo -e "Valid for: ${GREEN}${DAYS_VALID} days${NC}"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: This script must be run as root${NC}"
    echo "Run: sudo $0 $*"
    exit 1
fi

# Create certificate directory
mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

echo -e "${YELLOW}Generating certificates...${NC}"
echo ""

# =============================================================================
# 1. Generate CA (Certificate Authority)
# =============================================================================
echo -e "${BLUE}[1/4] Generating CA private key...${NC}"
openssl genrsa -out ca-key.pem 4096

echo -e "${BLUE}[2/4] Generating CA certificate...${NC}"
openssl req -new -x509 -days $DAYS_VALID -key ca-key.pem -sha256 -out ca.pem \
    -subj "/C=US/ST=Cloud/L=Docker/O=Rexec/CN=Docker CA"

# =============================================================================
# 2. Generate Server Certificate
# =============================================================================
echo -e "${BLUE}[3/4] Generating server certificate...${NC}"

# Generate server key
openssl genrsa -out server-key.pem 4096

# Generate server CSR
openssl req -subj "/CN=$DOCKER_HOST_IP" -sha256 -new -key server-key.pem -out server.csr

# Create extfile for server cert
cat > extfile.cnf <<EOF
subjectAltName = DNS:localhost,DNS:$(hostname),IP:$DOCKER_HOST_IP,IP:127.0.0.1
extendedKeyUsage = serverAuth
EOF

# Sign server certificate
openssl x509 -req -days $DAYS_VALID -sha256 \
    -in server.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
    -out server-cert.pem -extfile extfile.cnf

# =============================================================================
# 3. Generate Client Certificate
# =============================================================================
echo -e "${BLUE}[4/4] Generating client certificate...${NC}"

# Generate client key
openssl genrsa -out key.pem 4096

# Generate client CSR
openssl req -subj '/CN=client' -new -key key.pem -out client.csr

# Create extfile for client cert
echo extendedKeyUsage = clientAuth > extfile-client.cnf

# Sign client certificate
openssl x509 -req -days $DAYS_VALID -sha256 \
    -in client.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
    -out cert.pem -extfile extfile-client.cnf

# =============================================================================
# 4. Cleanup and Set Permissions
# =============================================================================
echo ""
echo -e "${YELLOW}Setting permissions...${NC}"

# Remove intermediate files
rm -f server.csr client.csr extfile.cnf extfile-client.cnf

# Set restrictive permissions
chmod 0400 ca-key.pem server-key.pem key.pem
chmod 0444 ca.pem server-cert.pem cert.pem

# Create client bundle directory
CLIENT_DIR="$CERT_DIR/client"
mkdir -p "$CLIENT_DIR"
cp ca.pem cert.pem key.pem "$CLIENT_DIR/"
chmod 755 "$CLIENT_DIR"

echo ""
echo -e "${GREEN}=============================================${NC}"
echo -e "${GREEN}  Certificates Generated Successfully!${NC}"
echo -e "${GREEN}=============================================${NC}"
echo ""
echo -e "${YELLOW}Certificate Files:${NC}"
echo "  Server certs: $CERT_DIR/"
echo "    - ca.pem (CA certificate)"
echo "    - server-cert.pem (server certificate)"
echo "    - server-key.pem (server private key)"
echo ""
echo "  Client certs: $CLIENT_DIR/"
echo "    - ca.pem (CA certificate)"
echo "    - cert.pem (client certificate)"
echo "    - key.pem (client private key)"
echo ""

# =============================================================================
# 5. Configure Docker Daemon
# =============================================================================
echo -e "${YELLOW}Docker Daemon Configuration:${NC}"
echo ""
echo "This configuration will be applied to /etc/docker/daemon.json:"
echo ""
cat <<EOF
{
  "tls": true,
  "tlscacert": "$CERT_DIR/ca.pem",
  "tlscert": "$CERT_DIR/server-cert.pem",
  "tlskey": "$CERT_DIR/server-key.pem",
  "tlsverify": true
}
EOF
echo ""
echo -e "${YELLOW}Note:${NC} The TCP listener will be added via systemd override to avoid conflicts."
echo ""

# Check if we should auto-configure
read -p "Apply this configuration now? (y/N) " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    # Backup existing config
    if [ -f /etc/docker/daemon.json ]; then
        cp /etc/docker/daemon.json /etc/docker/daemon.json.bak
        echo -e "${YELLOW}Backed up existing config to /etc/docker/daemon.json.bak${NC}"
    fi

    # Write daemon.json WITHOUT hosts directive (this avoids conflicts with systemd)
    # The hosts are configured in the systemd override instead
    cat > /etc/docker/daemon.json <<EOF
{
  "tls": true,
  "tlscacert": "$CERT_DIR/ca.pem",
  "tlscert": "$CERT_DIR/server-cert.pem",
  "tlskey": "$CERT_DIR/server-key.pem",
  "tlsverify": true
}
EOF

    # Create systemd override to add TCP listener alongside unix socket
    # This keeps the unix socket working for local SSH/console access
    mkdir -p /etc/systemd/system/docker.service.d/
    cat > /etc/systemd/system/docker.service.d/override.conf <<EOF
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd -H fd:// -H unix:///var/run/docker.sock -H tcp://0.0.0.0:2376
EOF

    echo -e "${YELLOW}Restarting Docker daemon...${NC}"
    systemctl daemon-reload
    systemctl restart docker

    # Wait for Docker to be ready
    sleep 2

    # Verify Docker is running
    if ! systemctl is-active --quiet docker; then
        echo -e "${RED}✗ Docker failed to start! Checking logs...${NC}"
        journalctl -u docker -n 20 --no-pager
        echo ""
        echo -e "${YELLOW}Restoring backup configuration...${NC}"
        if [ -f /etc/docker/daemon.json.bak ]; then
            mv /etc/docker/daemon.json.bak /etc/docker/daemon.json
        fi
        rm -f /etc/systemd/system/docker.service.d/override.conf
        systemctl daemon-reload
        systemctl restart docker
        echo -e "${RED}Configuration rolled back. Please check the error above.${NC}"
        exit 1
    fi

    echo -e "${GREEN}Docker daemon configured and restarted!${NC}"
    echo ""

    # Verify local socket still works (important for SSH access)
    echo -e "${YELLOW}Verifying local socket access...${NC}"
    if docker version > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Local socket (unix:///var/run/docker.sock) works!${NC}"
    else
        echo -e "${RED}✗ Local socket access failed!${NC}"
    fi

    # Test TLS connection
    echo -e "${YELLOW}Testing TLS connection...${NC}"
    if docker --tlsverify \
        --tlscacert=$CERT_DIR/ca.pem \
        --tlscert=$CLIENT_DIR/cert.pem \
        --tlskey=$CLIENT_DIR/key.pem \
        -H=tcp://127.0.0.1:2376 version > /dev/null 2>&1; then
        echo -e "${GREEN}✓ TLS connection (tcp://127.0.0.1:2376) works!${NC}"
    else
        echo -e "${RED}✗ TLS connection failed. Check Docker logs: journalctl -u docker${NC}"
    fi
fi

echo ""
echo -e "${YELLOW}=============================================${NC}"
echo -e "${YELLOW}  Environment Variables for Rexec${NC}"
echo -e "${YELLOW}=============================================${NC}"
echo ""
echo "Set these in your Railway/PipeOps deployment:"
echo ""
echo -e "${GREEN}DOCKER_HOST${NC}=tcp://${DOCKER_HOST_IP}:2376"
echo -e "${GREEN}DOCKER_TLS_VERIFY${NC}=1"
echo ""
echo -e "${GREEN}DOCKER_CA_CERT${NC}="
echo "$(cat $CLIENT_DIR/ca.pem)"
echo ""
echo -e "${GREEN}DOCKER_CLIENT_CERT${NC}="
echo "$(cat $CLIENT_DIR/cert.pem)"
echo ""
echo -e "${GREEN}DOCKER_CLIENT_KEY${NC}="
echo "$(cat $CLIENT_DIR/key.pem)"
echo ""

# =============================================================================
# 6. Firewall Reminder
# =============================================================================
echo -e "${YELLOW}=============================================${NC}"
echo -e "${YELLOW}  Firewall Configuration${NC}"
echo -e "${YELLOW}=============================================${NC}"
echo ""
echo "Make sure to open port 2376 in your firewall:"
echo ""
echo "  # UFW"
echo "  sudo ufw allow 2376/tcp"
echo ""
echo "  # iptables"
echo "  sudo iptables -A INPUT -p tcp --dport 2376 -j ACCEPT"
echo ""
echo "  # firewalld"
echo "  sudo firewall-cmd --permanent --add-port=2376/tcp"
echo "  sudo firewall-cmd --reload"
echo ""
echo -e "${GREEN}Setup complete!${NC}"
echo ""
echo -e "${BLUE}Summary:${NC}"
echo "  - Local access (SSH/console): docker commands work normally"
echo "  - Remote access (TLS): tcp://${DOCKER_HOST_IP}:2376"
