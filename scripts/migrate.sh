#!/bin/bash

###########################################
# Database Migration Script
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
MIGRATIONS_DIR="$PROJECT_DIR/migrations"

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
    echo "Run database migrations for the Loyalty Platform"
    echo ""
    echo "Options:"
    echo "  --dry-run          Show pending migrations without applying them"
    echo "  --rollback STEPS   Rollback last N migrations (not implemented)"
    echo "  -h, --help         Show this help message"
    echo ""
    exit 1
}

# Parse arguments
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
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

log "Database Migration Utility"
log "=========================="
log "Database: $DB_NAME"
log "User: $DB_USER"
log "Migrations directory: $MIGRATIONS_DIR"
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

# Check database connection
log "Checking database connection..."
if ! docker exec loyalty-db pg_isready -U "$DB_USER" > /dev/null 2>&1; then
    log_error "Database is not ready"
    exit 1
fi
log_success "Database connection OK"

# Create migrations tracking table if it doesn't exist
log "Ensuring migrations tracking table exists..."
docker exec -i loyalty-db psql -U "$DB_USER" -d "$DB_NAME" <<EOF
CREATE TABLE IF NOT EXISTS schema_migrations (
    id SERIAL PRIMARY KEY,
    version VARCHAR(255) NOT NULL UNIQUE,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);
EOF

log_success "Migrations tracking table ready"

# Get list of applied migrations
log "Checking applied migrations..."
APPLIED_MIGRATIONS=$(docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT version FROM schema_migrations ORDER BY version;" | tr -d ' ')

if [ -z "$APPLIED_MIGRATIONS" ]; then
    log "No migrations have been applied yet"
else
    log "Applied migrations:"
    echo "$APPLIED_MIGRATIONS" | while read -r migration; do
        [ -n "$migration" ] && log "  ✓ $migration"
    done
fi

# Get list of pending migrations
log ""
log "Scanning for pending migrations..."

PENDING_MIGRATIONS=()
MIGRATION_FILES=($(find "$MIGRATIONS_DIR" -name "*.sql" -type f | sort))

for migration_file in "${MIGRATION_FILES[@]}"; do
    migration_version=$(basename "$migration_file" .sql)

    # Check if migration has been applied
    if echo "$APPLIED_MIGRATIONS" | grep -q "^${migration_version}$"; then
        continue
    fi

    PENDING_MIGRATIONS+=("$migration_file")
done

if [ ${#PENDING_MIGRATIONS[@]} -eq 0 ]; then
    log_success "No pending migrations. Database is up to date!"
    exit 0
fi

log "Found ${#PENDING_MIGRATIONS[@]} pending migration(s):"
for migration in "${PENDING_MIGRATIONS[@]}"; do
    log "  → $(basename $migration)"
done

# Dry run mode
if [ "$DRY_RUN" = true ]; then
    log ""
    log_warning "DRY RUN MODE: No changes will be applied"
    exit 0
fi

# Apply migrations
log ""
log "Applying migrations..."

FAILED_MIGRATIONS=()
SUCCESSFUL_MIGRATIONS=()

for migration_file in "${PENDING_MIGRATIONS[@]}"; do
    migration_version=$(basename "$migration_file" .sql)
    migration_name=$(basename "$migration_file")

    log ""
    log "Applying: $migration_name"

    # Read migration description from first comment line
    description=$(head -n 5 "$migration_file" | grep -E "^-- " | head -n 1 | sed 's/^-- //')

    # Apply migration in a transaction
    if docker exec -i loyalty-db psql -U "$DB_USER" -d "$DB_NAME" <<EOF
BEGIN;

-- Apply migration
\i $migration_file

-- Record migration
INSERT INTO schema_migrations (version, description)
VALUES ('$migration_version', '$description')
ON CONFLICT (version) DO NOTHING;

COMMIT;
EOF
    then
        log_success "Migration applied: $migration_name"
        SUCCESSFUL_MIGRATIONS+=("$migration_name")
    else
        log_error "Migration failed: $migration_name"
        FAILED_MIGRATIONS+=("$migration_name")

        log_error "Migration failed. Rolling back transaction..."
        log "Check the migration file: $migration_file"
        break
    fi
done

# Summary
log ""
log "======================================"
log "Migration Summary"
log "======================================"
log "Total migrations found: ${#MIGRATION_FILES[@]}"
log "Pending migrations: ${#PENDING_MIGRATIONS[@]}"
log "Successfully applied: ${#SUCCESSFUL_MIGRATIONS[@]}"
log "Failed: ${#FAILED_MIGRATIONS[@]}"

if [ ${#FAILED_MIGRATIONS[@]} -gt 0 ]; then
    log ""
    log_error "Some migrations failed:"
    for migration in "${FAILED_MIGRATIONS[@]}"; do
        log "  ✗ $migration"
    done
    exit 1
fi

if [ ${#SUCCESSFUL_MIGRATIONS[@]} -gt 0 ]; then
    log ""
    log_success "All migrations applied successfully!"
fi

# Show current schema version
log ""
log "Current schema version:"
docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "
    SELECT version, applied_at, description
    FROM schema_migrations
    ORDER BY applied_at DESC
    LIMIT 5;
"

exit 0
