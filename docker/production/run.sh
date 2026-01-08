#!/bin/bash

# Start Vault container
docker compose up -d vault

sleep 10

# Run auto-unseal script
./auto-unseal.sh

# Start other services
docker compose up -d
