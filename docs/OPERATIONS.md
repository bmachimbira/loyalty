# Operations Guide
# Zimbabwe Loyalty Platform

This document provides guidance for day-to-day operations, monitoring, maintenance, and troubleshooting.

## Table of Contents

- [Daily Operations](#daily-operations)
- [Monitoring](#monitoring)
- [Backup and Restore](#backup-and-restore)
- [Database Maintenance](#database-maintenance)
- [Performance Tuning](#performance-tuning)
- [Scaling](#scaling)
- [Security Operations](#security-operations)
- [Common Issues](#common-issues)
- [Incident Response](#incident-response)

## Daily Operations

### Morning Checks

```bash
# 1. Check service status
docker-compose -f docker-compose.prod.yml ps

# 2. Check health endpoints
curl http://localhost:8080/health
curl http://localhost:8080/ready

# 3. Check database health
bash scripts/db-check.sh

# 4. Review logs for errors
docker-compose -f docker-compose.prod.yml logs --tail=100 | grep -i error

# 5. Check disk space
df -h

# 6. Check memory usage
free -h

# 7. Check resource usage
docker stats --no-stream
```

### Service Management

#### Start Services

```bash
# Start all services
docker-compose -f docker-compose.prod.yml up -d

# Start specific service
docker-compose -f docker-compose.prod.yml up -d api
```

#### Stop Services

```bash
# Stop all services
docker-compose -f docker-compose.prod.yml stop

# Stop specific service
docker-compose -f docker-compose.prod.yml stop api
```

#### Restart Services

```bash
# Restart all services
docker-compose -f docker-compose.prod.yml restart

# Restart specific service
docker-compose -f docker-compose.prod.yml restart api

# Graceful restart
docker-compose -f docker-compose.prod.yml restart -t 30 api
```

#### View Logs

```bash
# All services (follow mode)
docker-compose -f docker-compose.prod.yml logs -f

# Specific service
docker-compose -f docker-compose.prod.yml logs -f api

# Last N lines
docker-compose -f docker-compose.prod.yml logs --tail=100 api

# Since timestamp
docker-compose -f docker-compose.prod.yml logs --since 1h api

# Save to file
docker-compose -f docker-compose.prod.yml logs api > api-logs.txt
```

## Monitoring

### Health Checks

#### API Health Check

```bash
curl -i http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2023-11-14T12:00:00Z",
  "checks": {
    "database": {
      "status": "healthy",
      "message": "Database connection OK",
      "latency": "2ms"
    }
  }
}
```

#### Readiness Check

```bash
curl -i http://localhost:8080/ready
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2023-11-14T12:00:00Z",
  "checks": {
    "database": {
      "status": "healthy"
    },
    "database_pool": {
      "status": "healthy"
    },
    "database_migration": {
      "status": "healthy"
    }
  }
}
```

### Database Monitoring

```bash
# Quick health check
bash scripts/db-check.sh

# Connection statistics
docker exec loyalty-db-prod psql -U postgres -d loyalty -c "
  SELECT
    COUNT(*) as total_connections,
    COUNT(*) FILTER (WHERE state = 'active') as active,
    COUNT(*) FILTER (WHERE state = 'idle') as idle
  FROM pg_stat_activity
  WHERE datname = 'loyalty';
"

# Database size
docker exec loyalty-db-prod psql -U postgres -d loyalty -c "
  SELECT pg_size_pretty(pg_database_size('loyalty'));
"

# Table sizes
docker exec loyalty-db-prod psql -U postgres -d loyalty -c "
  SELECT
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
  FROM pg_tables
  WHERE schemaname = 'public'
  ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
  LIMIT 10;
"

# Slow queries
docker exec loyalty-db-prod psql -U postgres -d loyalty -c "
  SELECT
    pid,
    usename,
    application_name,
    state,
    age(clock_timestamp(), query_start) as duration,
    query
  FROM pg_stat_activity
  WHERE state != 'idle'
  AND query_start < NOW() - INTERVAL '10 seconds'
  ORDER BY query_start;
"
```

### Resource Monitoring

```bash
# Docker container stats
docker stats

# Disk usage
df -h
du -sh /var/lib/docker/volumes/*

# Memory usage
free -h

# CPU usage
top -b -n 1 | head -20

# Network connections
netstat -tulpn | grep -E ':(80|443|8080|5432)'
```

### Log Monitoring

```bash
# Monitor API logs for errors
docker-compose -f docker-compose.prod.yml logs -f api | grep -i error

# Count errors in last hour
docker-compose -f docker-compose.prod.yml logs --since 1h api | grep -i error | wc -l

# Monitor database logs
docker-compose -f docker-compose.prod.yml logs -f db | grep -i error

# Monitor all services
tail -f logs/*.log
```

## Backup and Restore

### Manual Backup

```bash
# Create backup
bash scripts/backup.sh

# Verify backup
ls -lh backups/

# Test backup integrity
gunzip -t backups/loyalty_YYYYMMDD_HHMMSS.dump.gz
```

### Automated Backups

Backups are automated via cron (if set up):

```bash
# Check cron jobs
crontab -l

# View backup logs
tail -f logs/backup.log

# Check recent backups
find backups/ -name "*.dump.gz" -mtime -7 -ls
```

### Restore from Backup

```bash
# List available backups
ls -lh backups/

# Restore (interactive with confirmation)
bash scripts/restore.sh backups/loyalty_20231114_120000.dump.gz
```

### Backup to S3

If S3 is configured in .env.prod:

```bash
# Backup will automatically upload to S3
bash scripts/backup.sh

# Manual S3 upload (if configured)
aws s3 cp backups/loyalty_20231114_120000.dump.gz s3://your-bucket/backups/
```

## Database Maintenance

### Weekly Maintenance

Automated via cron (Sundays at 3 AM):

```bash
# Manual execution
bash scripts/db-maintenance.sh

# View maintenance logs
tail -f logs/db-maintenance*.log
```

### VACUUM ANALYZE

```bash
# Full VACUUM ANALYZE
docker exec loyalty-db-prod psql -U postgres -d loyalty -c "VACUUM ANALYZE;"

# VACUUM specific table
docker exec loyalty-db-prod psql -U postgres -d loyalty -c "VACUUM ANALYZE events;"
```

### REINDEX

```bash
# Reindex database
docker exec loyalty-db-prod psql -U postgres -d loyalty -c "REINDEX DATABASE loyalty;"

# Reindex specific table
docker exec loyalty-db-prod psql -U postgres -d loyalty -c "REINDEX TABLE events;"
```

### Check Table Bloat

```bash
docker exec loyalty-db-prod psql -U postgres -d loyalty -c "
  SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) AS table_size,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) - pg_relation_size(schemaname||'.'||tablename)) AS indexes_size
  FROM pg_tables
  WHERE schemaname = 'public'
  ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
"
```

## Performance Tuning

### Database Performance

#### Connection Pool Tuning

Edit `docker-compose.prod.yml` database configuration:

```yaml
command: >
  postgres
  -c max_connections=200
  -c shared_buffers=256MB
  -c effective_cache_size=1GB
  -c work_mem=2621kB
```

#### Query Optimization

```bash
# Find slow queries
docker exec loyalty-db-prod psql -U postgres -d loyalty -c "
  SELECT
    calls,
    total_time,
    mean_time,
    query
  FROM pg_stat_statements
  ORDER BY mean_time DESC
  LIMIT 10;
"

# Analyze query plan
docker exec -i loyalty-db-prod psql -U postgres -d loyalty <<EOF
EXPLAIN ANALYZE
SELECT * FROM events WHERE tenant_id = 'xxx' ORDER BY created_at DESC LIMIT 10;
EOF
```

### Application Performance

#### API Response Times

Monitor logs for slow requests:

```bash
docker-compose -f docker-compose.prod.yml logs api | grep '"duration_ms"' | awk '{print $NF}' | sort -n | tail -20
```

#### Memory Optimization

```bash
# Check API memory usage
docker stats loyalty-api --no-stream

# Restart API if memory is high
docker-compose -f docker-compose.prod.yml restart api
```

### Caching

Redis is included for caching. Monitor usage:

```bash
# Connect to Redis
docker exec -it loyalty-redis-prod redis-cli

# Check memory usage
docker exec loyalty-redis-prod redis-cli INFO memory

# Check cache hit rate
docker exec loyalty-redis-prod redis-cli INFO stats | grep -E '(keyspace_hits|keyspace_misses)'
```

## Scaling

### Vertical Scaling

Increase resources in `docker-compose.prod.yml`:

```yaml
services:
  api:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G
```

### Horizontal Scaling

Scale API service:

```bash
# Scale to 4 instances
docker-compose -f docker-compose.prod.yml up -d --scale api=4

# Verify scaling
docker-compose -f docker-compose.prod.yml ps api
```

### Database Scaling

For read replicas and clustering, consider:
- PostgreSQL streaming replication
- pgpool-II for connection pooling
- External managed database (RDS, Cloud SQL)

## Security Operations

### Security Monitoring

```bash
# Check failed login attempts
docker-compose -f docker-compose.prod.yml logs api | grep "authentication failed"

# Check unauthorized access attempts
docker-compose -f docker-compose.prod.yml logs api | grep "401\|403"

# Check rate limiting
docker-compose -f docker-compose.prod.yml logs api | grep "rate limit exceeded"
```

### Security Updates

```bash
# Update Docker images
docker-compose -f docker-compose.prod.yml pull

# Update system packages
sudo apt-get update && sudo apt-get upgrade -y

# Update application
git pull origin main
bash scripts/deploy.sh
```

### SSL Certificate Management

```bash
# Check certificate expiry
docker exec loyalty-caddy-prod caddy list-certificates

# Force certificate renewal
docker exec loyalty-caddy-prod caddy reload --config /etc/caddy/Caddyfile
```

### Secret Rotation

```bash
# Generate new secrets
bash scripts/generate-secrets.sh

# Update .env.prod with new secrets
nano .env.prod

# Restart services
docker-compose -f docker-compose.prod.yml restart
```

## Common Issues

### Issue: API Not Responding

```bash
# Check API is running
docker-compose -f docker-compose.prod.yml ps api

# Check API logs
docker-compose -f docker-compose.prod.yml logs --tail=100 api

# Check health endpoint
curl http://localhost:8080/health

# Restart API
docker-compose -f docker-compose.prod.yml restart api
```

### Issue: Database Connection Errors

```bash
# Check database is running
docker-compose -f docker-compose.prod.yml ps db

# Check database logs
docker-compose -f docker-compose.prod.yml logs db

# Check connections
docker exec loyalty-db-prod psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# Restart database
docker-compose -f docker-compose.prod.yml restart db
```

### Issue: High Memory Usage

```bash
# Check memory usage
docker stats

# Check largest consumers
docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}"

# Restart service
docker-compose -f docker-compose.prod.yml restart <service>

# Clear Docker cache
docker system prune -a
```

### Issue: Disk Space Full

```bash
# Check disk usage
df -h

# Check Docker disk usage
docker system df

# Clean up old images
docker image prune -a

# Clean up old volumes
docker volume prune

# Clean up old backups
find backups/ -name "*.dump.gz" -mtime +30 -delete

# Clean up old logs
find logs/ -name "*.log" -mtime +30 -delete
```

## Incident Response

### Severity Levels

**P0 - Critical**: Complete service outage
**P1 - High**: Major feature unavailable
**P2 - Medium**: Minor feature issues
**P3 - Low**: Cosmetic issues

### P0 Incident Response

1. **Acknowledge**: Post in team chat
2. **Assess**: Check health endpoints, logs, metrics
3. **Mitigate**: Rollback or quick fix
4. **Communicate**: Update status page
5. **Resolve**: Full fix and verification
6. **Post-mortem**: Document incident and learnings

### Quick Rollback

```bash
# Rollback to previous version
bash scripts/rollback.sh --git-ref <previous-tag>

# Or rollback with database restore
bash scripts/rollback.sh --backup <backup-file> --git-ref <previous-tag>
```

### Emergency Contacts

- **On-call Engineer**: Check rotation schedule
- **Database Admin**: Check contact list
- **Infrastructure Team**: Check contact list

### Status Page Update

Update users during incidents:
- Post on status page
- Send email notifications
- Update social media if major

## Maintenance Windows

### Scheduled Maintenance

1. **Announce**: 48 hours in advance
2. **Backup**: Create full backup
3. **Execute**: During low-traffic period
4. **Verify**: Test all critical functions
5. **Monitor**: Watch for issues post-maintenance

### Maintenance Checklist

- [ ] Create backup
- [ ] Test changes in staging
- [ ] Notify users
- [ ] Execute during maintenance window
- [ ] Monitor logs and metrics
- [ ] Verify all services healthy
- [ ] Update documentation
- [ ] Close maintenance window

## Performance Baselines

### Expected Metrics

- **API Response Time (p95)**: < 150ms
- **Database Query Time**: < 25ms
- **Uptime**: > 99.9%
- **Error Rate**: < 0.1%

### When to Alert

- API response time > 500ms
- Error rate > 1%
- Database connections > 80% of max
- Disk usage > 80%
- Memory usage > 85%
- CPU usage > 90% for > 5 minutes

## Best Practices

1. **Always create backups** before making changes
2. **Test in staging** before deploying to production
3. **Monitor logs** during and after deployments
4. **Document all changes** in runbook
5. **Communicate** with team about changes
6. **Use rollback** if issues arise
7. **Review metrics** daily
8. **Keep secrets** secure and rotated
9. **Update documentation** when processes change
10. **Conduct post-mortems** after incidents
