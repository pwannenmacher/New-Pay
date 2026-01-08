#!/bin/bash

# ============================================================================
# NewPay Production Environment Setup Script
# ============================================================================
# This script guides you through the setup of a production NewPay instance.
# It will:
# - Generate secure secrets and keys
# - Create a production .env file
# - Configure SMTP settings
# - Initialize HashiCorp Vault
# - Set up OAuth providers (optional)
# - Configure LLM settings (optional)
# ============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_header() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

ask_question() {
    local question="$1"
    local default="$2"
    local response
    
    if [ -n "$default" ]; then
        echo -ne "${GREEN}?${NC} $question [${YELLOW}$default${NC}]: " >&2
        read response
        echo "${response:-$default}"
    else
        echo -ne "${GREEN}?${NC} $question: " >&2
        read response
        echo "$response"
    fi
}

ask_password() {
    local question="$1"
    local response
    
    echo -ne "${GREEN}?${NC} $question: " >&2
    read -s response
    echo "" >&2
    echo "$response"
}

ask_yes_no() {
    local question="$1"
    local default="${2:-n}"
    local response
    
    if [ "$default" = "y" ]; then
        echo -ne "${GREEN}?${NC} $question [${YELLOW}Y/n${NC}]: " >&2
        read response
        response=${response:-y}
    else
        echo -ne "${GREEN}?${NC} $question [${YELLOW}y/N${NC}]: " >&2
        read response
        response=${response:-n}
    fi
    
    [[ "$response" =~ ^[Yy]$ ]]
}

generate_random_string() {
    local length="${1:-32}"
    openssl rand -base64 "$length" | tr -d "=+/" | cut -c1-"$length"
}

generate_jwt_key() {
    openssl ecparam -genkey -name prime256v1 -noout | openssl ec -outform PEM 2>/dev/null | awk '{printf "%s\\n", $0}'
}

# Check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"
    
    local missing_tools=()
    
    command -v openssl >/dev/null 2>&1 || missing_tools+=("openssl")
    command -v docker >/dev/null 2>&1 || missing_tools+=("docker")
    command -v docker compose >/dev/null 2>&1 || missing_tools+=("docker compose")
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        print_error "Missing required tools: ${missing_tools[*]}"
        exit 1
    fi
    
    print_success "All prerequisites met"
}

