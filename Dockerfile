# Rexec Dockerfile (Remote Docker Daemon)
# ========================================
# This Dockerfile builds Rexec configured to connect to a remote Docker daemon.
# No local Docker daemon is required - set DOCKER_HOST to your remote Docker host.
#
# Environment Variables:
# ----------------------
# DOCKER_HOST          - Remote Docker daemon endpoint (required)
#                        Examples:
#                          tcp://your-docker-host:2376   (TLS)
#                          tcp://your-docker-host:2375   (no TLS, not recommended)
#                          ssh://user@your-docker-host   (via SSH)
#
# For TLS connections (recommended):
#   DOCKER_TLS_VERIFY  - Set to "1" to enable TLS verification
#   DOCKER_CERT_PATH   - Path to TLS certificates (default: ~/.docker)
#
#   Or provide certificates directly:
#   DOCKER_CA_CERT     - CA certificate content (PEM format)
#   DOCKER_CLIENT_CERT - Client certificate content (PEM format)
#   DOCKER_CLIENT_KEY  - Client private key content (PEM format)
#
# For SSH connections:
#   PORT               - HTTP port (default: 8080)
#   GIN_MODE           - Gin mode: debug or release (default: release)
#   DATABASE_URL       - PostgreSQL connection string
#   JWT_SECRET         - Secret for signing JWT tokens

# Frontend build stage
FROM node:22-alpine AS frontend-builder

WORKDIR /app/web-ui

# Copy package.json and package-lock.json for reproducible installs
COPY frontend/package.json ./
COPY frontend/package-lock.json ./

# Install dependencies using npm ci (clean install)
RUN npm ci && npm rebuild

# Copy source files
COPY frontend/ ./

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

# Build agent binaries for multiple platforms
RUN mkdir -p downloads && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o downloads/rexec-agent-linux-amd64 ./cmd/rexec-agent && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o downloads/rexec-agent-linux-arm64 ./cmd/rexec-agent && \
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o downloads/rexec-agent-darwin-amd64 ./cmd/rexec-agent && \
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o downloads/rexec-agent-darwin-arm64 ./cmd/rexec-agent

# Build CLI binaries for multiple platforms
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o downloads/rexec-cli-linux-amd64 ./cmd/rexec-cli && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o downloads/rexec-cli-linux-arm64 ./cmd/rexec-cli && \
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o downloads/rexec-cli-darwin-amd64 ./cmd/rexec-cli && \
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o downloads/rexec-cli-darwin-arm64 ./cmd/rexec-cli

# Runtime stage - lightweight Alpine
FROM alpine:3.20

# Install runtime dependencies
# - ca-certificates: For TLS connections
# - tzdata: For timezone support
# - docker-cli: For Docker commands (uses DOCKER_HOST)
# - openssh-client: For SSH-based Docker connections
# - curl: For health checks and debugging
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    docker-cli \
    openssh-client \
    curl

# Create non-root user
RUN adduser -D -g '' -u 1000 rexec

# Create necessary directories
RUN mkdir -p /var/lib/rexec/volumes && \
    mkdir -p /home/rexec/.docker && \
    mkdir -p /home/rexec/.ssh && \
    mkdir -p /app/recordings && \
    chown -R rexec:rexec /var/lib/rexec /home/rexec /app/recordings

# Volume for persistent recordings
VOLUME /recordings

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/rexec /usr/local/bin/rexec

# Copy web directory for frontend
COPY --from=builder /app/web /app/web

# Copy install scripts for static serving
COPY scripts/install-cli.sh /app/scripts/install-cli.sh
COPY scripts/install-agent.sh /app/scripts/install-agent.sh

# Copy downloadable binaries from builder
COPY --from=builder /app/downloads /app/downloads

# Copy entrypoint script
COPY scripts/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Set ownership
RUN chown -R rexec:rexec /app && mkdir -p /app/recordings

# Switch to non-root user
# USER rexec

# Set HOME for the rexec user (needed for .docker and .ssh directories)
ENV HOME=/home/rexec

# Expose the API port
EXPOSE 8080

# Set default environment variables
ENV PORT=8080
ENV RECORDINGS_PATH=/app/recordings
ENV SCRIPTS_DIR=/app/scripts
ENV DOWNLOADS_DIR=/app/downloads

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["/usr/local/bin/rexec", "server"]
