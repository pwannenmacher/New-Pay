#!/bin/bash
set -e

# Vault Restore Script
# Stellt Vault-Daten aus einem Backup wieder her

if [ -z "$1" ]; then
  echo "❌ Fehler: Backup-Verzeichnis muss angegeben werden"
  echo ""
  echo "Verwendung: ./restore-vault.sh <backup-verzeichnis>"
  echo ""
  echo "Verfügbare Backups:"
  ls -1d ./backups/*/ 2>/dev/null || echo "  Keine Backups gefunden"
  exit 1
fi

BACKUP_DIR="$1"
VAULT_VOLUME="vault_vault-data"

if [ ! -d "$BACKUP_DIR" ]; then
  echo "❌ Fehler: Backup-Verzeichnis nicht gefunden: $BACKUP_DIR"
  exit 1
fi

if [ ! -f "$BACKUP_DIR/vault-data.tar.gz" ]; then
  echo "❌ Fehler: vault-data.tar.gz nicht im Backup-Verzeichnis gefunden"
  exit 1
fi

echo "🔐 Vault Restore"
echo "=================="
echo "📁 Backup: $BACKUP_DIR"
echo ""

# Backup-Info anzeigen, falls vorhanden
if [ -f "$BACKUP_DIR/backup-info.txt" ]; then
  cat "$BACKUP_DIR/backup-info.txt"
  echo ""
fi

# Sicherheitsabfrage
read -p "⚠️  WARNUNG: Alle aktuellen Vault-Daten werden überschrieben! Fortfahren? (yes/no): " -r
echo
if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
  echo "❌ Abgebrochen"
  exit 0
fi

echo "🛑 Vault wird gestoppt..."
docker compose down

echo "🗑️  Altes Vault-Volume wird entfernt..."
docker volume rm ${VAULT_VOLUME} 2>/dev/null || true

echo "📦 Neues Vault-Volume wird erstellt..."
docker volume create ${VAULT_VOLUME}

echo "📥 Backup wird wiederhergestellt..."
docker run --rm \
  -v ${VAULT_VOLUME}:/target \
  -v "$(cd "$(dirname "$BACKUP_DIR")" && pwd)/$(basename "$BACKUP_DIR"):/backup:ro" \
  alpine sh -c "cd /target && tar xzf /backup/vault-data.tar.gz"

echo "🚀 Vault wird gestartet..."
docker compose up -d

echo "⏳ Warte auf Vault-Start..."
sleep 5

# Vault Status prüfen
echo ""
echo "📊 Vault Status:"
docker compose ps

echo ""
echo "✅ Restore abgeschlossen!"
echo ""
echo "🔓 WICHTIG: Vault muss noch entsiegelt werden:"
echo "   ./auto-unseal.sh"
echo ""
echo "   Oder manuell:"
echo "   docker exec -it vault_vault_1 vault operator unseal"
