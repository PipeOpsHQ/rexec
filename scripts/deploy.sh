#!/bin/bash
set -e

# Rexec Production Deployment Script
# Usage: ./deploy.sh [init|start|stop|restart|logs|ssl|backup]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
DOCKER_DIR="$PROJECT_DIR/docker"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_env() {
    if [ ! -f "$DOCKER_DIR/.env" ]; then
        log_error ".env file not found. Copy .env.example to .env and configure it."
        echo "  cp $DOCKER_DIR/.env.example $DOCKER_DIR/.env"
        exit 1
    fi

    # Source env file
    set -a
    source "$DOCKER_DIR/.env"
    set +a

    # Check required variables
    local missing=()
    [ -z "$JWT_SECRET" ] && missing+=("JWT_SECRET")
    [ -z "$POSTGRES_PASSWORD" ] && missing+=("POSTGRES_PASSWORD")
    [ -z "$STRIPE_SECRET_KEY" ] && missing+=("STRIPE_SECRET_KEY")
    [ -z "$STRIPE_WEBHOOK_SECRET" ] && missing+=("STRIPE_WEBHOOK_SECRET")
    [ -z "$DOMAIN" ] && missing+=("DOMAIN")

    if [ ${#missing[@]} -gt 0 ]; then
        log_error "Missing required environment variables:"
        for var in "${missing[@]}"; do
            echo "  - $var"
        done
        exit 1
    fi
}

init() {
    log_info "Initializing Rexec production environment..."

    check_env

    # Create required directories
    mkdir -p "$DOCKER_DIR/nginx/conf.d"
    mkdir -p "$DOCKER_DIR/init-db"
    mkdir -p "$PROJECT_DIR/backups"

    # Update nginx config with actual domain
    if [ -f "$DOCKER_DIR/nginx/conf.d/rexec.conf" ]; then
        sed -i.bak "s/server_name _;/server_name $DOMAIN;/" "$DOCKER_DIR/nginx/conf.d/rexec.conf"
        sed -i.bak "s|/etc/letsencrypt/live/rexec/|/etc/letsencrypt/live/$DOMAIN/|g" "$DOCKER_DIR/nginx/conf.d/rexec.conf"
        rm -f "$DOCKER_DIR/nginx/conf.d/rexec.conf.bak"
    fi

    # Build Docker images
    log_info "Building Docker images..."
    cd "$DOCKER_DIR"
    docker compose -f docker-compose.prod.yml build

    log_info "Initialization complete!"
    echo ""
    echo "Next steps:"
    echo "  1. Run './deploy.sh ssl' to obtain SSL certificates"
    echo "  2. Run './deploy.sh start' to start the services"
}

ssl() {
    log_info "Obtaining SSL certificate from Let's Encrypt..."

    check_env

    cd "$DOCKER_DIR"

    # Start nginx temporarily for ACME challenge
    log_info "Starting nginx for certificate validation..."

    # Create temporary nginx config without SSL
    cat > "$DOCKER_DIR/nginx/conf.d/temp-http.conf" << 'EOF'
server {
    listen 80;
    server_name _;

    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    location / {
        return 200 'Rexec is starting...';
        add_header Content-Type text/plain;
    }
}
EOF

    # Temporarily remove SSL config
    mv "$DOCKER_DIR/nginx/conf.d/rexec.conf" "$DOCKER_DIR/nginx/conf.d/rexec.conf.disabled" 2>/dev/null || true

    # Start nginx
    docker compose -f docker-compose.prod.yml up -d nginx

    sleep 5

    # Request certificate
    log_info "Requesting certificate for $DOMAIN..."
    docker compose -f docker-compose.prod.yml run --rm certbot certonly \
        --webroot \
        --webroot-path=/var/www/certbot \
        --email "${LETSENCRYPT_EMAIL:-admin@$DOMAIN}" \
        --agree-tos \
        --no-eff-email \
        -d "$DOMAIN"

    # Restore SSL config
    rm -f "$DOCKER_DIR/nginx/conf.d/temp-http.conf"
    mv "$DOCKER_DIR/nginx/conf.d/rexec.conf.disabled" "$DOCKER_DIR/nginx/conf.d/rexec.conf" 2>/dev/null || true

    # Restart nginx with SSL
    docker compose -f docker-compose.prod.yml restart nginx

    log_info "SSL certificate obtained successfully!"
}

start() {
    log_info "Starting Rexec services..."

    check_env

    cd "$DOCKER_DIR"
    docker compose -f docker-compose.prod.yml up -d

    log_info "Services started!"
    echo ""
    echo "Check status with: ./deploy.sh logs"
    echo "Access your app at: https://$DOMAIN"
}

stop() {
    log_info "Stopping Rexec services..."

    cd "$DOCKER_DIR"
    docker compose -f docker-compose.prod.yml down

    log_info "Services stopped."
}

restart() {
    log_info "Restarting Rexec services..."

    stop
    start
}

logs() {
    local service="${1:-}"

    cd "$DOCKER_DIR"

    if [ -n "$service" ]; then
        docker compose -f docker-compose.prod.yml logs -f "$service"
    else
        docker compose -f docker-compose.prod.yml logs -f
    fi
}

backup() {
    log_info "Creating backup..."

    check_env

    local backup_dir="$PROJECT_DIR/backups"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="$backup_dir/rexec_backup_$timestamp.sql"

    cd "$DOCKER_DIR"

    # Backup PostgreSQL
    log_info "Backing up PostgreSQL database..."
    docker compose -f docker-compose.prod.yml exec -T postgres \
        pg_dump -U rexec rexec > "$backup_file"

    # Compress
    gzip "$backup_file"

    log_info "Backup created: ${backup_file}.gz"

    # Keep only last 7 backups
    ls -t "$backup_dir"/rexec_backup_*.sql.gz 2>/dev/null | tail -n +8 | xargs -r rm

    log_info "Backup complete!"
}

status() {
    cd "$DOCKER_DIR"
    docker compose -f docker-compose.prod.yml ps
}

update() {
    log_info "Updating Rexec..."

    cd "$PROJECT_DIR"

    # Pull latest code (if using git)
    if [ -d ".git" ]; then
        log_info "Pulling latest code..."
        git pull
    fi

    # Rebuild and restart
    cd "$DOCKER_DIR"
    log_info "Rebuilding containers..."
    docker compose -f docker-compose.prod.yml build

    log_info "Restarting services..."
    docker compose -f docker-compose.prod.yml up -d

    log_info "Update complete!"
}

usage() {
    echo "Rexec Deployment Script"
    echo ""
    echo "Usage: $0 <command> [options]"
    echo ""
    echo "Commands:"
    echo "  init      Initialize production environment"
    echo "  ssl       Obtain SSL certificate from Let's Encrypt"
    echo "  start     Start all services"
    echo "  stop      Stop all services"
    echo "  restart   Restart all services"
    echo "  logs      View logs (optionally specify service name)"
    echo "  status    Show service status"
    echo "  backup    Backup PostgreSQL database"
    echo "  update    Pull latest code and rebuild"
    echo ""
    echo "Examples:"
    echo "  $0 init"
    echo "  $0 start"
    echo "  $0 logs rexec-api"
    echo "  $0 backup"
}

# Main
case "${1:-}" in
    init)
        init
        ;;
    ssl)
        ssl
        ;;
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    logs)
        logs "${2:-}"
        ;;
    status)
        status
        ;;
    backup)
        backup
        ;;
    update)
        update
        ;;
    *)
        usage
        exit 1
        ;;
esac
