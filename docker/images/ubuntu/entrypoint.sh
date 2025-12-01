#!/bin/bash
set -e

# Start SSH server if not running (use sudo since we run as non-root user)
if [ ! -f /var/run/sshd.pid ] || ! sudo kill -0 $(cat /var/run/sshd.pid 2>/dev/null) 2>/dev/null; then
    # Ensure SSH host keys exist
    if [ ! -f /etc/ssh/ssh_host_rsa_key ]; then
        sudo ssh-keygen -A
    fi

    # Start SSH daemon
    sudo /usr/sbin/sshd
fi

# If authorized_keys are mounted, copy them to the user's .ssh directory
if [ -f /home/user/.ssh/authorized_keys_mount ]; then
    cat /home/user/.ssh/authorized_keys_mount > /home/user/.ssh/authorized_keys
    chmod 600 /home/user/.ssh/authorized_keys
fi

# Execute the command or start shell
if [ $# -eq 0 ]; then
    exec /bin/bash
else
    exec "$@"
fi
