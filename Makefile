.PHONY: build run dev clean test docker-build docker-run help images ui ui-dev ui-install cli cli-all agent-all cli-all-platforms tui-all-platforms ssh-gateway ssh-gateway-all dist downloads-dir embed embed-install embed-dev firecracker-setup firecracker-guest-agent firecracker-rootfs

# Variables
BINARY_NAME=rexec
CLI_NAME=rexec-cli
TUI_NAME=rexec-tui
AGENT_NAME=rexec-agent
SSH_NAME=rexec-ssh
DOCKER_IMAGE=rexec-api
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# Build the web UI
ui-install:
	@echo "Installing web UI dependencies..."
	cd frontend && npm install

ui: ui-install
	@echo "Building web UI..."
	cd frontend && npm run build

ui-dev:
	@echo "Starting web UI dev server..."
	cd frontend && npm run dev

# Build the embed widget
embed-install:
	@echo "Installing embed widget dependencies..."
	cd embed && npm install

embed: embed-install
	@echo "Building embed widget..."
	cd embed && npm run build
	@echo "Copying embed widget to web/embed..."
	@mkdir -p web/embed
	cp embed/dist/rexec.min.js web/embed/
	cp embed/dist/rexec.esm.js web/embed/
	cp embed/dist/rexec.min.js.map web/embed/ 2>/dev/null || true
	@echo "Embed widget built and copied to web/embed/"

embed-dev:
	@echo "Starting embed widget dev build with watch..."
	cd embed && npm run dev

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -ldflags "-X main.Version=$(VERSION)" -o bin/$(BINARY_NAME) ./cmd/rexec

# Build CLI tools
cli:
	@echo "Building $(CLI_NAME)..."
	$(GOBUILD) -ldflags "-X main.Version=$(VERSION)" -o bin/$(CLI_NAME) ./cmd/rexec-cli

tui:
	@echo "Building $(TUI_NAME)..."
	$(GOBUILD) -ldflags "-X main.Version=$(VERSION)" -o bin/$(TUI_NAME) ./cmd/rexec-tui

agent:
	@echo "Building $(AGENT_NAME)..."
	$(GOBUILD) -ldflags "-X main.Version=$(VERSION)" -o bin/$(AGENT_NAME) ./cmd/rexec-agent

ssh-gateway:
	@echo "Building $(SSH_NAME)..."
	$(GOBUILD) -ldflags "-X main.Version=$(VERSION)" -o bin/$(SSH_NAME) ./cmd/rexec-ssh

# Build all CLI tools
cli-all: cli tui agent ssh-gateway
	@echo "All CLI tools built!"

# Build multi-arch binaries for distribution
PLATFORMS = linux-amd64 linux-arm64 darwin-amd64 darwin-arm64
DOWNLOADS_DIR = downloads

downloads-dir:
	@mkdir -p $(DOWNLOADS_DIR)

# Build agent for all platforms
agent-all: downloads-dir
	@echo "Building agent for all platforms..."
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d- -f1); \
		arch=$$(echo $$platform | cut -d- -f2); \
		echo "  Building rexec-agent-$$platform..."; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GOBUILD) -ldflags "-X main.Version=$(VERSION) -s -w" \
			-o $(DOWNLOADS_DIR)/rexec-agent-$$platform ./cmd/rexec-agent; \
	done
	@echo "Agent binaries built in $(DOWNLOADS_DIR)/"
	@ls -la $(DOWNLOADS_DIR)/rexec-agent-*

# Build CLI for all platforms
cli-all-platforms: downloads-dir
	@echo "Building CLI for all platforms..."
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d- -f1); \
		arch=$$(echo $$platform | cut -d- -f2); \
		echo "  Building rexec-cli-$$platform..."; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GOBUILD) -ldflags "-X main.Version=$(VERSION) -s -w" \
			-o $(DOWNLOADS_DIR)/rexec-cli-$$platform ./cmd/rexec-cli; \
	done
	@echo "CLI binaries built in $(DOWNLOADS_DIR)/"
	@ls -la $(DOWNLOADS_DIR)/rexec-cli-*

# Build TUI for all platforms
tui-all-platforms: downloads-dir
	@echo "Building TUI for all platforms..."
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d- -f1); \
		arch=$$(echo $$platform | cut -d- -f2); \
		echo "  Building rexec-tui-$$platform..."; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GOBUILD) -ldflags "-X main.Version=$(VERSION) -s -w" \
			-o $(DOWNLOADS_DIR)/rexec-tui-$$platform ./cmd/rexec-tui; \
	done
	@echo "TUI binaries built in $(DOWNLOADS_DIR)/"
	@ls -la $(DOWNLOADS_DIR)/rexec-tui-*

# Build SSH gateway for all platforms
ssh-gateway-all: downloads-dir
	@echo "Building SSH gateway for all platforms..."
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d- -f1); \
		arch=$$(echo $$platform | cut -d- -f2); \
		echo "  Building rexec-ssh-$$platform..."; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GOBUILD) -ldflags "-X main.Version=$(VERSION) -s -w" \
			-o $(DOWNLOADS_DIR)/rexec-ssh-$$platform ./cmd/rexec-ssh; \
	done
	@echo "SSH gateway binaries built in $(DOWNLOADS_DIR)/"
	@ls -la $(DOWNLOADS_DIR)/rexec-ssh-*

# Build all distribution binaries
dist: agent-all cli-all-platforms tui-all-platforms ssh-gateway-all
	@echo ""
	@echo "All distribution binaries built!"
	@echo "Contents of $(DOWNLOADS_DIR)/:"
	@ls -la $(DOWNLOADS_DIR)/

# Build everything (UI + Go binary + CLI tools + embed widget)
build-all: ui embed build cli-all
	@echo "Build complete!"

