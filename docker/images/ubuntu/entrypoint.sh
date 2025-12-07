#!/bin/bash
set -e

# Auto-fix common dpkg/apt issues
fix_dpkg() {
    # Check for corrupted dpkg updates
    if [ -d /var/lib/dpkg/updates ] && [ "$(ls -A /var/lib/dpkg/updates 2>/dev/null)" ]; then
        # Test if dpkg is broken
        if ! dpkg --configure -a >/dev/null 2>&1; then
            # Remove corrupted update files
            rm -f /var/lib/dpkg/updates/* 2>/dev/null || true
            dpkg --configure -a 2>/dev/null || true
        fi
    fi
    
    # Fix interrupted apt operations
    if [ -f /var/lib/dpkg/lock-frontend ]; then
        rm -f /var/lib/dpkg/lock-frontend 2>/dev/null || true
    fi
    if [ -f /var/lib/apt/lists/lock ]; then
        rm -f /var/lib/apt/lists/lock 2>/dev/null || true
    fi
    if [ -f /var/cache/apt/archives/lock ]; then
        rm -f /var/cache/apt/archives/lock 2>/dev/null || true
    fi
}

# Run dpkg fix silently in background
fix_dpkg &

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
