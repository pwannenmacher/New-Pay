#!/bin/bash
set -e

# Vault Backup Script
# Erstellt ein Backup der Vault-Daten und Konfiguration

BACKUP_DIR="./backups/$(date +%Y%m%d-%H%M%S)"
VAULT_VOLUME="vault_vault-data"

echo "🔐 Vault Backup wird erstellt..."
echo "📁 Backup-Verzeichnis: $BACKUP_DIR"

# Backup-Verzeichnis erstellen
mkdir -p "$BACKUP_DIR"

# 1. Vault Daten sichern
echo "📦 Vault-Daten werden gesichert..."
docker run --rm \
  -v ${VAULT_VOLUME}:/source:ro \
  -v "$(pwd)/backups:/backup" \
  alpine tar czf "/backup/$(basename $BACKUP_DIR)/vault-data.tar.gz" -C /source .

# 2. Konfiguration sichern
echo "⚙️  Konfiguration wird gesichert..."
cp vault-config.hcl "$BACKUP_DIR/vault-config.hcl"
cp auto-unseal.sh "$BACKUP_DIR/auto-unseal.sh"

# 3. Metadaten hinzufügen
echo "📝 Metadaten werden erstellt..."
cat > "$BACKUP_DIR/backup-info.txt" << EOF
Vault Backup
============
Datum: $(date)
Vault Version: $(docker exec vault_vault_1 vault version 2>/dev/null || echo "Container läuft nicht")
Backup-Größe: $(du -sh "$BACKUP_DIR/vault-data.tar.gz" | cut -f1)

Wiederherstellung:
------------------
./restore-vault.sh $BACKUP_DIR
EOF

echo "✅ Backup erfolgreich erstellt: $BACKUP_DIR"
echo ""
echo "📊 Backup-Größe: $(du -sh "$BACKUP_DIR" | cut -f1)"
echo "📦 Dateien:"
ls -lh "$BACKUP_DIR"

# Optional: Alte Backups aufräumen (älter als 7 Tage)
echo ""
echo "🧹 Alte Backups werden aufgeräumt (älter als 7 Tage)..."
find ./backups -mindepth 1 -maxdepth 1 -type d -mtime +7 -exec rm -rf {} \; 2>/dev/null || true
echo "✅ Fertig!"
