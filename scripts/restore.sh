#!/bin/bash

###########################################
# PostgreSQL Restore Script
# Zimbabwe Loyalty Platform
###########################################

set -e  # Exit on error
set -u  # Exit on undefined variable
set -o pipefail  # Exit on pipe failure

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Load environment variables
if [ -f "$PROJECT_DIR/.env.prod" ]; then
    source "$PROJECT_DIR/.env.prod"
elif [ -f "$PROJECT_DIR/.env" ]; then
    source "$PROJECT_DIR/.env"
else
    echo -e "${RED}Error: No .env file found${NC}"
    exit 1
fi

DB_NAME="${DB_NAME:-loyalty}"
DB_USER="${DB_USER:-postgres}"

# Logging function
log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
}

log_success() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS: $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

# Show usage
usage() {
    echo "Usage: $0 <backup-file>"
    echo ""
    echo "Restore PostgreSQL database from backup"
    echo ""
    echo "Arguments:"
    echo "  backup-file    Path to backup file (.dump or .dump.gz)"
    echo ""
    echo "Examples:"
    echo "  $0 backups/loyalty_20231114_120000.dump.gz"
    echo "  $0 backups/loyalty_20231114_120000.dump"
    echo ""
    echo "Available backups:"
    if [ -d "$PROJECT_DIR/backups" ]; then
        find "$PROJECT_DIR/backups" -name "*.dump.gz" -o -name "*.dump" | sort -r | head -10 | while read file; do
            size=$(du -h "$file" | cut -f1)
            echo "  $(basename $file) - $size"
        done
    else
        echo "  No backups found in $PROJECT_DIR/backups"
    fi
    exit 1
}