# Run the application
run: build
	@echo "Starting $(BINARY_NAME)..."
	./bin/$(BINARY_NAME) server

# Run the application with fresh UI build
run-all: build-all
	@echo "Starting $(BINARY_NAME)..."
	./bin/$(BINARY_NAME) server

# Run in development mode with hot reload (requires air)
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf tmp/

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Build Docker image for API
docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .

# Run with Docker Compose
docker-run:
	cd docker && docker-compose up -d

# Stop Docker Compose
docker-stop:
	cd docker && docker-compose down

# Build base container images for users
images:
	@echo "Building Ubuntu image..."
	docker build -t rexec-ubuntu:latest docker/images/ubuntu/
	@echo "Building Debian image..."
	docker build -t rexec-debian:latest docker/images/debian/
	@echo "Building Arch image..."
	docker build -t rexec-arch:latest docker/images/arch/
	@echo "All images built successfully!"

# Pull base images
pull-images:
	docker pull ubuntu:22.04
	docker pull debian:bookworm
	docker pull archlinux:latest

# Setup development environment
setup: deps pull-images
	@echo "Creating volume directory..."
	sudo mkdir -p /var/lib/rexec/volumes
	sudo chown -R $(USER):$(USER) /var/lib/rexec
	@echo "Copying .env file..."
	cp -n .env.example .env 2>/dev/null || true
	@echo "Setup complete!"

# Lint the code
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

# Format the code
fmt:
	$(GOCMD) fmt ./...

# Generate API documentation (if using swagger)
docs:
	@which swag > /dev/null || (echo "Installing swag..." && go install github.com/swaggo/swag/cmd/swag@latest)
	swag init -g cmd/rexec/main.go

# Show logs from docker-compose
logs:
	cd docker && docker-compose logs -f

# Help
help:
	@echo "Rexec - Terminal as a Service"
	@echo ""
	@echo "Usage:"
	@echo "  make build        - Build the Go server binary"
	@echo "  make build-all    - Build UI + server + CLI tools"
	@echo "  make run          - Build and run the application"
	@echo "  make run-all      - Build UI + Go and run"
	@echo "  make dev          - Run with hot reload (requires air)"
	@echo ""
	@echo "  make cli          - Build the rexec-cli tool"
	@echo "  make tui          - Build the rexec-tui dashboard"
	@echo "  make agent        - Build the rexec-agent"
	@echo "  make ssh-gateway  - Build the rexec-ssh gateway"
	@echo "  make cli-all      - Build all CLI tools (local platform)"
	@echo ""
	@echo "  make agent-all          - Build agent for all platforms (linux/darwin amd64/arm64)"
	@echo "  make cli-all-platforms  - Build CLI for all platforms"
	@echo "  make tui-all-platforms  - Build TUI for all platforms"
	@echo ""
	@echo "Firecracker:"
	@echo "  make firecracker-guest-agent  - Build guest agent binary"
	@echo "  make firecracker-rootfs        - Build rootfs (DISTRO=ubuntu VERSION=24.04)"
	@echo "  make firecracker-setup         - Setup Firecracker environment"
	@echo "  make ssh-gateway-all    - Build SSH gateway for all platforms"
	@echo "  make dist               - Build all binaries for all platforms (for distribution)"
	@echo ""
	@echo "  make ui           - Build the Svelte web UI"
	@echo "  make ui-dev       - Run web UI dev server (port 3000)"
	@echo "  make ui-install   - Install web UI dependencies"
	@echo ""
	@echo "  make embed        - Build the embeddable terminal widget"
	@echo "  make embed-dev    - Build embed widget with watch mode"
	@echo "  make embed-install - Install embed widget dependencies"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make deps         - Download dependencies"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run with Docker Compose"
	@echo "  make docker-stop  - Stop Docker Compose"
	@echo "  make images       - Build base container images (ubuntu, debian, arch)"
	@echo "  make setup        - Setup development environment"
	@echo "  make lint         - Lint the code"
	@echo "  make fmt          - Format the code"
	@echo "  make logs         - Show docker-compose logs"
	@echo "  make help         - Show this help message"

# Security scan
security:
	@echo "Scanning for security vulnerabilities..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found, installing..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi
	@echo "Checking for known vulnerabilities..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not found, installing..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		govulncheck ./...; \
	fi

# Firecracker targets
firecracker-guest-agent:
	@echo "Building Firecracker guest agent..."
	@mkdir -p bin
	cd cmd/rexec-guest-agent && go build -o ../../bin/rexec-guest-agent .
	@echo "✅ Guest agent built: bin/rexec-guest-agent"

firecracker-rootfs:
	@if [ -z "$(DISTRO)" ]; then \
		echo "❌ Usage: make firecracker-rootfs DISTRO=ubuntu VERSION=24.04"; \
		exit 1; \
	fi
	@echo "Building rootfs image for $(DISTRO) $(VERSION)..."
	@./scripts/build-rootfs.sh $(DISTRO) $(VERSION) $(SIZE)
	@echo "✅ Rootfs image built"

firecracker-setup: firecracker-guest-agent
	@echo "Setting up Firecracker..."
	@echo "1. Preparing kernel..."
	@./scripts/prepare-kernel.sh || echo "⚠️  Kernel preparation failed - see docs/FIRECRACKER_SETUP.md"
	@echo "2. Building default rootfs images..."
	@echo "   (Run 'make firecracker-rootfs DISTRO=ubuntu VERSION=24.04' to build images)"
	@echo "✅ Firecracker setup complete"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Build rootfs: make firecracker-rootfs DISTRO=ubuntu VERSION=24.04"
	@echo "  2. Set environment variables (see docs/FIRECRACKER_SETUP.md)"
	@echo "  3. Start server: make run"
