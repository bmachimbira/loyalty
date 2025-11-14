#!/bin/bash

###########################################
# Database Maintenance Script
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
LOG_FILE="$PROJECT_DIR/logs/db-maintenance_$(date +%Y%m%d).log"

# Ensure log directory exists
mkdir -p "$PROJECT_DIR/logs"

# Logging function
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS:${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING:${NC} $1" | tee -a "$LOG_FILE"
}

# Load environment variables
if [ -f "$PROJECT_DIR/.env.prod" ]; then
    source "$PROJECT_DIR/.env.prod"
elif [ -f "$PROJECT_DIR/.env" ]; then
    source "$PROJECT_DIR/.env"
else
    log_error "No .env file found"
    exit 1
fi

DB_NAME="${DB_NAME:-loyalty}"
DB_USER="${DB_USER:-postgres}"

log "======================================"
log "  Database Maintenance"
log "======================================"
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
    exit 1
fi

# Check database connection
log "Checking database connection..."
if docker exec loyalty-db pg_isready -U "$DB_USER" > /dev/null 2>&1; then
    log_success "Database connection OK"
else
    log_error "Database is not ready"
    exit 1
fi

# Get database size before maintenance
log ""
log "Database size before maintenance:"
docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "
    SELECT
        pg_size_pretty(pg_database_size('$DB_NAME')) as database_size;
" | tee -a "$LOG_FILE"

# VACUUM ANALYZE
log ""
log "Running VACUUM ANALYZE..."
log "This may take several minutes depending on database size..."

START_TIME=$(date +%s)

if docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "VACUUM ANALYZE VERBOSE;" >> "$LOG_FILE" 2>&1; then
    VACUUM_TIME=$(($(date +%s) - START_TIME))
    log_success "VACUUM ANALYZE completed in ${VACUUM_TIME}s"
else
    log_error "VACUUM ANALYZE failed"
    exit 1
fi

# REINDEX
log ""
log "Rebuilding indexes..."
START_TIME=$(date +%s)

# Get list of tables
TABLES=$(docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -t -c "
    SELECT tablename
    FROM pg_tables
    WHERE schemaname = 'public'
    ORDER BY tablename;
" | tr -d ' ')

for table in $TABLES; do
    [ -z "$table" ] && continue
    log "Reindexing table: $table"
    if docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "REINDEX TABLE $table;" >> "$LOG_FILE" 2>&1; then
        log_success "  ✓ $table reindexed"
    else
        log_warning "  ✗ Failed to reindex $table"
    fi
done

REINDEX_TIME=$(($(date +%s) - START_TIME))
log_success "REINDEX completed in ${REINDEX_TIME}s"

# Update statistics
log ""
log "Updating table statistics..."
if docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "ANALYZE;" >> "$LOG_FILE" 2>&1; then
    log_success "Statistics updated"
else
    log_warning "Failed to update statistics"
fi

# Check for bloat
log ""
log "Checking for table bloat..."
docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "
    SELECT
        schemaname,
        tablename,
        pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size,
        pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) - pg_relation_size(schemaname||'.'||tablename)) AS external_size
    FROM pg_tables
    WHERE schemaname = 'public'
    ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
    LIMIT 10;
" | tee -a "$LOG_FILE"

# Check index usage
log ""
log "Index usage statistics (unused indexes):"
docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "
    SELECT
        schemaname,
        tablename,
        indexname,
        idx_scan as index_scans,
        pg_size_pretty(pg_relation_size(indexrelid)) as index_size
    FROM pg_stat_user_indexes
    WHERE idx_scan = 0
    AND indexrelname NOT LIKE '%_pkey'
    ORDER BY pg_relation_size(indexrelid) DESC
    LIMIT 10;
" | tee -a "$LOG_FILE"

# Check slow queries (if pg_stat_statements is enabled)
log ""
log "Checking for slow queries..."
docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "
    SELECT EXISTS (
        SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements'
    ) as pg_stat_statements_enabled;
" >> "$LOG_FILE"

# Get database size after maintenance
log ""
log "Database size after maintenance:"
docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "
    SELECT
        pg_size_pretty(pg_database_size('$DB_NAME')) as database_size;
" | tee -a "$LOG_FILE"

# Get table sizes
log ""
log "Top 10 largest tables:"
docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "
    SELECT
        tablename,
        pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
        pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) AS table_size,
        pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) - pg_relation_size(schemaname||'.'||tablename)) AS indexes_size
    FROM pg_tables
    WHERE schemaname = 'public'
    ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
    LIMIT 10;
" | tee -a "$LOG_FILE"

# Connection statistics
log ""
log "Database connections:"
docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "
    SELECT
        COUNT(*) as total_connections,
        COUNT(*) FILTER (WHERE state = 'active') as active,
        COUNT(*) FILTER (WHERE state = 'idle') as idle,
        COUNT(*) FILTER (WHERE state = 'idle in transaction') as idle_in_transaction
    FROM pg_stat_activity
    WHERE datname = '$DB_NAME';
" | tee -a "$LOG_FILE"

# Check for long-running transactions
log ""
log "Long-running transactions (>5 minutes):"
docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "
    SELECT
        pid,
        usename,
        application_name,
        state,
        age(clock_timestamp(), query_start) as duration,
        query
    FROM pg_stat_activity
    WHERE state != 'idle'
    AND query_start < NOW() - INTERVAL '5 minutes'
    ORDER BY query_start;
" | tee -a "$LOG_FILE"

# Summary
log ""
log "======================================"
log_success "Database maintenance completed"
log "======================================"
log "Log file: $LOG_FILE"
log "Total time: $(($(date +%s) - START_TIME))s"
log ""
log "Maintenance tasks performed:"
log "  ✓ VACUUM ANALYZE"
log "  ✓ REINDEX all tables"
log "  ✓ Update statistics"
log "  ✓ Check for bloat"
log "  ✓ Check index usage"
log "  ✓ Check connections"
log ""

exit 0