# Check arguments
if [ $# -eq 0 ]; then
    usage
fi

BACKUP_FILE="$1"

# Check if backup file exists
if [ ! -f "$BACKUP_FILE" ]; then
    log_error "Backup file not found: $BACKUP_FILE"
    exit 1
fi

# Get absolute path
BACKUP_FILE=$(realpath "$BACKUP_FILE")

log "Database Restore Utility"
log "========================"
log "Backup file: $BACKUP_FILE"
log "Database: $DB_NAME"
log "User: $DB_USER"
log ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    log_error "Docker is not running"
    exit 1
fi

# Check if database container is running
if ! docker ps | grep -q loyalty-db; then
    log_error "Database container is not running"
    log "Start it with: docker-compose up -d db"
    exit 1
fi

# WARNING
log_warning "WARNING: This will OVERWRITE the current database!"
log_warning "Current database: $DB_NAME"
log_warning "All existing data will be LOST!"
log ""
read -p "Are you sure you want to continue? (yes/no): " -r
echo
if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    log "Restore cancelled"
    exit 0
fi

# Additional confirmation for production
if grep -q "GIN_MODE=release" "$PROJECT_DIR/.env.prod" 2>/dev/null; then
    log_warning "PRODUCTION ENVIRONMENT DETECTED!"
    log_warning "Type the database name to confirm: "
    read -p "Database name: " -r
    echo
    if [[ $REPLY != $DB_NAME ]]; then
        log_error "Database name does not match. Restore cancelled"
        exit 1
    fi
fi

# Decompress if needed
RESTORE_FILE="$BACKUP_FILE"
if [[ $BACKUP_FILE == *.gz ]]; then
    log "Decompressing backup file..."
    RESTORE_FILE="/tmp/restore_$(date +%s).dump"
    if gunzip -c "$BACKUP_FILE" > "$RESTORE_FILE"; then
        log_success "Backup file decompressed"
    else
        log_error "Failed to decompress backup file"
        exit 1
    fi
fi

# Verify backup file
log "Verifying backup file..."
BACKUP_SIZE=$(du -h "$RESTORE_FILE" | cut -f1)
log "Backup size: $BACKUP_SIZE"

# Create a safety backup of current database before restore
log "Creating safety backup of current database..."
SAFETY_BACKUP="/tmp/safety_backup_$(date +%s).dump"
if docker exec loyalty-db pg_dump -U "$DB_USER" -d "$DB_NAME" -F c -f "$SAFETY_BACKUP"; then
    log_success "Safety backup created: $SAFETY_BACKUP"
    log "You can restore this if needed with: docker exec loyalty-db pg_restore -U $DB_USER -d $DB_NAME -c $SAFETY_BACKUP"
else
    log_warning "Failed to create safety backup"
    read -p "Continue without safety backup? (yes/no): " -r
    echo
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        log "Restore cancelled"
        [ -f "$RESTORE_FILE" ] && [ "$RESTORE_FILE" != "$BACKUP_FILE" ] && rm -f "$RESTORE_FILE"
        exit 1
    fi
fi

# Stop API and Web services to prevent connections during restore
log "Stopping API and Web services..."
docker-compose -f "$PROJECT_DIR/docker-compose.prod.yml" stop api web 2>/dev/null || \
docker-compose -f "$PROJECT_DIR/docker-compose.yml" stop api web 2>/dev/null || \
log_warning "Could not stop services (they may not be running)"

# Wait a moment for connections to close
sleep 2

# Terminate existing connections to the database
log "Terminating existing database connections..."
docker exec loyalty-db psql -U "$DB_USER" -d postgres -c "
    SELECT pg_terminate_backend(pg_stat_activity.pid)
    FROM pg_stat_activity
    WHERE pg_stat_activity.datname = '$DB_NAME'
    AND pid <> pg_backend_pid();
" || log_warning "Could not terminate all connections"

# Copy backup file to container
log "Copying backup file to database container..."
CONTAINER_BACKUP="/tmp/restore_backup.dump"
if docker cp "$RESTORE_FILE" "loyalty-db:$CONTAINER_BACKUP"; then
    log_success "Backup file copied to container"
else
    log_error "Failed to copy backup file to container"
    [ -f "$RESTORE_FILE" ] && [ "$RESTORE_FILE" != "$BACKUP_FILE" ] && rm -f "$RESTORE_FILE"
    exit 1
fi

# Perform restore
log "Restoring database from backup..."
log "This may take several minutes depending on database size..."

if docker exec loyalty-db pg_restore -U "$DB_USER" -d "$DB_NAME" --clean --if-exists "$CONTAINER_BACKUP"; then
    log_success "Database restored successfully"
else
    log_error "Database restore failed"
    log "Attempting to restore from safety backup..."

    if [ -f "$SAFETY_BACKUP" ]; then
        docker exec loyalty-db pg_restore -U "$DB_USER" -d "$DB_NAME" -c "$SAFETY_BACKUP" && \
        log_success "Rolled back to safety backup" || \
        log_error "Failed to restore safety backup"
    fi

    # Cleanup
    docker exec loyalty-db rm -f "$CONTAINER_BACKUP"
    [ -f "$RESTORE_FILE" ] && [ "$RESTORE_FILE" != "$BACKUP_FILE" ] && rm -f "$RESTORE_FILE"
    exit 1
fi

# Cleanup container backup
docker exec loyalty-db rm -f "$CONTAINER_BACKUP"

# Verify restore
log "Verifying restore..."
if docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT COUNT(*) FROM tenants;" > /dev/null 2>&1; then
    log_success "Database verification passed"
else
    log_error "Database verification failed"
fi

# Restart services
log "Starting API and Web services..."
docker-compose -f "$PROJECT_DIR/docker-compose.prod.yml" start api web 2>/dev/null || \
docker-compose -f "$PROJECT_DIR/docker-compose.yml" start api web 2>/dev/null || \
log_warning "Could not start services - start them manually"

# Cleanup temporary files
[ -f "$RESTORE_FILE" ] && [ "$RESTORE_FILE" != "$BACKUP_FILE" ] && rm -f "$RESTORE_FILE"

log_success "Restore completed successfully"
log "Services are restarting. Check logs with: docker-compose logs -f api"

exit 0
