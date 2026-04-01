#!/bin/bash

# ============================================================================
# NewPay Production Environment Setup Script
# ============================================================================
# This script guides you through the setup of a production NewPay instance.
# It will:
# - Generate secure secrets and keys (JWT, Encryption Master Key)
# - Create a production .env file
# - Configure SMTP settings
# - Set up OAuth providers (optional)
# - Configure LLM settings (optional)
# ============================================================================

set -e

# Ensure Docker build output is streamed line-by-line instead of spinner UI,
# which can look like a hang in some terminals.
export BUILDKIT_PROGRESS=plain

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"
}
print_info()    { echo -e "${BLUE}ℹ${NC} $1"; }
print_success() { echo -e "${GREEN}✓${NC} $1"; }
print_warning() { echo -e "${YELLOW}⚠${NC} $1"; }
print_error()   { echo -e "${RED}✗${NC} $1"; }

ask_question() {
    local question="$1" default="$2" response
    if [ -n "$default" ]; then
        echo -ne "${GREEN}?${NC} $question [${YELLOW}$default${NC}]: " >&2
        read response; echo "${response:-$default}"
    else
        echo -ne "${GREEN}?${NC} $question: " >&2
        read response; echo "$response"
    fi
}

ask_password() {
    local question="$1" response
    echo -ne "${GREEN}?${NC} $question: " >&2
    read -s response; echo "" >&2; echo "$response"
}

