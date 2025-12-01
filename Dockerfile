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
#   SSH_PRIVATE_KEY    - SSH private key content for authentication
#
# Database:
#   DATABASE_URL       - PostgreSQL connection string
#   REDIS_URL          - Redis connection string (optional)

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
    chown -R rexec:rexec /var/lib/rexec /home/rexec

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/rexec /usr/local/bin/rexec

# Copy web directory for frontend
COPY --from=builder /app/web /app/web

# Copy entrypoint script
COPY scripts/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Set ownership
RUN chown -R rexec:rexec /app

# Switch to non-root user
USER rexec

# Set HOME for the rexec user (needed for .docker and .ssh directories)
ENV HOME=/home/rexec

# Expose the API port
EXPOSE 8080

# Set default environment variables
ENV PORT=8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["entrypoint.sh"]
CMD ["rexec"]
