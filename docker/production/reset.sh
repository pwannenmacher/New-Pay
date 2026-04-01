#!/bin/bash

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_success() { echo -e "${GREEN}✓${NC} $1"; }
print_warning() { echo -e "${YELLOW}⚠${NC} $1"; }
print_info()    { echo -e "ℹ $1"; }

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

print_warning "This script will reset the entire production environment!"
print_warning "All data will be permanently deleted."
echo ""

if ! ask_yes_no "Do you want to continue?" "n"; then
    print_info "Reset cancelled."; exit 0
fi

docker compose down -v
rm -f encryption-key.txt
rm -f .env
print_success "Production environment reset. All data and configurations removed."
