#!/bin/bash
# WARNUNG: Reduziert Vault-Sicherheit erheblich!
# Nur verwenden wenn Risiken verstanden (chmod 600)

set -e

VAULT_ADDR="${VAULT_ADDR:-http://localhost:8200}"
CONTAINER_NAME="${VAULT_CONTAINER:-newpay-vault-prod}"

# Keys im Script (unsicher):
# UNSEAL_KEY_1="..."
# UNSEAL_KEY_2="..."
# UNSEAL_KEY_3="..."

# Keys als Env-Variablen (besser):
# export VAULT_UNSEAL_KEY_1="..."
# export VAULT_UNSEAL_KEY_2="..."
# export VAULT_UNSEAL_KEY_3="..."

if ! docker ps | grep -q "$CONTAINER_NAME"; then
    echo "Container läuft nicht: $CONTAINER_NAME"
    exit 1
fi

STATUS=$(docker exec "$CONTAINER_NAME" vault status -format=json 2>/dev/null || echo '{}')
SEALED=$(echo "$STATUS" | jq -r '.sealed // true')
INITIALIZED=$(echo "$STATUS" | jq -r '.initialized // false')

if [ "$INITIALIZED" = "false" ]; then
    echo "Vault nicht initialisiert"
    echo "docker exec -it $CONTAINER_NAME vault operator init"
    exit 1
fi

[ "$SEALED" = "false" ] && echo "Bereits entsiegelt" && exit 0

KEY1="${VAULT_UNSEAL_KEY_1:-$UNSEAL_KEY_1}"
KEY2="${VAULT_UNSEAL_KEY_2:-$UNSEAL_KEY_2}"
KEY3="${VAULT_UNSEAL_KEY_3:-$UNSEAL_KEY_3}"

if [ -z "$KEY1" ] || [ -z "$KEY2" ] || [ -z "$KEY3" ]; then
    echo "Keys fehlen. Entweder:"
    echo "1. Im Script: UNSEAL_KEY_1/2/3"
    echo "2. Als Env: export VAULT_UNSEAL_KEY_1/2/3"
    exit 1
fi

echo "Entsiegele..."
docker exec "$CONTAINER_NAME" vault operator unseal "$KEY1" > /dev/null 2>&1
docker exec "$CONTAINER_NAME" vault operator unseal "$KEY2" > /dev/null 2>&1
docker exec "$CONTAINER_NAME" vault operator unseal "$KEY3" > /dev/null 2>&1

sleep 2
STATUS=$(docker exec "$CONTAINER_NAME" vault status -format=json 2>/dev/null || echo '{}')
SEALED=$(echo "$STATUS" | jq -r '.sealed // true')

[ "$SEALED" = "false" ] && echo "Erfolgreich" || (echo "Fehlgeschlagen" && exit 1)

