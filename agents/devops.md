# DevOps Agent

## Mission
Set up production deployment, monitoring, backups, and operational procedures.

## Prerequisites
- Docker and Docker Compose
- Basic Linux administration
- Understanding of PostgreSQL administration

## Tasks

### 1. Docker Optimization

**File**: `api/Dockerfile` (optimized)

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -a -installsuffix cgo \
    -o main ./cmd/api

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy binary
COPY --from=builder /app/main .

# Non-root user
RUN adduser -D -u 1000 appuser
USER appuser

EXPOSE 8080

CMD ["./main"]
```

**File**: `web/Dockerfile` (optimized)

```dockerfile
# Build stage
FROM node:18-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./
RUN npm ci --only=production

# Copy source
COPY . .

# Build
RUN npm run build

# Production stage
FROM nginx:alpine

# Copy built assets
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Non-root user
RUN chown -R nginx:nginx /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
```

### 2. Production Docker Compose

**File**: `docker-compose.prod.yml`

```yaml
version: '3.8'

services:
  db:
    image: postgres:16-alpine
    container_name: loyalty-db
    environment:
      POSTGRES_DB: ${DB_NAME}
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - dbdata:/var/lib/postgresql/data
      - ./backups:/backups
    restart: always
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - loyalty-network

  api:
    build:
      context: ./api
      dockerfile: Dockerfile
    container_name: loyalty-api
    environment:
      DATABASE_URL: "postgres://${DB_USER}:${DB_PASSWORD}@db:5432/${DB_NAME}?sslmode=require"
      JWT_SECRET: ${JWT_SECRET}
      PORT: "8080"
      GIN_MODE: "release"
      LOG_LEVEL: "info"
    depends_on:
      db:
        condition: service_healthy
    restart: always
    networks:
      - loyalty-network
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  web:
    build:
      context: ./web
      dockerfile: Dockerfile
    container_name: loyalty-web
    depends_on:
      - api
    restart: always
    networks:
      - loyalty-network

  caddy:
    image: caddy:2-alpine
    container_name: loyalty-caddy
    volumes:
      - ./Caddyfile.prod:/etc/caddy/Caddyfile
      - caddy_data:/data
      - caddy_config:/config
    depends_on:
      - web
      - api
    ports:
      - "80:80"
      - "443:443"
    restart: always
    networks:
      - loyalty-network

volumes:
  dbdata:
  caddy_data:
  caddy_config:

networks:
  loyalty-network:
    driver: bridge
```

### 3. Production Caddyfile

**File**: `Caddyfile.prod`

```caddy
{
    email admin@yourdomain.com
}