ask_yes_no() {
    local question="$1" default="${2:-n}" response
    if [ "$default" = "y" ]; then
        echo -ne "${GREEN}?${NC} $question [${YELLOW}Y/n${NC}]: " >&2
        read response; response=${response:-y}
    else
        echo -ne "${GREEN}?${NC} $question [${YELLOW}y/N${NC}]: " >&2
        read response; response=${response:-n}
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

generate_master_key() {
    openssl rand -hex 32
}

check_prerequisites() {
    print_header "Checking Prerequisites"
    local missing_tools=()
    command -v openssl >/dev/null 2>&1 || missing_tools+=("openssl")
    command -v docker  >/dev/null 2>&1 || missing_tools+=("docker")
    command -v docker  >/dev/null 2>&1 && docker compose version >/dev/null 2>&1 || missing_tools+=("docker compose")
    if [ ${#missing_tools[@]} -gt 0 ]; then
        print_error "Missing required tools: ${missing_tools[*]}"
        exit 1
    fi

    if ! docker info >/dev/null 2>&1; then
        print_error "Docker daemon is not running or not reachable. Start Docker and try again."
        exit 1
    fi

    print_success "All prerequisites met"
}

main() {
    clear
    echo -e "${GREEN}"
    cat << "BANNER"
    _   __               ____
   / | / /__ _      __  / __ \____ ___  __
  /  |/ / _ \ | /| / / /_/ / __ `/ / / /
 / /|  /  __/ |/ |/ / ____/ /_/ / /_/ /
/_/ |_/\___/|__/|__/_/    \__,_/\__, /
                                  /____/
    Production Environment Setup
BANNER
    echo -e "${NC}"

    print_warning "This script will create a production .env file with secure credentials."
    print_warning "Make sure you are in a secure environment!"
    echo ""

    if ! ask_yes_no "Do you want to continue?" "y"; then
        print_info "Setup cancelled."; exit 0
    fi

    check_prerequisites

    if [ -f .env ]; then
        print_warning "A .env file already exists!"
        if ! ask_yes_no "Do you want to overwrite it? (Backup will be created)" "n"; then
            print_info "Setup cancelled."; exit 0
        fi
        mv .env ".env.backup.$(date +%Y%m%d_%H%M%S)"
        print_success "Existing .env backed up"
    fi

    # ── Basic Configuration ──────────────────────────────────────────────────
    print_header "Basic Configuration"
    DOMAIN=$(ask_question "Enter your domain (e.g., newpay.example.com)" "localhost")
    APP_NAME=$(ask_question "Application name" "NewPay")

    PROTOCOL="https"
    [ "$DOMAIN" = "localhost" ] && PROTOCOL="http"

    # ── Database ─────────────────────────────────────────────────────────────
    print_header "Database Configuration"
    DB_USER=$(ask_question "Database username" "newpay_prod")
    DB_PASSWORD=$(ask_password "Database password (leave empty to auto-generate)")
    [ -z "$DB_PASSWORD" ] && DB_PASSWORD=$(generate_random_string 32)
    DB_NAME=$(ask_question "Database name" "newpay_prod")

    # ── JWT ──────────────────────────────────────────────────────────────────
    print_header "JWT Configuration"
    print_info "Generating secure JWT key (ECDSA P-256)..."
    JWT_SECRET=$(generate_jwt_key)
    print_success "JWT key generated"
    JWT_EXPIRATION=$(ask_question "JWT expiration" "24h")
    JWT_REFRESH_EXPIRATION=$(ask_question "JWT refresh expiration" "168h")
    SESSION_TIMEOUT=$(ask_question "Session timeout" "30m")

    # ── Encryption Master Key ────────────────────────────────────────────────
    print_header "Encryption Master Key"
    print_info "Generating AES-256 master key (32 bytes / 64 hex chars)..."
    ENCRYPTION_MASTER_KEY=$(generate_master_key)
    print_success "Encryption master key generated"
    print_warning "IMPORTANT: Back up this key securely (password manager, secret manager)."
    print_warning "Losing it makes all encrypted data permanently unreadable!"

    # ── SMTP ─────────────────────────────────────────────────────────────────
    print_header "Email (SMTP) Configuration"
    SMTP_HOST=$(ask_question "SMTP host" "smtp.example.com")
    SMTP_PORT=$(ask_question "SMTP port" "587")
    SMTP_USERNAME=$(ask_question "SMTP username" "noreply@${DOMAIN}")
    SMTP_PASSWORD=$(ask_password "SMTP password")
    SMTP_FROM=$(ask_question "Email from address" "noreply@${DOMAIN}")
    EMAIL_VERIFICATION_URL="${PROTOCOL}://${DOMAIN}/verify-email"
    PASSWORD_RESET_URL="${PROTOCOL}://${DOMAIN}/reset-password"

    # ── CORS ─────────────────────────────────────────────────────────────────
    print_header "CORS Configuration"
    if [ "$DOMAIN" = "localhost" ]; then
        CORS_ORIGINS="http://localhost:3000,http://localhost:5173,http://localhost"
    else
        CORS_ORIGINS="${PROTOCOL}://${DOMAIN}"
    fi
    CORS_ORIGINS=$(ask_question "Allowed CORS origins (comma-separated)" "$CORS_ORIGINS")

    # ── OAuth ────────────────────────────────────────────────────────────────
    print_header "OAuth/SSO Configuration"
    OAUTH_ENABLED="false"
    OAUTH_1_ENABLED="false"
    if ask_yes_no "Do you want to configure OAuth/SSO providers?" "n"; then
        OAUTH_ENABLED="true"
        OAUTH_REDIRECT_URL="${PROTOCOL}://${DOMAIN}/api/v1/auth/oauth/callback"
        OAUTH_FRONTEND_CALLBACK_URL="${PROTOCOL}://${DOMAIN}/oauth/callback"
        if ask_yes_no "Configure OAuth Provider 1?" "y"; then
            OAUTH_1_ENABLED="true"
            OAUTH_1_NAME=$(ask_question "Provider 1 name" "GitLab")
            OAUTH_1_CLIENT_ID=$(ask_question "Provider 1 Client ID")
            OAUTH_1_CLIENT_SECRET=$(ask_password "Provider 1 Client Secret")
            OAUTH_1_AUTH_URL=$(ask_question "Provider 1 Auth URL" "https://gitlab.com/oauth/authorize")
            OAUTH_1_TOKEN_URL=$(ask_question "Provider 1 Token URL" "https://gitlab.com/oauth/token")
            OAUTH_1_USER_INFO_URL=$(ask_question "Provider 1 User Info URL" "https://gitlab.com/api/v4/user")
            OAUTH_1_GROUP_MAPPING=$(ask_question "Provider 1 Group Mapping (optional)" "")
        fi
    fi

    # ── Registration ─────────────────────────────────────────────────────────
    print_header "User Registration"
    ENABLE_REGISTRATION="false"
    ask_yes_no "Enable email/password registration?" "n" && ENABLE_REGISTRATION="true"
    ENABLE_OAUTH_REGISTRATION="false"
    [ "$OAUTH_ENABLED" = "true" ] && ask_yes_no "Enable OAuth registration?" "y" && ENABLE_OAUTH_REGISTRATION="true"

    # ── Scheduler ────────────────────────────────────────────────────────────
    print_header "Scheduler Configuration"
    SCHEDULER_ENABLE_DRAFT_REMINDERS="false"
    SCHEDULER_DRAFT_REMINDER_CRON="0 9 * * 1"
    ask_yes_no "Enable draft reminder emails?" "y" && SCHEDULER_ENABLE_DRAFT_REMINDERS="true" &&         SCHEDULER_DRAFT_REMINDER_CRON=$(ask_question "Draft reminder cron" "0 9 * * 1")

    SCHEDULER_ENABLE_REVIEWER_SUMMARY="false"
    SCHEDULER_REVIEWER_SUMMARY_CRON="0 8 * * *"
    ask_yes_no "Enable reviewer summary emails?" "y" && SCHEDULER_ENABLE_REVIEWER_SUMMARY="true" &&         SCHEDULER_REVIEWER_SUMMARY_CRON=$(ask_question "Reviewer summary cron" "0 8 * * *")

    SCHEDULER_ENABLE_HASH_CHAIN_VALIDATION="false"
    SCHEDULER_HASH_CHAIN_VALIDATION_CRON="0 3 * * *"
    ask_yes_no "Enable hash chain validation?" "y" && SCHEDULER_ENABLE_HASH_CHAIN_VALIDATION="true" &&         SCHEDULER_HASH_CHAIN_VALIDATION_CRON=$(ask_question "Hash chain validation cron" "0 3 * * *")

    SCHEDULER_REMINDER_INTERVAL_MINS=$(ask_question "Draft reminder interval (minutes, 10080=7 days)" "10080")

    # ── LLM ─────────────────────────────────────────────────────────────────
    print_header "LLM Configuration (Optional)"
    LLM_ENABLED="false"
    LLM_MODEL="llama3"
    if ask_yes_no "Enable LLM/AI features (requires Ollama, ~4.5 GB model download)?" "n"; then
        LLM_ENABLED="true"
        LLM_MODEL=$(ask_question "LLM model name" "llama3")
    fi

    # ── Rate Limiting ────────────────────────────────────────────────────────
    print_header "Rate Limiting"
    RATE_LIMIT_REQUESTS=$(ask_question "Rate limit requests per window" "1000")
    RATE_LIMIT_DURATION=$(ask_question "Rate limit window duration" "1m")

    # ── Write .env ───────────────────────────────────────────────────────────
    print_header "Creating .env File"

    cat > .env << ENVEOF
# ============================================================================
# NewPay Production Environment Configuration
# Generated: $(date)
# ============================================================================

DB_HOST=postgres
DB_PORT=5432
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=${DB_NAME}
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_TIMEOUT_READ=15s
SERVER_TIMEOUT_WRITE=15s
SERVER_TIMEOUT_IDLE=60s

APP_ENV=production
APP_NAME=${APP_NAME}
APP_VERSION=1.0.0
LOG_LEVEL=info

JWT_SECRET=${JWT_SECRET}
JWT_EXPIRATION=${JWT_EXPIRATION}
JWT_REFRESH_EXPIRATION=${JWT_REFRESH_EXPIRATION}

SESSION_TIMEOUT=${SESSION_TIMEOUT}

# IMPORTANT: Back up this key securely — losing it makes encrypted data unreadable!
ENCRYPTION_MASTER_KEY=${ENCRYPTION_MASTER_KEY}

SMTP_HOST=${SMTP_HOST}
SMTP_PORT=${SMTP_PORT}
SMTP_USERNAME=${SMTP_USERNAME}
SMTP_PASSWORD=${SMTP_PASSWORD}
SMTP_FROM=${SMTP_FROM}
EMAIL_VERIFICATION_URL=${EMAIL_VERIFICATION_URL}
PASSWORD_RESET_URL=${PASSWORD_RESET_URL}

CORS_ALLOWED_ORIGINS=${CORS_ORIGINS}
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Accept,Authorization,Content-Type,X-CSRF-Token
CORS_EXPOSED_HEADERS=Link
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=300

RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=${RATE_LIMIT_REQUESTS}
RATE_LIMIT_DURATION=${RATE_LIMIT_DURATION}

ENABLE_REGISTRATION=${ENABLE_REGISTRATION}
ENABLE_OAUTH_REGISTRATION=${ENABLE_OAUTH_REGISTRATION}

SCHEDULER_ENABLE_DRAFT_REMINDERS=${SCHEDULER_ENABLE_DRAFT_REMINDERS}
SCHEDULER_ENABLE_REVIEWER_SUMMARY=${SCHEDULER_ENABLE_REVIEWER_SUMMARY}
SCHEDULER_ENABLE_HASH_CHAIN_VALIDATION=${SCHEDULER_ENABLE_HASH_CHAIN_VALIDATION}
SCHEDULER_DRAFT_REMINDER_CRON=${SCHEDULER_DRAFT_REMINDER_CRON}
SCHEDULER_REVIEWER_SUMMARY_CRON=${SCHEDULER_REVIEWER_SUMMARY_CRON}
SCHEDULER_HASH_CHAIN_VALIDATION_CRON=${SCHEDULER_HASH_CHAIN_VALIDATION_CRON}
SCHEDULER_REMINDER_INTERVAL_MINS=${SCHEDULER_REMINDER_INTERVAL_MINS}

LLM_ENABLED=${LLM_ENABLED}
LLM_BASE_URL=http://ollama:11434
LLM_MODEL=${LLM_MODEL}

VITE_API_BASE_URL=/api/v1
ENVEOF

    if [ "$OAUTH_ENABLED" = "true" ]; then
        cat >> .env << ENVEOF

OAUTH_REDIRECT_URL=${OAUTH_REDIRECT_URL}
OAUTH_FRONTEND_CALLBACK_URL=${OAUTH_FRONTEND_CALLBACK_URL}
OAUTH_1_ENABLED=${OAUTH_1_ENABLED}
ENVEOF
        if [ "$OAUTH_1_ENABLED" = "true" ]; then
            cat >> .env << ENVEOF
OAUTH_1_NAME=${OAUTH_1_NAME}
OAUTH_1_CLIENT_ID=${OAUTH_1_CLIENT_ID}
OAUTH_1_CLIENT_SECRET=${OAUTH_1_CLIENT_SECRET}
OAUTH_1_AUTH_URL=${OAUTH_1_AUTH_URL}
OAUTH_1_TOKEN_URL=${OAUTH_1_TOKEN_URL}
OAUTH_1_USER_INFO_URL=${OAUTH_1_USER_INFO_URL}
OAUTH_1_GROUP_MAPPING=${OAUTH_1_GROUP_MAPPING}
OAUTH_1_GROUPS_CLAIM=groups
ENVEOF
        fi
    fi

    chmod 600 .env
    print_success ".env file created (permissions set to 600)"

    # ── Save master key separately ───────────────────────────────────────────
    cat > encryption-key.txt << KEYEOF
# ============================================================================
# NewPay Encryption Master Key — KEEP THIS FILE SECURE!
# ============================================================================
# Generated: $(date)
#
# Store this key in a password manager or secret manager.
# Delete this file once the key is safely backed up.
# ============================================================================

ENCRYPTION_MASTER_KEY=${ENCRYPTION_MASTER_KEY}
KEYEOF
    chmod 600 encryption-key.txt
    print_success "Encryption key saved to: encryption-key.txt"
    print_warning "IMPORTANT: Back up encryption-key.txt and then delete it!"

    # ── Build & start ────────────────────────────────────────────────────────
    print_header "Pulling & Building Docker Images"
    print_info "Pulling base images (this can take several minutes on first run)..."
    if [ "$LLM_ENABLED" = "true" ]; then
        docker compose pull postgres ollama || true
    else
        docker compose pull postgres || true
    fi

    print_info "Building application images (frontend, api)..."
    if ! docker compose build --progress=plain frontend api; then
        print_error "Docker build failed. Run 'docker compose build --progress=plain frontend api' for detailed logs."
        exit 1
    fi
    print_success "Docker images ready"

    if [ "$LLM_ENABLED" = "true" ]; then
        print_header "LLM Model Download"
        print_info "Starting Ollama..."
        docker compose up -d ollama
        sleep 10
        if docker compose exec -T ollama ollama pull "${LLM_MODEL}"; then
            print_success "LLM model ${LLM_MODEL} downloaded"
        else
            print_warning "Could not download model. Pull manually: docker compose exec ollama ollama pull ${LLM_MODEL}"
        fi
        docker compose down
    fi

    # ── Summary ──────────────────────────────────────────────────────────────
    print_header "Setup Complete!"
    echo ""
    print_success "Production environment configured successfully!"
    echo ""
    print_info "Summary:"
    echo "  • Domain:      ${DOMAIN}"
    echo "  • Database:    ${DB_NAME} (user: ${DB_USER})"
    echo "  • JWT:         ECDSA P-256 key generated"
    echo "  • Encryption:  AES-256 master key generated"
    echo "  • SMTP:        ${SMTP_HOST}:${SMTP_PORT}"
    echo "  • OAuth:       ${OAUTH_ENABLED}"
    echo "  • LLM:         ${LLM_ENABLED}"
    echo ""
    print_info "Next steps:"
    echo "  1. Back up the encryption master key from encryption-key.txt"
    echo "  2. Delete encryption-key.txt after storing the key safely"
    echo "  3. Start the stack: docker compose up -d"
    echo "  4. Access the application at: ${PROTOCOL}://${DOMAIN}"
    echo ""
    print_warning "Security reminders:"
    echo "  • Never commit .env or encryption-key.txt to version control"
    echo "  • The ENCRYPTION_MASTER_KEY cannot be recovered if lost"
    echo "  • Consider HTTPS via a reverse proxy for ${DOMAIN}"
    echo ""
}

main
