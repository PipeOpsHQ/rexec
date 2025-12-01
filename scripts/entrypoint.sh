#!/bin/sh
set -e

# Rexec Entrypoint Script
# This script handles the setup required for connecting to a remote Docker daemon.
# Supports: TCP (plain), TCP with TLS, and SSH connections.

echo "🚀 Rexec Entrypoint"

# ============================================================================
# Docker Connection Configuration
# ============================================================================
#
# Environment Variables:
#
# DOCKER_HOST - The Docker daemon endpoint
#   Examples:
#     - unix:///var/run/docker.sock     (local socket, default)
#     - tcp://docker-host:2375          (remote, no TLS)
#     - tcp://docker-host:2376          (remote, with TLS)
#     - ssh://user@docker-host          (remote via SSH)
#
# For TLS connections (tcp:// with port 2376):
#   DOCKER_TLS_VERIFY=1                 - Enable TLS verification
#   DOCKER_CERT_PATH=/path/to/certs     - Path to TLS certificates
#
#   Or provide certificates directly via environment:
#   DOCKER_CA_CERT     - CA certificate content (PEM)
#   DOCKER_CLIENT_CERT - Client certificate content (PEM)
#   DOCKER_CLIENT_KEY  - Client private key content (PEM)
#
# For SSH connections (ssh://):
#   SSH_PRIVATE_KEY    - SSH private key content
#
# ============================================================================

# Setup TLS certificates if provided via environment variables
if [ -n "$DOCKER_CA_CERT" ] || [ -n "$DOCKER_CLIENT_CERT" ] || [ -n "$DOCKER_CLIENT_KEY" ]; then
    echo "📜 Configuring Docker TLS certificates..."

    # Create cert directory
    CERT_DIR="${DOCKER_CERT_PATH:-$HOME/.docker}"
    mkdir -p "$CERT_DIR"
    chmod 700 "$CERT_DIR"

    # Write CA certificate
    if [ -n "$DOCKER_CA_CERT" ]; then
        echo "$DOCKER_CA_CERT" > "$CERT_DIR/ca.pem"
        chmod 644 "$CERT_DIR/ca.pem"
        echo "  ✓ CA certificate configured"
    fi

    # Write client certificate
    if [ -n "$DOCKER_CLIENT_CERT" ]; then
        echo "$DOCKER_CLIENT_CERT" > "$CERT_DIR/cert.pem"
        chmod 644 "$CERT_DIR/cert.pem"
        echo "  ✓ Client certificate configured"
    fi

    # Write client key
    if [ -n "$DOCKER_CLIENT_KEY" ]; then
        echo "$DOCKER_CLIENT_KEY" > "$CERT_DIR/key.pem"
        chmod 600 "$CERT_DIR/key.pem"
        echo "  ✓ Client key configured"
    fi

    # Set cert path if not already set
    if [ -z "$DOCKER_CERT_PATH" ]; then
        export DOCKER_CERT_PATH="$CERT_DIR"
    fi

    # Enable TLS verification by default when certs are provided
    if [ -z "$DOCKER_TLS_VERIFY" ]; then
        export DOCKER_TLS_VERIFY=1
    fi

    echo "✅ Docker TLS configuration complete"
fi

# Setup SSH for remote Docker connection via SSH
if [ -n "$SSH_PRIVATE_KEY" ]; then
    echo "🔑 Configuring SSH for remote Docker connection..."

    # Ensure .ssh directory exists
    mkdir -p "$HOME/.ssh"
    chmod 700 "$HOME/.ssh"

    # Write the private key to id_rsa
    echo "$SSH_PRIVATE_KEY" > "$HOME/.ssh/id_rsa"
    chmod 600 "$HOME/.ssh/id_rsa"

    # Configure SSH to disable strict host key checking
    # This is necessary for connecting to remote hosts without manual intervention
    cat > "$HOME/.ssh/config" <<EOF
Host *
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    LogLevel ERROR
    ServerAliveInterval 60
    ServerAliveCountMax 3
EOF
    chmod 600 "$HOME/.ssh/config"

    echo "✅ SSH configuration complete"
fi

# Log Docker connection info
if [ -n "$DOCKER_HOST" ]; then
    echo "🐳 Docker Host: $DOCKER_HOST"
    if [ -n "$DOCKER_TLS_VERIFY" ] && [ "$DOCKER_TLS_VERIFY" = "1" ]; then
        echo "🔒 TLS verification enabled"
    fi
else
    echo "🐳 Docker Host: unix:///var/run/docker.sock (default)"
fi

# Execute the main command (usually "rexec")
exec "$@"
