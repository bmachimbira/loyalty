#!/bin/bash

###########################################
# Deployment Rollback Script
# Zimbabwe Loyalty Platform
###########################################

set -e
set -u
set -o pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"

# Logging
log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
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

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Rollback to previous deployment"
    echo ""
    echo "Options:"
    echo "  --backup FILE      Restore database from specific backup file"
    echo "  --git-ref REF      Git reference to rollback to (tag, commit, branch)"
    echo "  -h, --help         Show this help message"
    echo ""
    exit 1
}

# Parse arguments
BACKUP_FILE=""
GIT_REF=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --backup)
            BACKUP_FILE="$2"
            shift 2
            ;;
        --git-ref)
            GIT_REF="$2"
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
log "  Deployment Rollback"
log "======================================"
log ""

# WARNING
log_warning "WARNING: This will rollback your deployment!"
log_warning "This action will:"
if [ -n "$BACKUP_FILE" ]; then
    log_warning "  - Restore database from: $BACKUP_FILE"
fi
if [ -n "$GIT_REF" ]; then
    log_warning "  - Rollback code to: $GIT_REF"
fi
log_warning "  - Restart all services"
log ""
read -p "Are you sure you want to continue? (yes/no): " -r
echo
if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    log "Rollback cancelled"
    exit 0
fi

# Rollback code if git ref provided
if [ -n "$GIT_REF" ]; then
    log "Rolling back code to $GIT_REF..."

    if git fetch && git checkout "$GIT_REF"; then
        log_success "Code rolled back to $GIT_REF"
    else
        log_error "Failed to rollback code"
        exit 1
    fi
fi

# Restore database if backup file provided
if [ -n "$BACKUP_FILE" ]; then
    log "Restoring database from backup..."

    if bash "$SCRIPT_DIR/restore.sh" "$BACKUP_FILE"; then
        log_success "Database restored from backup"
    else
        log_error "Failed to restore database"
        exit 1
    fi
fi

# Rebuild and restart services
log "Rebuilding and restarting services..."

if docker-compose -f "$COMPOSE_FILE" build && \
   docker-compose -f "$COMPOSE_FILE" up -d; then
    log_success "Services restarted"
else
    log_error "Failed to restart services"
    exit 1
fi

# Wait for services to be healthy
log "Waiting for services to be healthy..."
sleep 10

# Check health
if curl -f -s http://localhost:8080/health > /dev/null 2>&1; then
    log_success "API is healthy"
else
    log_warning "API health check failed"
fi

log ""
log_success "Rollback completed"
log "Check logs with: docker-compose -f $COMPOSE_FILE logs -f"

exit 0
