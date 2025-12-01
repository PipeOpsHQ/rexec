#!/bin/bash
set -e

# =============================================================================
# Rexec Remote Docker Host Setup
# =============================================================================
# This script sets up a VM to serve as a remote Docker host for Rexec.
# Run this on a fresh Ubuntu/Debian VM (Hetzner, DigitalOcean, Linode, etc.)
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/your-repo/rexec/main/docker/docker-host/setup.sh | sudo bash
#   # or
#   sudo ./setup.sh
#
# =============================================================================

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=============================================${NC}"
echo -e "${BLUE}  Rexec Remote Docker Host Setup${NC}"
echo -e "${BLUE}=============================================${NC}"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: This script must be run as root${NC}"
    echo "Run: sudo $0"
    exit 1
fi

# Detect public IP
PUBLIC_IP=$(curl -s ifconfig.me || curl -s icanhazip.com || hostname -I | awk '{print $1}')
echo -e "Detected public IP: ${GREEN}${PUBLIC_IP}${NC}"
echo ""

# =============================================================================
# 1. Install Docker
# =============================================================================
echo -e "${YELLOW}[1/5] Installing Docker...${NC}"

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

echo ""

# =============================================================================
# 2. Generate TLS Certificates
# =============================================================================
echo -e "${YELLOW}[2/5] Generating TLS certificates...${NC}"

CERT_DIR="/etc/docker/certs"
DAYS_VALID=3650

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

# =============================================================================
# 3. Configure Docker Daemon
# =============================================================================
echo -e "${YELLOW}[3/5] Configuring Docker daemon for TLS...${NC}"

# Backup existing config
if [ -f /etc/docker/daemon.json ]; then
    cp /etc/docker/daemon.json /etc/docker/daemon.json.bak
fi

# Write daemon config (WITHOUT hosts directive to avoid systemd conflicts)
# The TCP listener is added via systemd override instead
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

# Create systemd override to add TCP listener alongside unix socket
# This keeps the unix socket working for local SSH/console access
mkdir -p /etc/systemd/system/docker.service.d/
cat > /etc/systemd/system/docker.service.d/override.conf <<EOF
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd -H fd:// -H unix:///var/run/docker.sock -H tcp://0.0.0.0:2376
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

# =============================================================================
# 4. Configure Firewall
# =============================================================================
echo -e "${YELLOW}[4/5] Configuring firewall...${NC}"

# Try UFW first (Ubuntu/Debian)
if command -v ufw &> /dev/null; then
    ufw allow 2376/tcp
    echo -e "${GREEN}UFW: Opened port 2376${NC}"
# Try firewalld (CentOS/RHEL/Fedora)
elif command -v firewall-cmd &> /dev/null; then
    firewall-cmd --permanent --add-port=2376/tcp
    firewall-cmd --reload
    echo -e "${GREEN}firewalld: Opened port 2376${NC}"
# Fall back to iptables
elif command -v iptables &> /dev/null; then
    iptables -A INPUT -p tcp --dport 2376 -j ACCEPT
    echo -e "${GREEN}iptables: Opened port 2376${NC}"
else
    echo -e "${YELLOW}No firewall detected. Make sure port 2376 is accessible.${NC}"
fi

echo ""

# =============================================================================
# 5. Test Connection
# =============================================================================
echo -e "${YELLOW}[5/5] Testing TLS connection...${NC}"

sleep 2  # Wait for Docker to fully restart

if docker --tlsverify \
    --tlscacert=$CERT_DIR/ca.pem \
    --tlscert=$CERT_DIR/client/cert.pem \
    --tlskey=$CERT_DIR/client/key.pem \
    -H=tcp://127.0.0.1:2376 version > /dev/null 2>&1; then
    echo -e "${GREEN}✓ TLS connection successful!${NC}"
else
    echo -e "${RED}✗ TLS connection failed${NC}"
    echo "Check Docker logs: journalctl -u docker -n 50"
    exit 1
fi

echo ""

# =============================================================================
# Output Environment Variables
# =============================================================================
echo -e "${GREEN}=============================================${NC}"
echo -e "${GREEN}  Setup Complete!${NC}"
echo -e "${GREEN}=============================================${NC}"
echo ""
echo -e "${YELLOW}Set these environment variables in your Railway/PipeOps deployment:${NC}"
echo ""
echo -e "${BLUE}DOCKER_HOST${NC}=tcp://${PUBLIC_IP}:2376"
echo -e "${BLUE}DOCKER_TLS_VERIFY${NC}=1"
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
cat > /root/rexec-docker-env.txt <<EOF
# Rexec Docker Host Environment Variables
# Generated on $(date)
# Docker Host: $PUBLIC_IP

DOCKER_HOST=tcp://${PUBLIC_IP}:2376
DOCKER_TLS_VERIFY=1

# CA Certificate
DOCKER_CA_CERT=$(cat "$CERT_DIR/client/ca.pem" | base64 -w0)

# Client Certificate
DOCKER_CLIENT_CERT=$(cat "$CERT_DIR/client/cert.pem" | base64 -w0)

# Client Key
DOCKER_CLIENT_KEY=$(cat "$CERT_DIR/client/key.pem" | base64 -w0)
EOF

echo -e "${GREEN}Environment variables also saved to: /root/rexec-docker-env.txt${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Copy the environment variables above to your deployment platform"
echo "2. Deploy Rexec using Dockerfile.remote"
echo "3. Test the connection from your Rexec instance"
echo ""
echo -e "${GREEN}Your Docker host is ready at: tcp://${PUBLIC_IP}:2376${NC}"
echo ""
echo -e "${BLUE}Summary:${NC}"
echo "  - Local access (SSH/console): docker commands work normally"
echo "  - Remote access (TLS): tcp://${PUBLIC_IP}:2376"
