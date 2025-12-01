.PHONY: build run dev clean test docker-build docker-run help images

# Variables
BINARY_NAME=rexec
DOCKER_IMAGE=rexec-api
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -ldflags "-X main.Version=$(VERSION)" -o bin/$(BINARY_NAME) ./cmd/rexec

# Run the application
run: build
	@echo "Starting $(BINARY_NAME)..."
	./bin/$(BINARY_NAME)

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
	@echo "  make build        - Build the binary"
	@echo "  make run          - Build and run the application"
	@echo "  make dev          - Run with hot reload (requires air)"
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