# Main setup
main() {
    clear
    echo -e "${GREEN}"
    cat << "EOF"
    _   __               ____              
   / | / /__ _      __  / __ \____ ___  __
  /  |/ / _ \ | /| / / / /_/ / __ `/ / / /
 / /|  /  __/ |/ |/ / / ____/ /_/ / /_/ / 
/_/ |_/\___/|__/|__/ /_/    \__,_/\__, /  
                                 /____/   
    Production Environment Setup
EOF
    echo -e "${NC}"
    
    print_warning "This script will create a production .env file with secure credentials."
    print_warning "Make sure you're in a secure environment!"
    echo ""
    
    if ! ask_yes_no "Do you want to continue?" "y"; then
        print_info "Setup cancelled."
        exit 0
    fi
    
    check_prerequisites
    
    # Check if .env already exists
    if [ -f .env ]; then
        print_warning "A .env file already exists!"
        if ! ask_yes_no "Do you want to overwrite it? (Backup will be created)" "n"; then
            print_info "Setup cancelled."
            exit 0
        fi
        mv .env ".env.backup.$(date +%Y%m%d_%H%M%S)"
        print_success "Existing .env backed up"
    fi
    
    # Start collecting configuration
    print_header "Basic Configuration"
    
    DOMAIN=$(ask_question "Enter your domain (e.g., newpay.example.com)" "localhost")
    APP_NAME=$(ask_question "Application name" "NewPay")
    
    # Database Configuration
    print_header "Database Configuration"
    
    DB_USER=$(ask_question "Database username" "newpay_prod")
    DB_PASSWORD=$(ask_password "Database password (leave empty to generate)")
    [ -z "$DB_PASSWORD" ] && DB_PASSWORD=$(generate_random_string 32)
    DB_NAME=$(ask_question "Database name" "newpay_prod")
    
    # JWT Configuration
    print_header "JWT Configuration"
    
    print_info "Generating secure JWT key (ECDSA P-256)..."
    JWT_SECRET=$(generate_jwt_key)
    print_success "JWT key generated"
    
    JWT_EXPIRATION=$(ask_question "JWT expiration time" "24h")
    JWT_REFRESH_EXPIRATION=$(ask_question "JWT refresh expiration time" "168h")
    SESSION_TIMEOUT=$(ask_question "Session timeout" "30m")
    
    # SMTP Configuration
    print_header "Email (SMTP) Configuration"
    
    SMTP_HOST=$(ask_question "SMTP host" "smtp.example.com")
    SMTP_PORT=$(ask_question "SMTP port" "587")
    SMTP_USERNAME=$(ask_question "SMTP username" "noreply@${DOMAIN}")
    SMTP_PASSWORD=$(ask_password "SMTP password")
    SMTP_FROM=$(ask_question "Email from address" "noreply@${DOMAIN}")
    
    PROTOCOL="https"
    if [ "$DOMAIN" = "localhost" ]; then
        PROTOCOL="http"
    fi
    
    EMAIL_VERIFICATION_URL="${PROTOCOL}://${DOMAIN}/verify-email"
    PASSWORD_RESET_URL="${PROTOCOL}://${DOMAIN}/reset-password"
    
    # CORS Configuration
    print_header "CORS Configuration"
    
    if [ "$DOMAIN" = "localhost" ]; then
        CORS_ORIGINS="http://localhost:3000,http://localhost:5173,http://localhost"
    else
        CORS_ORIGINS="${PROTOCOL}://${DOMAIN}"
    fi
    CORS_ORIGINS=$(ask_question "Allowed CORS origins (comma-separated)" "$CORS_ORIGINS")
    
    # OAuth Configuration
    print_header "OAuth/SSO Configuration"
    
    OAUTH_ENABLED="false"
    if ask_yes_no "Do you want to configure OAuth/SSO providers?" "n"; then
        OAUTH_ENABLED="true"
        OAUTH_REDIRECT_URL="${PROTOCOL}://${DOMAIN}/api/v1/auth/oauth/callback"
        OAUTH_FRONTEND_CALLBACK_URL="${PROTOCOL}://${DOMAIN}/oauth/callback"
        
        # Provider 1
        if ask_yes_no "Configure OAuth Provider 1?" "y"; then
            OAUTH_1_ENABLED="true"
            OAUTH_1_NAME=$(ask_question "Provider 1 name" "GitLab")
            OAUTH_1_CLIENT_ID=$(ask_question "Provider 1 Client ID")
            OAUTH_1_CLIENT_SECRET=$(ask_password "Provider 1 Client Secret")
            OAUTH_1_AUTH_URL=$(ask_question "Provider 1 Auth URL" "https://gitlab.com/oauth/authorize")
            OAUTH_1_TOKEN_URL=$(ask_question "Provider 1 Token URL" "https://gitlab.com/oauth/token")
            OAUTH_1_USER_INFO_URL=$(ask_question "Provider 1 User Info URL" "https://gitlab.com/api/v4/user")
            OAUTH_1_GROUP_MAPPING=$(ask_question "Provider 1 Group Mapping (optional, e.g., admin-group:admin,user-group:user)" "")
        else
            OAUTH_1_ENABLED="false"
        fi
    fi
    
    # Registration Settings
    print_header "User Registration"
    
    if ask_yes_no "Enable email/password registration?" "n"; then
        ENABLE_REGISTRATION="true"
    else
        ENABLE_REGISTRATION="false"
    fi
    
    if [ "$OAUTH_ENABLED" = "true" ]; then
        if ask_yes_no "Enable OAuth registration?" "y"; then
            ENABLE_OAUTH_REGISTRATION="true"
        else
            ENABLE_OAUTH_REGISTRATION="false"
        fi
    else
        ENABLE_OAUTH_REGISTRATION="false"
    fi
    
    # Scheduler Configuration
    print_header "Scheduler Configuration"
    
    if ask_yes_no "Enable draft reminder emails?" "y"; then
        SCHEDULER_ENABLE_DRAFT_REMINDERS="true"
        SCHEDULER_DRAFT_REMINDER_CRON=$(ask_question "Draft reminder cron (Monday 9 AM)" "0 9 * * 1")
    else
        SCHEDULER_ENABLE_DRAFT_REMINDERS="false"
        SCHEDULER_DRAFT_REMINDER_CRON="0 9 * * 1"
    fi
    
    if ask_yes_no "Enable reviewer summary emails?" "y"; then
        SCHEDULER_ENABLE_REVIEWER_SUMMARY="true"
        SCHEDULER_REVIEWER_SUMMARY_CRON=$(ask_question "Reviewer summary cron (Daily 8 AM)" "0 8 * * *")
    else
        SCHEDULER_ENABLE_REVIEWER_SUMMARY="false"
        SCHEDULER_REVIEWER_SUMMARY_CRON="0 8 * * *"
    fi
    
    if ask_yes_no "Enable hash chain validation?" "y"; then
        SCHEDULER_ENABLE_HASH_CHAIN_VALIDATION="true"
        SCHEDULER_HASH_CHAIN_VALIDATION_CRON=$(ask_question "Hash chain validation cron (Daily 3 AM)" "0 3 * * *")
    else
        SCHEDULER_ENABLE_HASH_CHAIN_VALIDATION="false"
        SCHEDULER_HASH_CHAIN_VALIDATION_CRON="0 3 * * *"
    fi
    
    SCHEDULER_REMINDER_INTERVAL_MINS=$(ask_question "Draft reminder interval (minutes, 10080=7 days)" "10080")
    
    # LLM Configuration
    print_header "LLM Configuration (Optional)"
    
    if ask_yes_no "Enable LLM/AI features (requires Ollama, approximately 4.5 GB Model data will be downloaded if llama3 is selected)?" "n"; then
        LLM_ENABLED="true"
        LLM_MODEL=$(ask_question "LLM model name" "llama3")
        print_info "Note: The LLM model will be downloaded after Vault initialization."
    else
        LLM_ENABLED="false"
        LLM_MODEL="llama3"
    fi
    
    # Rate Limiting
    print_header "Rate Limiting"
    
    RATE_LIMIT_REQUESTS=$(ask_question "Rate limit requests per window" "1000")
    RATE_LIMIT_DURATION=$(ask_question "Rate limit window duration" "1m")
    
    # Create .env file
    print_header "Creating .env File"
    
    cat > .env << EOF
# ============================================================================
# NewPay Production Environment Configuration
# Generated: $(date)
# ============================================================================

# -----------------------------------------------------------------------------
# Database Configuration
# -----------------------------------------------------------------------------
DB_HOST=postgres
DB_PORT=5432
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=${DB_NAME}
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

# -----------------------------------------------------------------------------
# Server Configuration
# -----------------------------------------------------------------------------
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_TIMEOUT_READ=15s
SERVER_TIMEOUT_WRITE=15s
SERVER_TIMEOUT_IDLE=60s

# -----------------------------------------------------------------------------
# Application
# -----------------------------------------------------------------------------
APP_ENV=production
APP_NAME=${APP_NAME}
APP_VERSION=1.0.0
LOG_LEVEL=info

# -----------------------------------------------------------------------------
# JWT Configuration
# -----------------------------------------------------------------------------
JWT_SECRET=${JWT_SECRET}
JWT_EXPIRATION=${JWT_EXPIRATION}
JWT_REFRESH_EXPIRATION=${JWT_REFRESH_EXPIRATION}

# -----------------------------------------------------------------------------
# Session Configuration
# -----------------------------------------------------------------------------
SESSION_TIMEOUT=${SESSION_TIMEOUT}

# -----------------------------------------------------------------------------
# Email Configuration
# -----------------------------------------------------------------------------
SMTP_HOST=${SMTP_HOST}
SMTP_PORT=${SMTP_PORT}
SMTP_USERNAME=${SMTP_USERNAME}
SMTP_PASSWORD=${SMTP_PASSWORD}
SMTP_FROM=${SMTP_FROM}
EMAIL_VERIFICATION_URL=${EMAIL_VERIFICATION_URL}
PASSWORD_RESET_URL=${PASSWORD_RESET_URL}

# -----------------------------------------------------------------------------
# OAuth/SSO Configuration
# -----------------------------------------------------------------------------
EOF

    if [ "$OAUTH_ENABLED" = "true" ]; then
        cat >> .env << EOF
OAUTH_REDIRECT_URL=${OAUTH_REDIRECT_URL}
OAUTH_FRONTEND_CALLBACK_URL=${OAUTH_FRONTEND_CALLBACK_URL}

OAUTH_1_ENABLED=${OAUTH_1_ENABLED:-false}
EOF
        if [ "${OAUTH_1_ENABLED}" = "true" ]; then
            cat >> .env << EOF
OAUTH_1_NAME=${OAUTH_1_NAME}
OAUTH_1_CLIENT_ID=${OAUTH_1_CLIENT_ID}
OAUTH_1_CLIENT_SECRET=${OAUTH_1_CLIENT_SECRET}
OAUTH_1_AUTH_URL=${OAUTH_1_AUTH_URL}
OAUTH_1_TOKEN_URL=${OAUTH_1_TOKEN_URL}
OAUTH_1_USER_INFO_URL=${OAUTH_1_USER_INFO_URL}
OAUTH_1_GROUP_MAPPING=${OAUTH_1_GROUP_MAPPING}
OAUTH_1_GROUPS_CLAIM=groups
EOF
        fi
    else
        cat >> .env << EOF
# OAuth disabled
EOF
    fi
    
    cat >> .env << EOF

# -----------------------------------------------------------------------------
# CORS Configuration
# -----------------------------------------------------------------------------
CORS_ALLOWED_ORIGINS=${CORS_ORIGINS}
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Accept,Authorization,Content-Type,X-CSRF-Token
CORS_EXPOSED_HEADERS=Link
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=300

# -----------------------------------------------------------------------------
# Rate Limiting
# -----------------------------------------------------------------------------
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=${RATE_LIMIT_REQUESTS}
RATE_LIMIT_DURATION=${RATE_LIMIT_DURATION}

# -----------------------------------------------------------------------------
# Registration Settings
# -----------------------------------------------------------------------------
ENABLE_REGISTRATION=${ENABLE_REGISTRATION}
ENABLE_OAUTH_REGISTRATION=${ENABLE_OAUTH_REGISTRATION}

# -----------------------------------------------------------------------------
# Scheduler Configuration
# -----------------------------------------------------------------------------
SCHEDULER_ENABLE_DRAFT_REMINDERS=${SCHEDULER_ENABLE_DRAFT_REMINDERS}
SCHEDULER_ENABLE_REVIEWER_SUMMARY=${SCHEDULER_ENABLE_REVIEWER_SUMMARY}
SCHEDULER_ENABLE_HASH_CHAIN_VALIDATION=${SCHEDULER_ENABLE_HASH_CHAIN_VALIDATION}
SCHEDULER_DRAFT_REMINDER_CRON=${SCHEDULER_DRAFT_REMINDER_CRON}
SCHEDULER_REVIEWER_SUMMARY_CRON=${SCHEDULER_REVIEWER_SUMMARY_CRON}
SCHEDULER_HASH_CHAIN_VALIDATION_CRON=${SCHEDULER_HASH_CHAIN_VALIDATION_CRON}
SCHEDULER_REMINDER_INTERVAL_MINS=${SCHEDULER_REMINDER_INTERVAL_MINS}

# -----------------------------------------------------------------------------
# Vault Configuration
# -----------------------------------------------------------------------------
# NOTE: VAULT_TOKEN will be set after Vault initialization
VAULT_ENABLED=true
VAULT_ADDR=http://vault:8200
VAULT_TOKEN=PLACEHOLDER_WILL_BE_SET_BY_VAULT_INIT
VAULT_TRANSIT_MOUNT=transit

# -----------------------------------------------------------------------------
# LLM Configuration
# -----------------------------------------------------------------------------
LLM_ENABLED=${LLM_ENABLED}
LLM_BASE_URL=http://ollama:11434
LLM_MODEL=${LLM_MODEL}

# -----------------------------------------------------------------------------
# Frontend Configuration
# -----------------------------------------------------------------------------
VITE_API_BASE_URL=/api/v1
EOF

    print_success ".env file created"
    
    # Pull Images
    print_header "Pulling Docker Images"
    docker compose pull
    print_success "Docker images pulled successfully"

    # Build Images
    print_header "Building Docker Images"
    docker compose build frontend api

    # Vault Initialization
    print_header "HashiCorp Vault Initialization"
    
    print_info "Starting Vault service for initialization..."
    docker compose up -d vault

    print_info "Waiting for Vault to be ready..."
    sleep 15

    print_info "Fixing Vault Volume permissions..."
    docker compose exec -T vault chown -R vault:vault /vault/data /vault/logs /vault/file
    
    # Check if Vault is already initialized
    if docker compose exec -T vault vault status 2>/dev/null | grep -q "Initialized.*false"; then
        print_info "Initializing Vault..."
        
        # Initialize Vault and capture output
        VAULT_INIT_OUTPUT=$(docker compose exec -T vault vault operator init -key-shares=1 -key-threshold=1 -format=json)
        
        # Parse JSON output using Python (more reliable than grep)
        if command -v python3 >/dev/null 2>&1; then
            UNSEAL_KEY=$(echo "$VAULT_INIT_OUTPUT" | python3 -c "import sys, json; print(json.load(sys.stdin)['unseal_keys_b64'][0])")
            ROOT_TOKEN=$(echo "$VAULT_INIT_OUTPUT" | python3 -c "import sys, json; print(json.load(sys.stdin)['root_token'])")
        else
            # Fallback to sed/awk parsing
            UNSEAL_KEY=$(echo "$VAULT_INIT_OUTPUT" | sed -n 's/.*"unseal_keys_b64":\["\([^"]*\)".*/\1/p')
            ROOT_TOKEN=$(echo "$VAULT_INIT_OUTPUT" | sed -n 's/.*"root_token":"\([^"]*\)".*/\1/p')
        fi
        
        if [ -z "$UNSEAL_KEY" ] || [ -z "$ROOT_TOKEN" ]; then
            print_error "Failed to parse Vault initialization output"
            echo "Raw output:" >&2
            echo "$VAULT_INIT_OUTPUT" >&2
            exit 1
        fi
        
        print_success "Vault initialized"
        print_info "Unsealing Vault..."

        # Unseal Vault
        docker compose exec -T vault vault operator unseal "$UNSEAL_KEY" > /dev/null
        
        # Update .env with Vault token
        sed -i.bak "s/VAULT_TOKEN=PLACEHOLDER_WILL_BE_SET_BY_VAULT_INIT/VAULT_TOKEN=${ROOT_TOKEN}/" .env
        rm -f .env.bak
        
        # Save Vault credentials securely
        cat > vault-credentials.txt << EOF
# ============================================================================
# VAULT CREDENTIALS - KEEP THIS FILE SECURE!
# ============================================================================
# Generated: $(date)
#
# IMPORTANT: Store these credentials in a secure location (e.g., password manager)
# and delete this file after storing them safely.
# ============================================================================

Unseal Key: ${UNSEAL_KEY}
Root Token: ${ROOT_TOKEN}

# To unseal Vault manually:
# docker compose exec vault vault operator unseal ${UNSEAL_KEY}

# To login to Vault:
# docker compose exec vault vault login ${ROOT_TOKEN}
EOF
        
        chmod 600 vault-credentials.txt
        
        print_success "Vault initialized successfully"
        print_warning "Vault credentials saved to: vault-credentials.txt"
        print_warning "IMPORTANT: Store these credentials securely and delete the file!"
        
        # Create auto-unseal script
        print_info "Creating auto-unseal script..."
        cat > auto-unseal.sh << 'EOFSCRIPT'
#!/bin/bash
# ============================================================================
# Vault Auto-Unseal Script for NewPay Production
# ============================================================================
# WARNUNG: Dieses Skript enthält sensible Vault-Credentials!
# - Nur in sicherer Umgebung verwenden
# - Berechtigungen: chmod 600 auto-unseal.sh
# - Niemals in Git committen
# ============================================================================

set -e

# Vault Configuration
VAULT_ADDR="${VAULT_ADDR:-http://localhost:8200}"
CONTAINER_NAME="${VAULT_CONTAINER:-newpay-vault-prod}"

# Vault Credentials (auto-generated by setup.sh)
EOFSCRIPT
        echo "UNSEAL_KEY=\"${UNSEAL_KEY}\"" >> auto-unseal.sh
        echo "ROOT_TOKEN=\"${ROOT_TOKEN}\"" >> auto-unseal.sh
        cat >> auto-unseal.sh << 'EOFSCRIPT'

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_info "Starting Vault auto-unseal process..."

# Check if container is running
if ! docker ps | grep -q "$CONTAINER_NAME"; then
    print_error "Vault container is not running: $CONTAINER_NAME"
    print_info "Start it with: docker compose up -d vault"
    exit 1
else
    print_info "Vault container is running: $CONTAINER_NAME"
fi

STATUS=$(docker compose exec -T vault vault status -format=json 2>&1) || true

SEALED=$(echo "$STATUS" | jq -r '.sealed // true' 2>/dev/null)

print_info "Vault Sealed: $SEALED"

INITIALIZED=$(echo "$STATUS" | jq -r '.initialized // false' 2>/dev/null)

print_info "Vault Initialized: $INITIALIZED"

# Check if Vault is initialized
if [ "$INITIALIZED" = "False" ] || [ "$INITIALIZED" = "false" ]; then
    print_error "Vault is not initialized"
    print_info "Initialize it with: docker compose exec -it vault vault operator init"
    exit 1
fi

# Check if already unsealed
if [ "$SEALED" = "False" ] || [ "$SEALED" = "false" ]; then
    print_success "Vault is already unsealed"
    exit 0
fi

# Unseal Vault
print_info "Unsealing Vault..."
if docker compose exec -T vault vault operator unseal "$UNSEAL_KEY" > /dev/null 2>&1; then
    print_success "Vault unsealed successfully"
else
    print_error "Failed to unseal Vault"
    exit 1
fi

# Verify unsealed status
sleep 1
STATUS=$(docker compose exec -T vault vault status -format=json 2>/dev/null || echo '{}')
SEALED=$(echo "$STATUS" | python3 -c "import sys, json; print(json.load(sys.stdin).get('sealed', True))" 2>/dev/null || echo "true")

if [ "$SEALED" = "False" ] || [ "$SEALED" = "false" ]; then
    print_success "Vault is now operational"
else
    print_error "Vault unsealing verification failed"
    exit 1
fi
EOFSCRIPT
        
        chmod 600 auto-unseal.sh
        chmod +x auto-unseal.sh
        print_success "Auto-unseal script created: auto-unseal.sh"
        print_warning "IMPORTANT: This script contains sensitive credentials!"
        
        # Enable transit engine
        print_info "Enabling Vault transit engine for encryption..."
        docker compose exec -T vault sh -c "export VAULT_TOKEN=${ROOT_TOKEN} && vault secrets enable transit" > /dev/null 2>&1 || true
        print_success "Transit engine enabled"
        
    else
        print_warning "Vault appears to be already initialized"
        print_info "Please set VAULT_TOKEN manually in .env file"
    fi
    
    # Download LLM model if enabled
    if [ "$LLM_ENABLED" = "true" ]; then
        print_header "LLM Model Download"
        
        print_info "Starting Ollama service..."
        docker compose up -d ollama
        
        print_info "Waiting for Ollama to be ready..."
        sleep 10
        
        print_info "Downloading LLM model: ${LLM_MODEL} (this may take several minutes)..."
        if docker compose exec -T ollama ollama pull "${LLM_MODEL}"; then
            print_success "LLM model ${LLM_MODEL} downloaded successfully"
        else
            print_warning "Failed to download LLM model. You can pull it manually later with:"
            print_warning "docker compose exec ollama ollama pull ${LLM_MODEL}"
        fi
    fi
    
    # Stop Vault
    docker compose down
    
    # Final summary
    print_header "Setup Complete!"
    
    echo ""
    print_success "Production environment configured successfully!"
    echo ""
    print_info "Summary:"
    echo "  • Domain: ${DOMAIN}"
    echo "  • Database: ${DB_NAME} (user: ${DB_USER})"
    echo "  • JWT: Secure ECDSA key generated"
    echo "  • SMTP: ${SMTP_HOST}:${SMTP_PORT}"
    echo "  • OAuth: ${OAUTH_ENABLED}"
    echo "  • LLM: ${LLM_ENABLED}"
    echo "  • Vault: Initialized with transit engine"
    echo ""
    print_info "Next steps:"
    echo "  1. Review the .env file and make any necessary adjustments"
    echo "  2. Securely store Vault credentials from vault-credentials.txt"
    echo "  3. Delete vault-credentials.txt after storing credentials"
    echo "  4. Start the stack: docker compose up -d"
    echo "  5. Access the application at: ${PROTOCOL}://${DOMAIN}"
    echo ""
    print_warning "Security reminders:"
    echo "  • Never commit .env, vault-credentials.txt, or auto-unseal.sh to version control"
    echo "  • Regularly backup your Vault unseal key and root token"
    echo "  • auto-unseal.sh contains sensitive credentials - protect it (chmod 600 set)"
    echo "  • Set up Vault auto-unseal for production (see vault-config.hcl)"
    echo "  • Consider using a proper SSL/TLS certificate for ${DOMAIN}"
    echo ""
}

# Run main function
main
