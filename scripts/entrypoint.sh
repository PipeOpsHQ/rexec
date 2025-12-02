#!/bin/sh
set -e

# Rexec Entrypoint Script
# Handles setup for connecting to a remote Docker daemon via TLS or SSH.

# Function to write PEM certificate content
# Handles the case where PaaS platforms replace newlines with spaces
write_pem() {
    local content="$1"
    local file="$2"

    if [ -z "$content" ]; then
        return 1
    fi

    # Check if content already has proper newlines (more than 5 lines)
    local line_count
    line_count=$(printf '%s\n' "$content" | wc -l)

    if [ "$line_count" -gt 5 ]; then
        # Already has newlines, write directly
        printf '%s\n' "$content" > "$file"
    else
        # Content is on a single line with spaces instead of newlines
        # Reconstruct proper PEM format
        printf '%s' "$content" | sed -E '
            s/(-----BEGIN [A-Z ]+-----) /\1\n/g
            s/ (-----END [A-Z ]+-----)/\n\1/g
        ' | awk '
        {
            if (/^-----BEGIN/ || /^-----END/) {
                print
            } else {
                gsub(/ /, "\n")
                print
            }
        }' > "$file"
    fi

    # Verify the file has proper PEM structure
    if grep -q "^-----BEGIN" "$file" && grep -q "^-----END" "$file"; then
        return 0
    else
        return 1
    fi
}

# Setup TLS certificates if provided via environment variables
if [ -n "$DOCKER_CA_CERT" ] || [ -n "$DOCKER_CLIENT_CERT" ] || [ -n "$DOCKER_CLIENT_KEY" ]; then
    # Create cert directory
    CERT_DIR="${DOCKER_CERT_PATH:-$HOME/.docker}"
    mkdir -p "$CERT_DIR"
    chmod 700 "$CERT_DIR"

    # Write CA certificate
    if [ -n "$DOCKER_CA_CERT" ]; then
        write_pem "$DOCKER_CA_CERT" "$CERT_DIR/ca.pem"
        chmod 644 "$CERT_DIR/ca.pem"
    fi

    # Write client certificate
    if [ -n "$DOCKER_CLIENT_CERT" ]; then
        write_pem "$DOCKER_CLIENT_CERT" "$CERT_DIR/cert.pem"
        chmod 644 "$CERT_DIR/cert.pem"
    fi

    # Write client key
    if [ -n "$DOCKER_CLIENT_KEY" ]; then
        write_pem "$DOCKER_CLIENT_KEY" "$CERT_DIR/key.pem"
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
fi

# Setup SSH for remote Docker connection via SSH
if [ -n "$SSH_PRIVATE_KEY" ]; then
    mkdir -p "$HOME/.ssh"
    chmod 700 "$HOME/.ssh"

    # Write the private key
    write_pem "$SSH_PRIVATE_KEY" "$HOME/.ssh/id_rsa"
    chmod 600 "$HOME/.ssh/id_rsa"

    # Configure SSH
    cat > "$HOME/.ssh/config" <<EOF
Host *
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    LogLevel ERROR
    ServerAliveInterval 60
    ServerAliveCountMax 3
EOF
    chmod 600 "$HOME/.ssh/config"
fi

# Execute the main command
exec "$@"