yourdomain.com {
    # Security headers
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
        X-XSS-Protection "1; mode=block"
        Referrer-Policy "strict-origin-when-cross-origin"
    }

    # Rate limiting
    rate_limit {
        zone api {
            key {remote_host}
            events 100
            window 1m
        }
    }

    # API routes
    @api {
        path /v1/*
        path /public/*
    }
    reverse_proxy @api api:8080 {
        health_uri /health
        health_interval 10s
    }

    # Web application
    reverse_proxy web:80

    # Logging
    log {
        output file /var/log/caddy/access.log
        format json
    }
}
```

### 4. Logging

**File**: `api/internal/logging/logger.go`

```go
package logging

import (
    "log/slog"
    "os"
)

func New() *slog.Logger {
    level := os.Getenv("LOG_LEVEL")
    var logLevel slog.Level

    switch level {
    case "debug":
        logLevel = slog.LevelDebug
    case "info":
        logLevel = slog.LevelInfo
    case "warn":
        logLevel = slog.LevelWarn
    case "error":
        logLevel = slog.LevelError
    default:
        logLevel = slog.LevelInfo
    }

    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: logLevel,
    })

    return slog.New(handler)
}
```

### 5. Monitoring

**File**: `api/internal/http/middleware/metrics.go`

```go
package middleware

import (
    "time"
    "github.com/gin-gonic/gin"
)

func Metrics() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        c.Next()

        duration := time.Since(start)

        // Log request metrics
        slog.Info("request",
            "method", c.Request.Method,
            "path", c.Request.URL.Path,
            "status", c.Writer.Status(),
            "duration_ms", duration.Milliseconds(),
            "ip", c.ClientIP(),
        )

        // Export to Prometheus (optional)
    }
}
```

### 6. Health Checks

**File**: `api/internal/http/handlers/health.go`

```go
package handlers

type HealthHandler struct {
    pool *pgxpool.Pool
}

func (h *HealthHandler) Check(c *gin.Context) {
    ctx := c.Request.Context()

    // Check database
    if err := h.pool.Ping(ctx); err != nil {
        c.JSON(503, gin.H{
            "status": "unhealthy",
            "error":  "database unavailable",
        })
        return
    }

    c.JSON(200, gin.H{
        "status": "healthy",
        "time":   time.Now().Format(time.RFC3339),
    })
}

func (h *HealthHandler) Ready(c *gin.Context) {
    // Check if service is ready to accept traffic
    // Check database, external dependencies, etc.

    c.JSON(200, gin.H{"status": "ready"})
}
```

### 7. Backup Scripts

**File**: `scripts/backup.sh`

```bash
#!/bin/bash

set -e

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups"
BACKUP_FILE="$BACKUP_DIR/loyalty_$DATE.dump"

echo "Starting backup: $BACKUP_FILE"

# Create backup
docker-compose exec -T db pg_dump -U ${DB_USER} -d ${DB_NAME} -F c -f /backups/loyalty_$DATE.dump

# Compress
gzip $BACKUP_FILE

# Delete backups older than 30 days
find $BACKUP_DIR -name "*.dump.gz" -mtime +30 -delete

# Upload to S3 (optional)
# aws s3 cp $BACKUP_FILE.gz s3://your-bucket/backups/

echo "Backup completed: $BACKUP_FILE.gz"
```

**File**: `scripts/restore.sh`

```bash
#!/bin/bash

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <backup-file>"
    exit 1
fi

BACKUP_FILE=$1

echo "Restoring from: $BACKUP_FILE"

# Decompress if gzipped
if [[ $BACKUP_FILE == *.gz ]]; then
    gunzip -k $BACKUP_FILE
    BACKUP_FILE=${BACKUP_FILE%.gz}
fi

# Stop services
docker-compose stop api web

# Restore database
docker-compose exec -T db pg_restore -U ${DB_USER} -d ${DB_NAME} -c $BACKUP_FILE

# Start services
docker-compose start api web

echo "Restore completed"
```

### 8. Deployment Script

**File**: `scripts/deploy.sh`

```bash
#!/bin/bash

set -e

echo "Starting deployment..."

# Pull latest code
git pull origin main

# Build images
docker-compose -f docker-compose.prod.yml build

# Run migrations
docker-compose -f docker-compose.prod.yml run --rm api ./migrate

# Rolling restart
docker-compose -f docker-compose.prod.yml up -d api
sleep 10  # Wait for health checks

docker-compose -f docker-compose.prod.yml up -d web

echo "Deployment completed"
```

### 9. Database Maintenance

**File**: `scripts/db-maintenance.sh`

```bash
#!/bin/bash

# Vacuum and analyze
docker-compose exec db psql -U ${DB_USER} -d ${DB_NAME} -c "VACUUM ANALYZE;"

# Reindex
docker-compose exec db psql -U ${DB_USER} -d ${DB_NAME} -c "REINDEX DATABASE ${DB_NAME};"

# Check for bloat
docker-compose exec db psql -U ${DB_USER} -d ${DB_NAME} -f /scripts/check-bloat.sql
```

### 10. Cron Jobs

**File**: `crontab`

```cron
# Backups (daily at 2 AM)
0 2 * * * /app/scripts/backup.sh >> /var/log/backup.log 2>&1

# DB maintenance (weekly on Sunday at 3 AM)
0 3 * * 0 /app/scripts/db-maintenance.sh >> /var/log/maintenance.log 2>&1

# Expire old rewards (hourly)
0 * * * * curl -X POST http://localhost:8080/internal/expire-rewards

# Reset monthly budgets (1st of month at midnight)
0 0 1 * * curl -X POST http://localhost:8080/internal/reset-monthly-budgets
```

### 11. Monitoring Dashboard

**File**: `docker-compose.monitoring.yml`

```yaml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

### 12. Environment Setup

**File**: `.env.prod`

```bash
# Database
DB_NAME=loyalty
DB_USER=postgres
DB_PASSWORD=<strong-password>

# API
JWT_SECRET=<strong-secret>
HMAC_KEYS_JSON={"key1":"<base64-secret>"}

# WhatsApp
WHATSAPP_VERIFY_TOKEN=<token>
WHATSAPP_APP_SECRET=<secret>
WHATSAPP_PHONE_ID=<phone-id>

# Logging
LOG_LEVEL=info
```

## Completion Criteria

- [ ] Docker images optimized
- [ ] Production docker-compose configured
- [ ] HTTPS enabled via Caddy
- [ ] Logging implemented
- [ ] Monitoring setup
- [ ] Health checks working
- [ ] Backup/restore procedures documented
- [ ] Deployment script tested
- [ ] Cron jobs configured
- [ ] Database maintenance scheduled
