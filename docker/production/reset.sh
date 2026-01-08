#!/bin/bash

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
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

print_warning "This script will reset the entire production environment!"
print_warning "All data, including Vault data and configurations, will be permanently deleted."
echo ""

if ! ask_yes_no "Do you want to continue?" "n"; then
    print_info "Reset cancelled."
    exit 0
fi

docker compose down -v
rm -f vault-credentials.txt
rm -f auto-unseal.sh
rm -f .env
print_success "Production environment reset. All data and configurations have been removed."
