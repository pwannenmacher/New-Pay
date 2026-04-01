#!/bin/bash
# Start all NewPay production services.
# Encryption is handled by the application using ENCRYPTION_MASTER_KEY from .env.
set -e

if [ ! -f .env ]; then
    echo "ERROR: .env not found. Run ./setup.sh first." >&2
    exit 1
fi

docker compose up -d
echo "NewPay started. Check status with: docker compose ps"
