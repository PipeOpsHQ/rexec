#!/bin/sh
set -e

# Rexec Standalone Startup Script
# This script initializes a local Docker daemon (Rootless) in the background
# and then starts the Rexec application.

echo "--> Starting Rexec in Standalone Mode"

# Check for required binaries
if ! command -v rexec >/dev/null 2>&1; then
    echo "Error: 'rexec' binary not found in PATH."
    echo "This script is intended to run inside the rexec-standalone container."
    exit 1
fi

# Ensure XDG_RUNTIME_DIR is set for rootless docker
if [ -z "$XDG_RUNTIME_DIR" ]; then
    export XDG_RUNTIME_DIR=/run/user/$(id -u)
    mkdir -p "$XDG_RUNTIME_DIR"
fi

# Start the Docker daemon
# We look for dockerd-rootless.sh which is standard in rootless-capable images
if command -v dockerd-rootless.sh >/dev/null 2>&1; then
    echo "--> Found dockerd-rootless.sh in PATH, starting daemon..."
    dockerd-rootless.sh &
    DOCKER_PID=$!
elif [ -x /usr/local/bin/dockerd-rootless.sh ]; then
    echo "--> Found /usr/local/bin/dockerd-rootless.sh, starting daemon..."
    /usr/local/bin/dockerd-rootless.sh &
    DOCKER_PID=$!
elif [ -x /usr/local/bin/dockerd-entrypoint.sh ]; then
    echo "--> Found /usr/local/bin/dockerd-entrypoint.sh, starting daemon..."
    /usr/local/bin/dockerd-entrypoint.sh &
    DOCKER_PID=$!
elif command -v dockerd >/dev/null 2>&1; then
    echo "--> dockerd-rootless.sh not found, attempting standard dockerd..."
    dockerd &
    DOCKER_PID=$!
else
    echo "Error: Neither 'dockerd-rootless.sh' nor 'dockerd' found."
    echo "Cannot start Docker daemon."
    exit 1
fi

# Wait for the daemon to be ready
echo "--> Waiting for Docker daemon to initialize..."
MAX_RETRIES=30
i=0

# Check if docker cli exists
if ! command -v docker >/dev/null 2>&1; then
     echo "Error: 'docker' CLI not found."
     exit 1
fi

while ! docker info >/dev/null 2>&1; do
    i=$((i+1))
    if [ $i -ge $MAX_RETRIES ]; then
        echo "--> Error: Docker daemon failed to start within $MAX_RETRIES seconds."
        exit 1
    fi
    echo "    ... waiting ($i/$MAX_RETRIES)"
    sleep 1
done

echo "--> Docker daemon is active!"

# Start Rexec
echo "--> Starting Rexec API..."
# Execute rexec in the background so we can trap signals
rexec "$@" &
REXEC_PID=$!

# Handle shutdown signals
cleanup() {
    echo "--> Stopping Rexec..."
    kill -TERM "$REXEC_PID" 2>/dev/null
    wait "$REXEC_PID"

    echo "--> Stopping Docker daemon..."
    kill -TERM "$DOCKER_PID" 2>/dev/null
    wait "$DOCKER_PID"

    exit 0
}

trap cleanup SIGTERM SIGINT

# Wait for rexec to finish
wait "$REXEC_PID"
