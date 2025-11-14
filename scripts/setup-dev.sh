#!/bin/bash

###########################################
# Development Environment Setup Script
# Zimbabwe Loyalty Platform
###########################################

set -e
set -u
set -o pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Logging
log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $1"
}

log_error() {
    echo -e "${RED}[$(date '+%H:%M:%S')] ERROR:${NC} $1"
}

log_success() {
    echo -e "${GREEN}[$(date '+%H:%M:%S')] SUCCESS:${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[$(date '+%H:%M:%S')] WARNING:${NC} $1"
}

cd "$PROJECT_DIR"

log "======================================"
log "  Development Environment Setup"
log "======================================"
log "Project: Zimbabwe Loyalty Platform"
log "Directory: $PROJECT_DIR"
echo ""

# Check prerequisites
log "Checking prerequisites..."

# Check Docker
if ! command -v docker &> /dev/null; then
    log_error "Docker is not installed"
    log "Install Docker from: https://docs.docker.com/get-docker/"
    exit 1
fi
log_success "✓ Docker installed"

# Check Docker Compose
if ! command -v docker-compose &> /dev/null; then
    log_error "Docker Compose is not installed"
    log "Install Docker Compose from: https://docs.docker.com/compose/install/"
    exit 1
fi
log_success "✓ Docker Compose installed"

# Check Docker is running
if ! docker info > /dev/null 2>&1; then
    log_error "Docker is not running"
    log "Start Docker Desktop or the Docker daemon"
    exit 1
fi
log_success "✓ Docker is running"

# Check Go
if ! command -v go &> /dev/null; then
    log_warning "✗ Go is not installed (optional for local development)"
    log "  Install from: https://golang.org/dl/"
else
    GO_VERSION=$(go version | awk '{print $3}')
    log_success "✓ Go installed ($GO_VERSION)"
fi

# Check Node.js
if ! command -v node &> /dev/null; then
    log_warning "✗ Node.js is not installed (optional for local development)"
    log "  Install from: https://nodejs.org/"
else
    NODE_VERSION=$(node --version)
    log_success "✓ Node.js installed ($NODE_VERSION)"
fi

echo ""

# Create .env file if it doesn't exist
if [ ! -f "$PROJECT_DIR/.env" ]; then
    log "Creating .env file from .env.example..."
    cp "$PROJECT_DIR/.env.example" "$PROJECT_DIR/.env"
    log_success "✓ .env file created"
    log_warning "⚠ Please review and update .env file with your configuration"
else
    log "✓ .env file already exists"
fi

# Create directories
log ""
log "Creating required directories..."
mkdir -p "$PROJECT_DIR/backups"
mkdir -p "$PROJECT_DIR/logs"
log_success "✓ Directories created"

# Pull Docker images
log ""
log "Pulling Docker images..."
if docker-compose pull; then
    log_success "✓ Docker images pulled"
else
    log_warning "✗ Failed to pull some images (will build locally)"
fi

# Build images
log ""
log "Building Docker images..."
if docker-compose build; then
    log_success "✓ Docker images built"
else
    log_error "Failed to build Docker images"
    exit 1
fi

# Start database
log ""
log "Starting database..."
if docker-compose up -d db; then
    log_success "✓ Database started"
else
    log_error "Failed to start database"
    exit 1
fi

# Wait for database to be ready
log ""
log "Waiting for database to be ready..."
ATTEMPTS=0
MAX_ATTEMPTS=30

while [ $ATTEMPTS -lt $MAX_ATTEMPTS ]; do
    if docker-compose exec -T db pg_isready -U postgres > /dev/null 2>&1; then
        log_success "✓ Database is ready"
        break
    fi

    ATTEMPTS=$((ATTEMPTS + 1))
    echo -n "."
    sleep 1
done

if [ $ATTEMPTS -eq $MAX_ATTEMPTS ]; then
    log_error "Database failed to start"
    exit 1
fi

# Run migrations
log ""
log "Running database migrations..."
# Check if migrations directory exists
if [ -d "$PROJECT_DIR/migrations" ]; then
    if bash "$SCRIPT_DIR/migrate.sh"; then
        log_success "✓ Migrations completed"
    else
        log_warning "✗ Migrations failed (you may need to run them manually)"
    fi
else
    log_warning "No migrations directory found"
fi

# Start all services
log ""
log "Starting all services..."
if docker-compose up -d; then
    log_success "✓ All services started"
else
    log_error "Failed to start services"
    exit 1
fi

# Wait for API to be ready
log ""
log "Waiting for API to be ready..."
sleep 5

ATTEMPTS=0
while [ $ATTEMPTS -lt 20 ]; do
    if curl -f -s http://localhost:8080/health > /dev/null 2>&1; then
        log_success "✓ API is ready"
        break
    fi

    ATTEMPTS=$((ATTEMPTS + 1))
    echo -n "."
    sleep 1
done

# Show service status
log ""
log "Service Status:"
docker-compose ps

# Summary
log ""
log "======================================"
log_success "  Setup Complete!"
log "======================================"
echo ""
log "Services are running:"
log "  - Database: localhost:5432"
log "  - API: http://localhost:8080"
log "  - Web: http://localhost (via Caddy)"
log "  - Caddy: http://localhost:80, https://localhost:443"
echo ""
log "Useful commands:"
log "  - View logs: docker-compose logs -f"
log "  - Stop services: docker-compose stop"
log "  - Restart services: docker-compose restart"
log "  - Remove all: docker-compose down -v"
log "  - Run migrations: bash scripts/migrate.sh"
log "  - Create backup: bash scripts/backup.sh"
echo ""
log "API Documentation:"
log "  - Health check: http://localhost:8080/health"
log "  - API base URL: http://localhost:8080/v1"
echo ""
log "Next steps:"
log "  1. Review and update .env file"
log "  2. Configure WhatsApp credentials (optional)"
log "  3. Test API endpoints"
log "  4. Access web UI: http://localhost"
echo ""

exit 0
