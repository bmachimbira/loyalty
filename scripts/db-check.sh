#!/bin/bash

###########################################
# Database Health Check Script
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

# Load environment variables
if [ -f "$PROJECT_DIR/.env.prod" ]; then
    source "$PROJECT_DIR/.env.prod"
elif [ -f "$PROJECT_DIR/.env" ]; then
    source "$PROJECT_DIR/.env"
fi

DB_NAME="${DB_NAME:-loyalty}"
DB_USER="${DB_USER:-postgres}"

echo "Database Health Check"
echo "===================="
echo ""

# Connection test
echo -n "Database connection: "
if docker exec loyalty-db pg_isready -U "$DB_USER" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ OK${NC}"
else
    echo -e "${RED}✗ FAILED${NC}"
    exit 1
fi

# Ping test
echo -n "Database ping: "
if docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ OK${NC}"
else
    echo -e "${RED}✗ FAILED${NC}"
    exit 1
fi

# Disk space check
echo ""
echo "Disk space:"
docker exec loyalty-db df -h | grep -E '(Filesystem|/$)' || true

# Connection pool stats
echo ""
echo "Active connections:"
docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "
    SELECT
        COUNT(*) as total,
        COUNT(*) FILTER (WHERE state = 'active') as active,
        COUNT(*) FILTER (WHERE state = 'idle') as idle
    FROM pg_stat_activity
    WHERE datname = '$DB_NAME';
"

# Slow queries
echo ""
echo "Long-running queries (>30 seconds):"
docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "
    SELECT
        pid,
        usename,
        application_name,
        state,
        age(clock_timestamp(), query_start) as duration,
        LEFT(query, 60) as query_preview
    FROM pg_stat_activity
    WHERE state != 'idle'
    AND query_start < NOW() - INTERVAL '30 seconds'
    ORDER BY query_start
    LIMIT 5;
"

# Database size
echo ""
echo "Database size:"
docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "
    SELECT pg_size_pretty(pg_database_size('$DB_NAME')) as size;
"

# Table count
echo ""
echo "Table statistics:"
docker exec loyalty-db psql -U "$DB_USER" -d "$DB_NAME" -c "
    SELECT
        schemaname,
        COUNT(*) as table_count,
        pg_size_pretty(SUM(pg_total_relation_size(schemaname||'.'||tablename))) as total_size
    FROM pg_tables
    WHERE schemaname = 'public'
    GROUP BY schemaname;
"

echo ""
echo -e "${GREEN}Health check completed${NC}"
