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

# Function to write certificate content, handling various formats:
# - Proper PEM with newlines
# - Single line with spaces instead of newlines (common in PaaS env vars)
# - Single line with literal \n
write_cert() {
    local content="$1"
    local file="$2"
    local desc="$3"

    # Check if content is empty
    if [ -z "$content" ]; then
        echo "  ⚠ Warning: Empty certificate content for $desc"
        return 1
    fi

    # Detect format and convert to proper PEM
    # Check if it's already multi-line (has actual newlines)
    line_count=$(printf '%s' "$content" | wc -l)

    if [ "$line_count" -gt 1 ]; then
        # Already has newlines, write directly
        printf '%s\n' "$content" > "$file"
    elif echo "$content" | grep -q '\\n'; then
        # Has literal \n sequences, convert them
        printf '%s' "$content" | sed 's/\\n/\n/g' > "$file"
    elif echo "$content" | grep -q ' '; then
        # Single line with spaces (PaaS replaced newlines with spaces)
        # This is the tricky case - we need to reconstruct the PEM format

        # Extract the BEGIN and END markers and the base64 content
        # Format: -----BEGIN CERTIFICATE----- BASE64DATA -----END CERTIFICATE-----

        # Use awk to properly format the certificate
        printf '%s' "$content" | awk '
        {
            # Replace spaces between BEGIN/END markers with newlines
            # First, handle the BEGIN line
            gsub(/-----BEGIN [A-Z ]+----- /, "-----BEGIN CERTIFICATE-----\n")
            gsub(/ -----END [A-Z ]+-----/, "\n-----END CERTIFICATE-----")

            # Now split the base64 content into 64-char lines
            # Remove the markers temporarily
            if (match($0, /-----BEGIN [A-Z ]+-----/)) {
                start = RSTART
                end = RSTART + RLENGTH
            }

            # Just print with newlines restored
            gsub(/ /, "\n")
            print
        }' > "$file"

        # Simpler approach: replace all spaces with newlines, then fix the markers
        printf '%s' "$content" | tr ' ' '\n' > "$file"
    else
        # No spaces, no \n - might be malformed or very short
        printf '%s\n' "$content" > "$file"
    fi

    # Verify the file has proper PEM structure
    if grep -q "BEGIN" "$file" && grep -q "END" "$file"; then
        local lines=$(wc -l < "$file")
        echo "  ✓ $desc configured ($lines lines)"
        return 0
    else
        echo "  ⚠ Warning: $desc may be malformed"
        return 1
    fi
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
        write_cert "$DOCKER_CA_CERT" "$CERT_DIR/ca.pem" "CA certificate"
        chmod 644 "$CERT_DIR/ca.pem"
    fi

    # Write client certificate
    if [ -n "$DOCKER_CLIENT_CERT" ]; then
        write_cert "$DOCKER_CLIENT_CERT" "$CERT_DIR/cert.pem" "Client certificate"
        chmod 644 "$CERT_DIR/cert.pem"
    fi

    # Write client key
    if [ -n "$DOCKER_CLIENT_KEY" ]; then
        write_cert "$DOCKER_CLIENT_KEY" "$CERT_DIR/key.pem" "Client key"
        chmod 600 "$CERT_DIR/key.pem"
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
        echo "  📄 CA cert preview:"
        echo "      First line: $(head -1 "$CERT_DIR/ca.pem")"
        echo "      Last line:  $(tail -1 "$CERT_DIR/ca.pem")"
        echo "      Total lines: $(wc -l < "$CERT_DIR/ca.pem")"
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
    write_cert "$SSH_PRIVATE_KEY" "$HOME/.ssh/id_rsa" "SSH private key"
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

echo ""
echo "Starting application..."
echo "=============================================="

# Execute the main command (usually "rexec")
exec "$@"
