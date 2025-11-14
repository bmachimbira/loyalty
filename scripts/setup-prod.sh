#!/bin/bash

###########################################
# Production Server Setup Script
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

# Logging
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

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    log_error "This script must be run as root"
    log "Run with: sudo bash $0"
    exit 1
fi

log "======================================"
log "  Production Server Setup"
log "======================================"
log "Zimbabwe Loyalty Platform"
echo ""

# Detect OS
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$NAME
    VER=$VERSION_ID
else
    log_error "Cannot detect OS"
    exit 1
fi

log "Detected OS: $OS $VER"
echo ""

# Update system
log "Updating system packages..."
if [[ "$OS" == *"Ubuntu"* ]] || [[ "$OS" == *"Debian"* ]]; then
    apt-get update
    apt-get upgrade -y
    log_success "✓ System updated"
elif [[ "$OS" == *"CentOS"* ]] || [[ "$OS" == *"Red Hat"* ]]; then
    yum update -y
    log_success "✓ System updated"
else
    log_warning "Unsupported OS for automatic setup"
fi

# Install Docker if not installed
if ! command -v docker &> /dev/null; then
    log "Installing Docker..."

    if [[ "$OS" == *"Ubuntu"* ]] || [[ "$OS" == *"Debian"* ]]; then
        # Install prerequisites
        apt-get install -y apt-transport-https ca-certificates curl software-properties-common gnupg

        # Add Docker GPG key
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -

        # Add Docker repository
        add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"

        # Install Docker
        apt-get update
        apt-get install -y docker-ce docker-ce-cli containerd.io

        log_success "✓ Docker installed"
    else
        log_error "Please install Docker manually for your OS"
        log "Visit: https://docs.docker.com/engine/install/"
        exit 1
    fi
else
    log_success "✓ Docker already installed"
fi

# Start and enable Docker
systemctl start docker
systemctl enable docker
log_success "✓ Docker service enabled"

# Install Docker Compose if not installed
if ! command -v docker-compose &> /dev/null; then
    log "Installing Docker Compose..."

    DOCKER_COMPOSE_VERSION="2.20.2"
    curl -L "https://github.com/docker/compose/releases/download/v${DOCKER_COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose

    log_success "✓ Docker Compose installed"
else
    log_success "✓ Docker Compose already installed"
fi

# Install additional tools
log "Installing additional tools..."
if [[ "$OS" == *"Ubuntu"* ]] || [[ "$OS" == *"Debian"* ]]; then
    apt-get install -y git curl wget htop vim ufw fail2ban logrotate
    log_success "✓ Additional tools installed"
fi

# Configure firewall
log "Configuring firewall..."
if command -v ufw &> /dev/null; then
    # Allow SSH
    ufw allow 22/tcp
    # Allow HTTP
    ufw allow 80/tcp
    # Allow HTTPS
    ufw allow 443/tcp
    # Enable firewall
    ufw --force enable
    log_success "✓ Firewall configured"
else
    log_warning "UFW not available, configure firewall manually"
fi

# Configure fail2ban
log "Configuring fail2ban..."
if command -v fail2ban-client &> /dev/null; then
    systemctl enable fail2ban
    systemctl start fail2ban
    log_success "✓ fail2ban configured"
fi

# Create deployment user
if ! id -u deploy &> /dev/null; then
    log "Creating deployment user..."
    useradd -m -s /bin/bash deploy
    usermod -aG docker deploy
    log_success "✓ Deployment user created"
else
    log "✓ Deployment user already exists"
fi

# Create project directory
PROJECT_DIR="/opt/loyalty"
log "Creating project directory: $PROJECT_DIR"
mkdir -p "$PROJECT_DIR"
chown -R deploy:deploy "$PROJECT_DIR"
log_success "✓ Project directory created"

# Set up log rotation
log "Setting up log rotation..."
cat > /etc/logrotate.d/loyalty <<EOF
$PROJECT_DIR/logs/*.log {
    daily
    missingok
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 deploy deploy
    sharedscripts
}
EOF
log_success "✓ Log rotation configured"

# Set up swap if not exists
if [ ! -f /swapfile ]; then
    log "Creating swap file (2GB)..."
    fallocate -l 2G /swapfile
    chmod 600 /swapfile
    mkswap /swapfile
    swapon /swapfile
    echo '/swapfile none swap sw 0 0' | tee -a /etc/fstab
    log_success "✓ Swap file created"
else
    log "✓ Swap file already exists"
fi

# Configure kernel parameters for better performance
log "Optimizing kernel parameters..."
cat >> /etc/sysctl.conf <<EOF

# Loyalty Platform optimizations
net.core.somaxconn = 1024
net.ipv4.tcp_max_syn_backlog = 2048
vm.swappiness = 10
vm.dirty_ratio = 60
vm.dirty_background_ratio = 2
EOF
sysctl -p > /dev/null
log_success "✓ Kernel parameters optimized"

# Summary
log ""
log "======================================"
log_success "  Production Server Setup Complete"
log "======================================"
echo ""
log "Installation summary:"
log "  ✓ System updated"
log "  ✓ Docker installed and configured"
log "  ✓ Docker Compose installed"
log "  ✓ Firewall configured (ports 22, 80, 443)"
log "  ✓ fail2ban configured"
log "  ✓ Deployment user created (deploy)"
log "  ✓ Project directory created ($PROJECT_DIR)"
log "  ✓ Log rotation configured"
log "  ✓ Swap configured (2GB)"
log "  ✓ Kernel parameters optimized"
echo ""
log "Next steps:"
log "  1. Clone the repository to $PROJECT_DIR"
log "  2. Copy .env.prod.example to .env.prod and configure"
log "  3. Generate secrets: bash scripts/generate-secrets.sh"
log "  4. Run deployment: bash scripts/deploy.sh"
log "  5. Set up cron jobs: bash scripts/setup-cron.sh"
log "  6. Configure domain DNS to point to this server"
log "  7. Update Caddyfile.prod with your domain"
echo ""
log "Security recommendations:"
log "  - Change SSH port from default 22"
log "  - Set up SSH key authentication (disable password auth)"
log "  - Configure automatic security updates"
log "  - Set up monitoring and alerting"
log "  - Regular backups to offsite location"
log "  - Keep Docker and all packages updated"
echo ""

exit 0
