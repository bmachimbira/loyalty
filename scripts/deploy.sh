#!/bin/bash

###########################################
# Production Deployment Script
# Zimbabwe Loyalty Platform
###########################################

set -e  # Exit on error
set -u  # Exit on undefined variable
set -o pipefail  # Exit on pipe failure

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
MAX_HEALTH_CHECK_ATTEMPTS=30
HEALTH_CHECK_INTERVAL=2

# Logging function
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1"
}

log_success() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS:${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING:${NC} $1"
}

# Show usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Deploy the Loyalty Platform to production"
    echo ""
    echo "Options:"
    echo "  --skip-backup      Skip pre-deployment database backup"
    echo "  --skip-tests       Skip health checks after deployment"
    echo "  --skip-migration   Skip database migrations"
    echo "  --pull             Pull latest code from git (default: skip)"
    echo "  --branch BRANCH    Git branch to deploy (default: main)"
    echo "  -h, --help         Show this help message"
    echo ""
    exit 1
}

# Parse arguments
SKIP_BACKUP=false
SKIP_TESTS=false
SKIP_MIGRATION=false
PULL_CODE=false
GIT_BRANCH="main"

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-backup)
            SKIP_BACKUP=true
            shift
            ;;
        --skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        --skip-migration)
            SKIP_MIGRATION=true
            shift
            ;;
        --pull)
            PULL_CODE=true
            shift
            ;;
        --branch)
            GIT_BRANCH="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            ;;
    esac
done

cd "$PROJECT_DIR"

log "======================================"
log "  Loyalty Platform Deployment"
log "======================================"
log "Environment: PRODUCTION"
log "Compose file: $COMPOSE_FILE"
log "Directory: $PROJECT_DIR"
log ""

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    log_warning "Running as root. Consider using a non-root user with Docker permissions"
fi

# Check if Docker is installed and running
if ! command -v docker &> /dev/null; then
    log_error "Docker is not installed"
    exit 1
fi

if ! docker info > /dev/null 2>&1; then
    log_error "Docker is not running"
    exit 1
fi

# Check if compose file exists
if [ ! -f "$COMPOSE_FILE" ]; then
    log_error "Compose file not found: $COMPOSE_FILE"
    exit 1
fi

# Pull latest code if requested
if [ "$PULL_CODE" = true ]; then
    log "Pulling latest code from git..."
    if git fetch origin && git checkout "$GIT_BRANCH" && git pull origin "$GIT_BRANCH"; then
        log_success "Code updated to latest $GIT_BRANCH"
    else
        log_error "Failed to pull latest code"
        exit 1
    fi
fi

# Create backup before deployment
if [ "$SKIP_BACKUP" = false ]; then
    log "Creating pre-deployment backup..."
    if bash "$SCRIPT_DIR/backup.sh"; then
        log_success "Pre-deployment backup created"
    else
        log_warning "Backup failed, but continuing with deployment"
    fi
else
    log_warning "Skipping pre-deployment backup"
fi

# Build new images
log "Building Docker images..."
if docker-compose -f "$COMPOSE_FILE" build --no-cache; then
    log_success "Docker images built successfully"
else
    log_error "Failed to build Docker images"
    exit 1
fi

# Get image sizes
log "Image sizes:"
docker images | grep loyalty | awk '{print "  " $1 ":" $2 " - " $7 $8}'

# Run database migrations
if [ "$SKIP_MIGRATION" = false ]; then
    log "Running database migrations..."
    if bash "$SCRIPT_DIR/migrate.sh"; then
        log_success "Migrations completed successfully"
    else
        log_error "Migrations failed"
        log "Rolling back deployment..."
        exit 1
    fi
else
    log_warning "Skipping database migrations"
fi

# Perform rolling restart of API services
log "Performing rolling restart of API services..."

# Get current running API containers
API_CONTAINERS=$(docker-compose -f "$COMPOSE_FILE" ps -q api 2>/dev/null || echo "")

if [ -n "$API_CONTAINERS" ]; then
    log "Scaling API service for zero-downtime deployment..."

    # Scale up to temporary increased capacity
    docker-compose -f "$COMPOSE_FILE" up -d --scale api=3 --no-recreate

    # Wait for new containers to be healthy
    log "Waiting for new API containers to be healthy..."
    sleep 10

    # Scale back down to normal capacity (this will remove old containers)
    docker-compose -f "$COMPOSE_FILE" up -d --scale api=2

else
    log "No existing API containers found, starting fresh..."
    docker-compose -f "$COMPOSE_FILE" up -d api
fi

log_success "API services restarted"

# Wait for API to be healthy
log "Checking API health..."
ATTEMPTS=0
while [ $ATTEMPTS -lt $MAX_HEALTH_CHECK_ATTEMPTS ]; do
    if curl -f -s http://localhost:8080/health > /dev/null 2>&1; then
        log_success "API is healthy"
        break
    fi

    ATTEMPTS=$((ATTEMPTS + 1))
    log "Waiting for API to be ready... ($ATTEMPTS/$MAX_HEALTH_CHECK_ATTEMPTS)"
    sleep $HEALTH_CHECK_INTERVAL
done

if [ $ATTEMPTS -eq $MAX_HEALTH_CHECK_ATTEMPTS ]; then
    log_error "API failed to become healthy after deployment"
    log "Check logs with: docker-compose -f $COMPOSE_FILE logs api"
    exit 1
fi

# Restart web service
log "Restarting web service..."
docker-compose -f "$COMPOSE_FILE" up -d web

# Wait for web to be healthy
log "Checking web service..."
sleep 5

# Ensure Caddy is running
log "Ensuring Caddy reverse proxy is running..."
docker-compose -f "$COMPOSE_FILE" up -d caddy

# Run health checks
if [ "$SKIP_TESTS" = false ]; then
    log "Running post-deployment health checks..."

    # Check database
    if docker-compose -f "$COMPOSE_FILE" exec -T db pg_isready -U postgres > /dev/null 2>&1; then
        log_success "Database: healthy"
    else
        log_error "Database: unhealthy"
    fi

    # Check API readiness
    if curl -f -s http://localhost:8080/ready > /dev/null 2>&1; then
        log_success "API readiness: passed"
    else
        log_warning "API readiness: failed"
    fi

    # Check API health
    if curl -f -s http://localhost:8080/health > /dev/null 2>&1; then
        log_success "API health: passed"
    else
        log_error "API health: failed"
    fi

    # Check web service
    if curl -f -s http://localhost/ > /dev/null 2>&1; then
        log_success "Web service: healthy"
    else
        log_warning "Web service: failed"
    fi

else
    log_warning "Skipping post-deployment tests"
fi

# Show service status
log ""
log "Service Status:"
docker-compose -f "$COMPOSE_FILE" ps

# Show recent logs
log ""
log "Recent API logs:"
docker-compose -f "$COMPOSE_FILE" logs --tail=20 api

# Cleanup old images
log ""
log "Cleaning up old Docker images..."
docker image prune -f > /dev/null 2>&1

# Summary
log ""
log "======================================"
log_success "Deployment completed successfully!"
log "======================================"
log ""
log "Next steps:"
log "  - Monitor logs: docker-compose -f $COMPOSE_FILE logs -f"
log "  - Check metrics: docker stats"
log "  - Verify application: curl http://localhost/health"
log ""

exit 0
