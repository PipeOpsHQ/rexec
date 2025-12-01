# Frontend build stage
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web-ui

# Copy package files and install dependencies
COPY web-ui/package.json ./

# Install dependencies
RUN npm install

# Copy source files
COPY web-ui/ ./

# Build the Svelte app (outputs to ../web)
RUN npm run build

# Go build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend from frontend-builder
COPY --from=frontend-builder /app/web ./web

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rexec ./cmd/rexec

# Runtime stage - using Docker-in-Docker Rootless
# This image contains the necessary binaries and configuration to run Docker
# without root privileges inside the container.
FROM docker:27-dind-rootless

# Switch to root temporarily to setup directories and install deps
USER root

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    git \
    bash

# Create application directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/rexec /usr/local/bin/rexec

# Copy web directory for frontend
COPY --from=builder /app/web /app/web

# Copy standalone startup script
COPY scripts/start-standalone.sh /usr/local/bin/start-standalone.sh
RUN chmod +x /usr/local/bin/start-standalone.sh

# Create all necessary directories and ensure permissions for the rootless user (UID 1000)
# These directories must be writable at runtime for rootless Docker
RUN mkdir -p /var/lib/rexec/volumes && \
    mkdir -p /run/user/1000 && \
    mkdir -p /home/rootless/.local/share/docker && \
    mkdir -p /home/rootless/.docker && \
    chown -R rootless:rootless /var/lib/rexec && \
    chown -R rootless:rootless /app && \
    chown -R rootless:rootless /run/user/1000 && \
    chown -R rootless:rootless /home/rootless && \
    chmod 700 /run/user/1000

# Switch back to the unprivileged user provided by the base image
USER rootless

# Expose the API port
EXPOSE 8080

# Set environment variables for rootless Docker
ENV PORT=8080
ENV HOME=/home/rootless
ENV XDG_RUNTIME_DIR=/run/user/1000
ENV DOCKER_HOST=unix:///run/user/1000/docker.sock

# Use the standalone startup script
ENTRYPOINT ["start-standalone.sh"]
