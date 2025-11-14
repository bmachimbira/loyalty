# Deployment Guide
# Zimbabwe Loyalty Platform

This document provides comprehensive deployment instructions for the Zimbabwe Loyalty Platform.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Server Requirements](#server-requirements)
- [Initial Server Setup](#initial-server-setup)
- [Application Deployment](#application-deployment)
- [Environment Configuration](#environment-configuration)
- [SSL/TLS Configuration](#ssltls-configuration)
- [Database Setup](#database-setup)
- [Monitoring Setup](#monitoring-setup)
- [Troubleshooting](#troubleshooting)
- [Rollback Procedures](#rollback-procedures)

## Prerequisites

### Required Software

- **Operating System**: Ubuntu 20.04 LTS or 22.04 LTS (recommended)
- **Docker**: Version 20.10 or higher
- **Docker Compose**: Version 2.0 or higher
- **Git**: For code deployment
- **Domain Name**: With DNS configured

### Required Access

- Root or sudo access to the server
- SSH access to the server
- Access to GitHub repository
- Domain DNS control
- Meta App Dashboard access (for WhatsApp integration)

### Minimum Server Specifications

**Development/Staging:**
- 2 CPU cores
- 4 GB RAM
- 40 GB SSD storage
- Ubuntu 20.04/22.04 LTS

**Production:**
- 4 CPU cores
- 8 GB RAM
- 100 GB SSD storage
- Ubuntu 20.04/22.04 LTS
- Backup storage (S3 or equivalent)

## Initial Server Setup

### 1. Automated Setup (Recommended)

Run the automated production server setup script:

```bash
# Download and run setup script
wget https://raw.githubusercontent.com/yourusername/loyalty/main/scripts/setup-prod.sh
sudo bash setup-prod.sh
```

This script will:
- Install Docker and Docker Compose
- Configure firewall (UFW)
- Set up fail2ban
- Create deployment user
- Create project directory
- Configure log rotation
- Set up swap space
- Optimize kernel parameters

### 2. Manual Setup

If you prefer manual setup:

#### Install Docker

```bash
# Update package index
sudo apt-get update
sudo apt-get upgrade -y

# Install prerequisites
sudo apt-get install -y apt-transport-https ca-certificates curl software-properties-common

# Add Docker GPG key
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -

# Add Docker repository
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"

# Install Docker
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io

# Start and enable Docker
sudo systemctl start docker
sudo systemctl enable docker
```

#### Install Docker Compose

```bash
# Download Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.20.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose

# Make executable
sudo chmod +x /usr/local/bin/docker-compose

# Verify installation
docker-compose --version
```

#### Configure Firewall

```bash
# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP
sudo ufw allow 80/tcp

# Allow HTTPS
sudo ufw allow 443/tcp

# Enable firewall
sudo ufw enable
```

#### Create Deployment User

```bash
# Create user
sudo useradd -m -s /bin/bash deploy

# Add to docker group
sudo usermod -aG docker deploy

# Create project directory
sudo mkdir -p /opt/loyalty
sudo chown -R deploy:deploy /opt/loyalty
```

## Application Deployment

### 1. Clone Repository

```bash
# Switch to deployment user
sudo su - deploy

# Clone repository
cd /opt/loyalty
git clone https://github.com/yourusername/loyalty.git .
```

### 2. Generate Secrets

```bash
# Generate all required secrets
bash scripts/generate-secrets.sh

# This will generate:
# - JWT_SECRET
# - HMAC_KEYS_JSON
# - DB_PASSWORD
# - INTERNAL_API_SECRET
# - WHATSAPP_VERIFY_TOKEN
```

### 3. Configure Environment

```bash
# Copy environment template
cp .env.prod.example .env.prod

# Edit configuration
nano .env.prod

# Set restrictive permissions
chmod 600 .env.prod
```

### 4. Configure Domain

Edit `Caddyfile.prod` and replace `yourdomain.com` with your actual domain:

```bash
nano Caddyfile.prod
```

Update the email address for Let's Encrypt notifications.

### 5. Run Database Migrations

```bash
# Start database first
docker-compose -f docker-compose.prod.yml up -d db

# Wait for database to be ready
sleep 10

# Run migrations
bash scripts/migrate.sh
```

### 6. Deploy Application

```bash
# Deploy using automated script
bash scripts/deploy.sh

# Or manually:
docker-compose -f docker-compose.prod.yml build
docker-compose -f docker-compose.prod.yml up -d
```

### 7. Verify Deployment

```bash
# Check service status
docker-compose -f docker-compose.prod.yml ps

# Check API health
curl http://localhost:8080/health

# Check API readiness
curl http://localhost:8080/ready

# Check web service
curl http://localhost/

# View logs
docker-compose -f docker-compose.prod.yml logs -f
```

## Environment Configuration

### Required Environment Variables

```bash
# Database
DB_NAME=loyalty
DB_USER=postgres
DB_PASSWORD=<generated-strong-password>

# JWT Authentication
JWT_SECRET=<generated-jwt-secret>

# HMAC Webhook Verification
HMAC_KEYS_JSON='{"primary":"<key1>","secondary":"<key2>"}'

# Application
PORT=8080
GIN_MODE=release
LOG_LEVEL=info
LOG_FORMAT=json

# WhatsApp (from Meta App Dashboard)
WHATSAPP_VERIFY_TOKEN=<your-verify-token>
WHATSAPP_APP_SECRET=<from-meta-dashboard>
WHATSAPP_ACCESS_TOKEN=<from-meta-dashboard>
WHATSAPP_PHONE_ID=<from-meta-business-account>
WHATSAPP_API_VERSION=v18.0

# Domain
DOMAIN=yourdomain.com
ADMIN_EMAIL=admin@yourdomain.com
```

### Optional Environment Variables

```bash
# S3 Backups
S3_BUCKET=loyalty-backups
S3_ACCESS_KEY=<your-access-key>
S3_SECRET_KEY=<your-secret-key>
S3_REGION=us-east-1

# Email Notifications
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=notifications@yourdomain.com
SMTP_PASSWORD=<your-smtp-password>

# Monitoring
SENTRY_DSN=<your-sentry-dsn>
```

## SSL/TLS Configuration

The platform uses Caddy for automatic SSL/TLS certificate management via Let's Encrypt.

### Automatic HTTPS (Recommended)

1. Ensure your domain DNS is pointing to your server
2. Caddy will automatically obtain and renew certificates
3. Certificates are stored in the `caddy_data` volume

### Manual Certificate Configuration

If you have your own certificates:

```bash
# Edit Caddyfile.prod
nano Caddyfile.prod

# Add tls directive with certificate paths
yourdomain.com {
    tls /path/to/cert.pem /path/to/key.pem
    # ... rest of configuration
}
```

### Certificate Renewal

Caddy automatically renews certificates. To check certificate status:

```bash
docker exec loyalty-caddy-prod caddy list-certificates
```

## Database Setup

### Initial Database Creation

The database is automatically created when the PostgreSQL container starts.

### Running Migrations

```bash
# Dry run (shows pending migrations without applying)
bash scripts/migrate.sh --dry-run

# Apply migrations
bash scripts/migrate.sh
```

### Database Backups

```bash
# Manual backup
bash scripts/backup.sh

# Automated backups (via cron)
bash scripts/setup-cron.sh
```

### Database Restoration

```bash
# List available backups
ls -lh backups/

# Restore from backup
bash scripts/restore.sh backups/loyalty_20231114_120000.dump.gz
```

## Monitoring Setup

### Application Logs

```bash
# View all logs
docker-compose -f docker-compose.prod.yml logs -f

# View specific service logs
docker-compose -f docker-compose.prod.yml logs -f api
docker-compose -f docker-compose.prod.yml logs -f db

# View last 100 lines
docker-compose -f docker-compose.prod.yml logs --tail=100 api
```

### Health Checks

```bash
# API health check
curl http://localhost:8080/health

# API readiness check
curl http://localhost:8080/ready

# Database health check
bash scripts/db-check.sh
```

### Metrics

Application metrics are logged in JSON format and can be sent to monitoring systems:

```bash
# View metrics in logs
docker-compose -f docker-compose.prod.yml logs api | grep metrics
```

## Scheduled Tasks

### Set Up Cron Jobs

```bash
# Set up automated tasks
bash scripts/setup-cron.sh
```

This sets up:
- Daily database backups (2:00 AM)
- Weekly database maintenance (Sunday 3:00 AM)
- Hourly reward expiry checks
- Monthly budget reset (1st at midnight)
- Log cleanup (daily)

### Manual Task Execution

```bash
# Database backup
bash scripts/backup.sh

# Database maintenance
bash scripts/db-maintenance.sh

# Database health check
bash scripts/db-check.sh
```

## Troubleshooting

### Services Won't Start

```bash
# Check Docker status
sudo systemctl status docker

# Check service logs
docker-compose -f docker-compose.prod.yml logs

# Verify environment file
cat .env.prod

# Check disk space
df -h

# Check memory
free -h
```

### Database Connection Issues

```bash
# Check database is running
docker-compose -f docker-compose.prod.yml ps db

# Check database logs
docker-compose -f docker-compose.prod.yml logs db

# Test database connection
docker exec loyalty-db-prod pg_isready -U postgres

# Connect to database
docker exec -it loyalty-db-prod psql -U postgres -d loyalty
```

### SSL Certificate Issues

```bash
# Check Caddy logs
docker-compose -f docker-compose.prod.yml logs caddy

# Verify DNS is pointing to server
dig yourdomain.com

# Check if port 80 and 443 are accessible
sudo netstat -tlnp | grep -E ':(80|443)'
```

### Performance Issues

```bash
# Check resource usage
docker stats

# Check database performance
bash scripts/db-check.sh

# Check slow queries
docker exec loyalty-db-prod psql -U postgres -d loyalty -c "
  SELECT pid, query, state, age(clock_timestamp(), query_start)
  FROM pg_stat_activity
  WHERE state != 'idle'
  ORDER BY query_start;
"
```

## Rollback Procedures

### Rollback to Previous Version

```bash
# Rollback with database restore
bash scripts/rollback.sh --backup backups/loyalty_20231114_120000.dump.gz --git-ref v1.0.0

# Rollback code only
bash scripts/rollback.sh --git-ref v1.0.0

# Rollback database only
bash scripts/restore.sh backups/loyalty_20231114_120000.dump.gz
```

### Emergency Rollback

If automated rollback fails:

```bash
# 1. Stop all services
docker-compose -f docker-compose.prod.yml down

# 2. Checkout previous version
git checkout <previous-tag-or-commit>

# 3. Restore database
bash scripts/restore.sh <backup-file>

# 4. Rebuild and restart
docker-compose -f docker-compose.prod.yml build
docker-compose -f docker-compose.prod.yml up -d

# 5. Verify services
bash scripts/db-check.sh
curl http://localhost:8080/health
```

## Update Procedures

### Regular Updates

```bash
# 1. Create backup
bash scripts/backup.sh

# 2. Pull latest code
git pull origin main

# 3. Deploy updates
bash scripts/deploy.sh

# 4. Verify deployment
curl http://localhost:8080/health
```

### Zero-Downtime Updates

The `deploy.sh` script performs rolling updates:

```bash
bash scripts/deploy.sh
```

This automatically:
- Creates a backup
- Builds new images
- Runs migrations
- Performs rolling restart
- Verifies health checks

## Security Checklist

- [ ] Strong passwords generated for all services
- [ ] .env.prod file has restrictive permissions (600)
- [ ] Firewall configured (only ports 22, 80, 443 open)
- [ ] SSH key authentication enabled
- [ ] fail2ban configured
- [ ] SSL/TLS certificates configured
- [ ] Regular backups scheduled
- [ ] Log rotation configured
- [ ] Security headers enabled (via Caddy)
- [ ] Rate limiting configured
- [ ] Database connections encrypted
- [ ] Secrets not in version control

## Production Readiness Checklist

- [ ] Server setup complete
- [ ] Application deployed
- [ ] Database migrations run
- [ ] SSL certificates configured
- [ ] Environment variables configured
- [ ] Backups scheduled
- [ ] Monitoring configured
- [ ] Logging configured
- [ ] Health checks passing
- [ ] Performance tested
- [ ] Security audit completed
- [ ] Documentation reviewed
- [ ] Runbook created
- [ ] Team trained

## Support

For deployment issues:
- Check logs: `docker-compose logs`
- Review documentation: `/docs`
- Contact: support@yourdomain.com
