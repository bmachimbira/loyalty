#!/bin/bash

###########################################
# PostgreSQL Backup Script
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
BACKUP_DIR="${BACKUP_DIR:-$PROJECT_DIR/backups}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DATE_STAMP=$(date +%Y%m%d)

# Load environment variables
if [ -f "$PROJECT_DIR/.env.prod" ]; then
    source "$PROJECT_DIR/.env.prod"
elif [ -f "$PROJECT_DIR/.env" ]; then
    source "$PROJECT_DIR/.env"
else
    echo -e "${RED}Error: No .env file found${NC}"
    exit 1
fi

# Backup file names
DB_NAME="${DB_NAME:-loyalty}"
BACKUP_FILE="$BACKUP_DIR/${DB_NAME}_${TIMESTAMP}.dump"
LOG_FILE="$BACKUP_DIR/backup_${DATE_STAMP}.log"

# Ensure backup directory exists
mkdir -p "$BACKUP_DIR"

# Logging function
log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS: $1${NC}" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}" | tee -a "$LOG_FILE"
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    log_error "Docker is not running"
    exit 1
fi

# Check if database container is running
if ! docker ps | grep -q loyalty-db; then
    log_error "Database container is not running"
    exit 1
fi

log "Starting database backup"
log "Database: $DB_NAME"
log "Backup file: $BACKUP_FILE"

# Perform backup using pg_dump
log "Creating PostgreSQL dump..."
if docker exec loyalty-db pg_dump -U "${DB_USER:-postgres}" -d "$DB_NAME" -F c -f "/tmp/backup_${TIMESTAMP}.dump"; then
    log_success "Database dump created successfully"
else
    log_error "Failed to create database dump"
    exit 1
fi

# Copy backup from container to host
log "Copying backup file from container..."
if docker cp "loyalty-db:/tmp/backup_${TIMESTAMP}.dump" "$BACKUP_FILE"; then
    log_success "Backup file copied to host"
    # Remove backup from container
    docker exec loyalty-db rm "/tmp/backup_${TIMESTAMP}.dump"
else
    log_error "Failed to copy backup file"
    exit 1
fi

# Get backup file size
BACKUP_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
log "Backup size: $BACKUP_SIZE"

# Compress backup
log "Compressing backup..."
if gzip "$BACKUP_FILE"; then
    COMPRESSED_FILE="${BACKUP_FILE}.gz"
    COMPRESSED_SIZE=$(du -h "$COMPRESSED_FILE" | cut -f1)
    log_success "Backup compressed (size: $COMPRESSED_SIZE)"
else
    log_error "Failed to compress backup"
    COMPRESSED_FILE="$BACKUP_FILE"
fi

# Verify backup integrity
log "Verifying backup integrity..."
if gunzip -t "$COMPRESSED_FILE" 2>/dev/null; then
    log_success "Backup integrity verified"
else
    log_warning "Backup integrity check failed or backup is not compressed"
fi

# Upload to S3 if configured
if [ -n "${S3_BUCKET:-}" ] && command -v aws >/dev/null 2>&1; then
    log "Uploading backup to S3..."
    S3_PATH="s3://${S3_BUCKET}/backups/${DB_NAME}/${DATE_STAMP}/$(basename $COMPRESSED_FILE)"

    if aws s3 cp "$COMPRESSED_FILE" "$S3_PATH"; then
        log_success "Backup uploaded to $S3_PATH"
    else
        log_warning "Failed to upload backup to S3"
    fi
fi

# Clean up old backups
log "Cleaning up old backups (retention: $RETENTION_DAYS days)..."
DELETED_COUNT=$(find "$BACKUP_DIR" -name "${DB_NAME}_*.dump.gz" -type f -mtime +$RETENTION_DAYS -delete -print | wc -l)
if [ "$DELETED_COUNT" -gt 0 ]; then
    log "Deleted $DELETED_COUNT old backup(s)"
else
    log "No old backups to delete"
fi

# List recent backups
log "Recent backups:"
find "$BACKUP_DIR" -name "${DB_NAME}_*.dump.gz" -type f -printf "%T@ %p\n" | sort -rn | head -5 | while read timestamp file; do
    size=$(du -h "$file" | cut -f1)
    date=$(date -d "@${timestamp%.*}" '+%Y-%m-%d %H:%M:%S')
    log "  $date - $(basename $file) - $size"
done

# Calculate total backup size
TOTAL_SIZE=$(du -sh "$BACKUP_DIR" | cut -f1)
log "Total backup directory size: $TOTAL_SIZE"

# Send notification if email is configured
if [ -n "${BACKUP_NOTIFICATION_EMAIL:-}" ] && command -v mail >/dev/null 2>&1; then
    log "Sending notification email..."
    echo "Backup completed successfully at $(date)

Database: $DB_NAME
Backup file: $(basename $COMPRESSED_FILE)
Size: $COMPRESSED_SIZE
Location: $BACKUP_DIR

Total backups: $(find "$BACKUP_DIR" -name "${DB_NAME}_*.dump.gz" -type f | wc -l)
Total size: $TOTAL_SIZE
" | mail -s "Loyalty Platform Backup Success - $DATE_STAMP" "$BACKUP_NOTIFICATION_EMAIL"
fi

log_success "Backup completed successfully"
log "Backup location: $COMPRESSED_FILE"

exit 0
