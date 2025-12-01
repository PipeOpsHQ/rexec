#!/bin/sh
set -e

# Rexec Entrypoint Script
# This script handles the setup required for connecting to a remote Docker daemon.
# Supports: TCP (plain), TCP with TLS, and SSH connections.

echo "=============================================="
echo "🚀 Rexec Entrypoint Starting"
echo "=============================================="
echo "Date: $(date)"
echo "User: $(whoami) (UID: $(id -u))"
echo "HOME: $HOME"
echo "PWD: $(pwd)"
echo ""
echo "Environment variables:"
echo "  DOCKER_HOST=${DOCKER_HOST:-<not set>}"
echo "  DOCKER_TLS_VERIFY=${DOCKER_TLS_VERIFY:-<not set>}"
echo "  DOCKER_CERT_PATH=${DOCKER_CERT_PATH:-<not set>}"
echo "  DOCKER_CA_CERT length: ${#DOCKER_CA_CERT} chars"
echo "  DOCKER_CLIENT_CERT length: ${#DOCKER_CLIENT_CERT} chars"
echo "  DOCKER_CLIENT_KEY length: ${#DOCKER_CLIENT_KEY} chars"
echo ""

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

# Function to write certificate content, handling escaped newlines
write_cert() {
    local content="$1"
    local file="$2"

    # Check if content is empty
    if [ -z "$content" ]; then
        echo "  ⚠ Warning: Empty certificate content for $file"
        return 1
    fi

    # Method 1: Try printf with escaped newlines (handles \n literals)
    # This converts literal \n to actual newlines
    printf '%b' "$content" | sed 's/\\n/\n/g' > "$file"

    # Verify the file looks like a valid PEM
    if grep -q "BEGIN" "$file" 2>/dev/null; then
        return 0
    fi

    # Method 2: If that didn't work, try echo -e (if available)
    if command -v echo >/dev/null 2>&1; then
        echo "$content" | sed 's/\\n/\n/g' > "$file"
        if grep -q "BEGIN" "$file" 2>/dev/null; then
            return 0
        fi
    fi

    # Method 3: Direct write (content might already have real newlines)
    printf '%s\n' "$content" > "$file"
    if grep -q "BEGIN" "$file" 2>/dev/null; then
        return 0
    fi

    echo "  ⚠ Warning: Certificate may be malformed in $file"
    return 1
}

# Setup TLS certificates if provided via environment variables
if [ -n "$DOCKER_CA_CERT" ] || [ -n "$DOCKER_CLIENT_CERT" ] || [ -n "$DOCKER_CLIENT_KEY" ]; then
    echo "📜 Configuring Docker TLS certificates..."

    # Create cert directory
    CERT_DIR="${DOCKER_CERT_PATH:-$HOME/.docker}"
    mkdir -p "$CERT_DIR"
    chmod 700 "$CERT_DIR"

    # Write CA certificate
    if [ -n "$DOCKER_CA_CERT" ]; then
        if write_cert "$DOCKER_CA_CERT" "$CERT_DIR/ca.pem"; then
            chmod 644 "$CERT_DIR/ca.pem"
            echo "  ✓ CA certificate configured ($(wc -l < "$CERT_DIR/ca.pem") lines)"
        fi
    fi

    # Write client certificate
    if [ -n "$DOCKER_CLIENT_CERT" ]; then
        if write_cert "$DOCKER_CLIENT_CERT" "$CERT_DIR/cert.pem"; then
            chmod 644 "$CERT_DIR/cert.pem"
            echo "  ✓ Client certificate configured ($(wc -l < "$CERT_DIR/cert.pem") lines)"
        fi
    fi

    # Write client key
    if [ -n "$DOCKER_CLIENT_KEY" ]; then
        if write_cert "$DOCKER_CLIENT_KEY" "$CERT_DIR/key.pem"; then
            chmod 600 "$CERT_DIR/key.pem"
            echo "  ✓ Client key configured ($(wc -l < "$CERT_DIR/key.pem") lines)"
        fi
    fi

    # Set cert path if not already set
    if [ -z "$DOCKER_CERT_PATH" ]; then
        export DOCKER_CERT_PATH="$CERT_DIR"
    fi

    # Enable TLS verification by default when certs are provided
    if [ -z "$DOCKER_TLS_VERIFY" ]; then
        export DOCKER_TLS_VERIFY=1
    fi

    # Debug: show first and last line of CA cert to verify format
    if [ -f "$CERT_DIR/ca.pem" ]; then
        echo "  📄 CA cert starts with: $(head -1 "$CERT_DIR/ca.pem")"
        echo "  📄 CA cert ends with: $(tail -1 "$CERT_DIR/ca.pem")"
    fi

    echo "✅ Docker TLS configuration complete"
    echo "   DOCKER_CERT_PATH=$DOCKER_CERT_PATH"
fi

# Setup SSH for remote Docker connection via SSH
if [ -n "$SSH_PRIVATE_KEY" ]; then
    echo "🔑 Configuring SSH for remote Docker connection..."

    # Ensure .ssh directory exists
    mkdir -p "$HOME/.ssh"
    chmod 700 "$HOME/.ssh"

    # Write the private key, handling escaped newlines
    write_cert "$SSH_PRIVATE_KEY" "$HOME/.ssh/id_rsa"
    chmod 600 "$HOME/.ssh/id_rsa"

    # Configure SSH to disable strict host key checking
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
        echo "   Cert path: ${DOCKER_CERT_PATH:-$HOME/.docker}"
    fi
else
    echo "🐳 Docker Host: unix:///var/run/docker.sock (default)"
fi

# Execute the main command (usually "rexec")
exec "$@"
