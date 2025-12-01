#!/bin/bash
set -e

# Start SSH server if not running
if [ ! -f /var/run/sshd.pid ] || ! kill -0 $(cat /var/run/sshd.pid) 2>/dev/null; then
    # Ensure SSH host keys exist
    if [ ! -f /etc/ssh/ssh_host_rsa_key ]; then
        ssh-keygen -A
    fi

    # Start SSH daemon
    /usr/sbin/sshd
fi

# If authorized_keys are mounted, copy them to the user's .ssh directory
if [ -f /home/user/.ssh/authorized_keys_mount ]; then
    cat /home/user/.ssh/authorized_keys_mount > /home/user/.ssh/authorized_keys
    chown user:user /home/user/.ssh/authorized_keys
    chmod 600 /home/user/.ssh/authorized_keys
fi

# If running as root, switch to user for the main process
if [ "$(id -u)" = "0" ]; then
    # Execute the command as the user
    if [ $# -eq 0 ]; then
        exec su - user
    else
        exec su - user -c "$*"
    fi
else
    # Already running as user
    if [ $# -eq 0 ]; then
        exec /bin/bash
    else
        exec "$@"
    fi
fi
