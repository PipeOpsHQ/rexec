#!/bin/bash
set -e

# Rexec Custom Image Builder
# Builds Docker images with SSH server and dev tools pre-installed
# Compatible with bash 3.2+ (macOS default)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
IMAGES_DIR="$PROJECT_DIR/docker/images"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Image list (space-separated: dir:tag)
IMAGES="ubuntu:rexec-ubuntu debian:rexec-debian alpine:rexec-alpine fedora:rexec-fedora"

get_tag_for_dir() {
    local dir="$1"
    for pair in $IMAGES; do
        local d="${pair%%:*}"
        local t="${pair##*:}"
        if [ "$d" = "$dir" ]; then
            echo "$t"
            return
        fi
    done
    echo ""
}

create_entrypoint() {
    local image="$1"
    local entrypoint="$IMAGES_DIR/$image/entrypoint.sh"

    cat > "$entrypoint" << 'ENTRYPOINT_EOF'
#!/bin/bash
set -e

# Start SSH server if not running
if command -v sshd &> /dev/null; then
    if [ ! -f /var/run/sshd.pid ] || ! kill -0 $(cat /var/run/sshd.pid 2>/dev/null) 2>/dev/null; then
        # Ensure SSH host keys exist
        if [ ! -f /etc/ssh/ssh_host_rsa_key ]; then
            ssh-keygen -A 2>/dev/null || true
        fi

        # Start SSH daemon
        /usr/sbin/sshd 2>/dev/null || true
    fi
fi

# If running as root, switch to user for the main process
if [ "$(id -u)" = "0" ]; then
    if [ $# -eq 0 ]; then
        exec su - user
    else
        exec su - user -c "$*"
    fi
else
    if [ $# -eq 0 ]; then
        exec /bin/bash
    else
        exec "$@"
    fi
fi
ENTRYPOINT_EOF

    chmod +x "$entrypoint"
    log_info "Created entrypoint.sh for $image"
}

create_dockerfile() {
    local image="$1"
    local dockerfile="$IMAGES_DIR/$image/Dockerfile"

    mkdir -p "$IMAGES_DIR/$image"

    case "$image" in
        ubuntu)
            cat > "$dockerfile" << 'DOCKERFILE_EOF'
FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive
ENV TERM=xterm-256color
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

# Install packages including SSH server
RUN apt-get update && apt-get install -y \
    bash curl wget git vim nano htop tree unzip zip sudo \
    openssh-server openssh-client \
    ca-certificates locales build-essential \
    python3 python3-pip nodejs npm \
    net-tools iputils-ping dnsutils procps less man-db jq \
    && rm -rf /var/lib/apt/lists/* \
    && locale-gen en_US.UTF-8

# Configure SSH
RUN mkdir -p /var/run/sshd \
    && sed -i 's/#PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config \
    && sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config \
    && sed -i 's/#PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config \
    && ssh-keygen -A

# Create user
RUN useradd -m -s /bin/bash -u 1000 user \
    && echo "user ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers \
    && mkdir -p /home/user/.ssh \
    && chmod 700 /home/user/.ssh \
    && touch /home/user/.ssh/authorized_keys \
    && chmod 600 /home/user/.ssh/authorized_keys \
    && chown -R user:user /home/user

# Configure bash
RUN echo 'export PS1="\[\033[01;32m\]\u@rexec\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ "' >> /home/user/.bashrc \
    && echo 'alias ll="ls -la"' >> /home/user/.bashrc \
    && echo 'alias la="ls -A"' >> /home/user/.bashrc

WORKDIR /home/user
EXPOSE 22

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["/bin/bash"]
DOCKERFILE_EOF
            create_entrypoint "$image"
            ;;

        debian)
            cat > "$dockerfile" << 'DOCKERFILE_EOF'
FROM debian:bookworm

ENV DEBIAN_FRONTEND=noninteractive
ENV TERM=xterm-256color
ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

# Install packages including SSH server
RUN apt-get update && apt-get install -y \
    bash curl wget git vim nano htop tree unzip zip sudo \
    openssh-server openssh-client \
    ca-certificates gnupg lsb-release build-essential \
    python3 python3-pip nodejs npm \
    net-tools iputils-ping dnsutils procps less man-db jq \
    && rm -rf /var/lib/apt/lists/*

# Configure SSH
RUN mkdir -p /var/run/sshd \
    && sed -i 's/#PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config \
    && sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config \
    && sed -i 's/#PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config \
    && ssh-keygen -A

# Create user
RUN useradd -m -s /bin/bash -G sudo user \
    && echo "user ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers \
    && mkdir -p /home/user/.ssh \
    && chmod 700 /home/user/.ssh \
    && touch /home/user/.ssh/authorized_keys \
    && chmod 600 /home/user/.ssh/authorized_keys \
    && chown -R user:user /home/user

# Configure bash
RUN echo 'export PS1="\[\033[01;32m\]\u@rexec\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ "' >> /home/user/.bashrc \
    && echo 'alias ll="ls -la"' >> /home/user/.bashrc

WORKDIR /home/user
EXPOSE 22

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["/bin/bash"]
DOCKERFILE_EOF
            create_entrypoint "$image"
            ;;

        alpine)
            cat > "$dockerfile" << 'DOCKERFILE_EOF'
FROM alpine:latest

ENV TERM=xterm-256color
ENV LANG=C.UTF-8

# Install packages including SSH server
RUN apk add --no-cache \
    bash curl wget git vim nano htop tree unzip zip sudo shadow \
    openssh openssh-server \
    ca-certificates build-base \
    python3 py3-pip nodejs npm \
    net-tools bind-tools procps less mandoc jq

# Configure SSH
RUN mkdir -p /var/run/sshd \
    && ssh-keygen -A \
    && sed -i 's/#PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config \
    && sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config \
    && sed -i 's/#PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config

# Create user
RUN adduser -D -s /bin/bash -u 1000 user \
    && echo "user ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers \
    && mkdir -p /home/user/.ssh \
    && chmod 700 /home/user/.ssh \
    && touch /home/user/.ssh/authorized_keys \
    && chmod 600 /home/user/.ssh/authorized_keys \
    && chown -R user:user /home/user

# Configure bash
RUN echo 'export PS1="\[\033[01;36m\]\u@rexec-alpine\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ "' >> /home/user/.bashrc \
    && echo 'alias ll="ls -la"' >> /home/user/.bashrc

WORKDIR /home/user
EXPOSE 22

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["/bin/bash"]
DOCKERFILE_EOF
            create_entrypoint "$image"
            ;;

        fedora)
            cat > "$dockerfile" << 'DOCKERFILE_EOF'
FROM fedora:latest

ENV TERM=xterm-256color
ENV LANG=en_US.UTF-8

# Install packages including SSH server
RUN dnf install -y \
    bash curl wget git vim nano htop tree unzip zip sudo \
    openssh-server openssh-clients \
    ca-certificates gcc gcc-c++ make \
    python3 python3-pip nodejs npm \
    net-tools bind-utils procps-ng less man-db jq \
    && dnf clean all

# Configure SSH
RUN mkdir -p /var/run/sshd \
    && ssh-keygen -A \
    && sed -i 's/#PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config \
    && sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config \
    && sed -i 's/#PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config

# Create user
RUN useradd -m -s /bin/bash -u 1000 user \
    && echo "user ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers \
    && mkdir -p /home/user/.ssh \
    && chmod 700 /home/user/.ssh \
    && touch /home/user/.ssh/authorized_keys \
    && chmod 600 /home/user/.ssh/authorized_keys \
    && chown -R user:user /home/user

# Configure bash
RUN echo 'export PS1="\[\033[01;35m\]\u@rexec-fedora\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ "' >> /home/user/.bashrc \
    && echo 'alias ll="ls -la"' >> /home/user/.bashrc

WORKDIR /home/user
EXPOSE 22

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["/bin/bash"]
DOCKERFILE_EOF
            create_entrypoint "$image"
            ;;
        *)
            log_error "Unknown image type: $image"
            return 1
            ;;
    esac

    log_info "Created Dockerfile for $image"
}

build_image() {
    local dir="$1"
    local tag="$2"

    if [ ! -d "$IMAGES_DIR/$dir" ]; then
        log_warn "Directory $IMAGES_DIR/$dir not found, creating..."
        mkdir -p "$IMAGES_DIR/$dir"
        create_dockerfile "$dir"
    fi

    if [ ! -f "$IMAGES_DIR/$dir/Dockerfile" ]; then
        log_warn "Dockerfile not found in $IMAGES_DIR/$dir, creating..."
        create_dockerfile "$dir"
    fi

    log_step "Building $tag from $IMAGES_DIR/$dir..."

    if docker build -t "$tag:latest" "$IMAGES_DIR/$dir"; then
        log_info "Successfully built $tag"
        return 0
    else
        log_error "Failed to build $tag"
        return 1
    fi
}

build_all() {
    log_info "Building all rexec images..."
    echo ""

    local failed=0

    for pair in $IMAGES; do
        local dir="${pair%%:*}"
        local tag="${pair##*:}"
        if ! build_image "$dir" "$tag"; then
            failed=$((failed + 1))
        fi
        echo ""
    done

    echo "=========================================="
    if [ $failed -eq 0 ]; then
        log_info "All images built successfully!"
    else
        log_warn "$failed image(s) failed to build"
    fi

    echo ""
    log_info "Built images:"
    docker images | grep rexec- | head -10
}

build_single() {
    local image="$1"
    local tag=$(get_tag_for_dir "$image")

    if [ -z "$tag" ]; then
        log_error "Unknown image: $image"
        echo "Available images: ubuntu debian alpine fedora"
        exit 1
    fi

    build_image "$image" "$tag"
}

usage() {
    echo "Rexec Image Builder"
    echo ""
    echo "Usage: $0 [command] [image]"
    echo ""
    echo "Commands:"
    echo "  all             Build all images (default)"
    echo "  build <image>   Build a specific image"
    echo "  list            List available images"
    echo "  clean           Remove all rexec images"
    echo ""
    echo "Available images: ubuntu debian alpine fedora"
    echo ""
    echo "Examples:"
    echo "  $0                  # Build all images"
    echo "  $0 build ubuntu     # Build only ubuntu"
    echo "  $0 list             # List available images"
}

# Main
case "${1:-all}" in
    all)
        build_all
        ;;
    build)
        if [ -z "$2" ]; then
            log_error "Please specify an image to build"
            usage
            exit 1
        fi
        build_single "$2"
        ;;
    list)
        echo "Available images:"
        for pair in $IMAGES; do
            dir="${pair%%:*}"
            tag="${pair##*:}"
            echo "  $dir -> $tag"
        done
        ;;
    clean)
        log_info "Removing all rexec images..."
        docker images | grep rexec- | awk '{print $3}' | xargs -r docker rmi -f 2>/dev/null || true
        log_info "Done"
        ;;
    -h|--help|help)
        usage
        ;;
    *)
        log_error "Unknown command: $1"
        usage
        exit 1
        ;;
esac
