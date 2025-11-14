#!/bin/bash

###########################################
# Secrets Generation Script
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
    echo -e "${BLUE}$1${NC}"
}

log_success() {
    echo -e "${GREEN}$1${NC}"
}

log_warning() {
    echo -e "${YELLOW}$1${NC}"
}

log_error() {
    echo -e "${RED}$1${NC}"
}

echo ""
log "======================================"
log "  Secrets Generation Utility"
log "======================================"
echo ""

# Check if openssl is available
if ! command -v openssl &> /dev/null; then
    log_error "Error: openssl is not installed"
    log "Install it with: apt-get install openssl (Ubuntu/Debian) or brew install openssl (macOS)"
    exit 1
fi

# Generate JWT Secret
log "Generating JWT Secret..."
JWT_SECRET=$(openssl rand -base64 48)
log_success "✓ JWT_SECRET generated (64 characters)"
echo ""

# Generate HMAC Keys
log "Generating HMAC Keys..."
HMAC_KEY_PRIMARY=$(openssl rand -base64 32)
HMAC_KEY_SECONDARY=$(openssl rand -base64 32)
HMAC_KEYS_JSON="{\"primary\":\"$HMAC_KEY_PRIMARY\",\"secondary\":\"$HMAC_KEY_SECONDARY\"}"
log_success "✓ HMAC_KEYS_JSON generated (primary and secondary keys)"
echo ""

# Generate Database Password
log "Generating Database Password..."
DB_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)
log_success "✓ DB_PASSWORD generated (32 characters)"
echo ""

# Generate Internal API Secret
log "Generating Internal API Secret..."
INTERNAL_API_SECRET=$(openssl rand -hex 32)
log_success "✓ INTERNAL_API_SECRET generated (64 characters)"
echo ""

# Generate WhatsApp Verify Token
log "Generating WhatsApp Verify Token..."
WHATSAPP_VERIFY_TOKEN=$(openssl rand -hex 24)
log_success "✓ WHATSAPP_VERIFY_TOKEN generated (48 characters)"
echo ""

# Display generated secrets
log "======================================"
log "  Generated Secrets"
log "======================================"
echo ""
echo "Copy these values to your .env.prod file:"
echo ""
echo "# Database"
echo "DB_PASSWORD=$DB_PASSWORD"
echo ""
echo "# JWT Authentication"
echo "JWT_SECRET=$JWT_SECRET"
echo ""
echo "# HMAC Webhook Verification"
echo "HMAC_KEYS_JSON='$HMAC_KEYS_JSON'"
echo ""
echo "# Internal API"
echo "INTERNAL_API_SECRET=$INTERNAL_API_SECRET"
echo ""
echo "# WhatsApp"
echo "WHATSAPP_VERIFY_TOKEN=$WHATSAPP_VERIFY_TOKEN"
echo ""
log "======================================"
echo ""

# Option to save to file
read -p "Save these secrets to a file? (yes/no): " -r
echo

if [[ $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    SECRETS_FILE="secrets-$(date +%Y%m%d_%H%M%S).txt"

    cat > "$SECRETS_FILE" <<EOF
# Zimbabwe Loyalty Platform - Generated Secrets
# Generated on: $(date)
# IMPORTANT: Keep this file secure and never commit it to version control!

# Database
DB_PASSWORD=$DB_PASSWORD

# JWT Authentication
JWT_SECRET=$JWT_SECRET

# HMAC Webhook Verification
HMAC_KEYS_JSON='$HMAC_KEYS_JSON'

# Internal API
INTERNAL_API_SECRET=$INTERNAL_API_SECRET

# WhatsApp
WHATSAPP_VERIFY_TOKEN=$WHATSAPP_VERIFY_TOKEN

# Additional secrets to configure manually:
# WHATSAPP_APP_SECRET=<get from Meta App Dashboard>
# WHATSAPP_ACCESS_TOKEN=<get from Meta App Dashboard>
# WHATSAPP_PHONE_ID=<get from Meta Business Account>

# Optional S3 backup credentials:
# S3_ACCESS_KEY=<your-s3-access-key>
# S3_SECRET_KEY=<your-s3-secret-key>

# Optional email notification:
# SMTP_PASSWORD=<your-smtp-password>
EOF

    log_success "✓ Secrets saved to: $SECRETS_FILE"
    log_warning "⚠ IMPORTANT: This file contains sensitive secrets!"
    log_warning "  - Store it securely (password manager, vault, etc.)"
    log_warning "  - Never commit it to version control"
    log_warning "  - Delete it after copying to your .env.prod file"
    echo ""

    # Set restrictive permissions
    chmod 600 "$SECRETS_FILE"
    log_success "✓ File permissions set to 600 (read/write for owner only)"
fi

echo ""
log "======================================"
log_success "  Secrets Generation Complete"
log "======================================"
echo ""
log "Next steps:"
log "  1. Copy the secrets to your .env.prod file"
log "  2. Configure WhatsApp credentials from Meta Dashboard"
log "  3. Configure any optional services (S3, SMTP, etc.)"
log "  4. Secure your .env.prod file (chmod 600 .env.prod)"
log "  5. Never commit .env.prod to version control"
echo ""

exit 0
