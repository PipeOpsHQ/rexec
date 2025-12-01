#!/bin/sh
set -e

# Rexec Entrypoint Script
# This script handles the setup required for connecting to a remote Docker daemon via SSH.
# It allows running Rexec without a local Docker socket or privileged mode.

# Check if SSH_PRIVATE_KEY is provided
if [ -n "$SSH_PRIVATE_KEY" ]; then
    echo "Configuring SSH for remote Docker connection..."

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
EOF
    chmod 600 "$HOME/.ssh/config"

    echo "SSH configuration complete."
fi

# Execute the main command (usually "rexec")
exec "$@"
