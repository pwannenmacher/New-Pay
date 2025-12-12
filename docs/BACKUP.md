# Backup & Restore

## Komponenten

- Datenbank: PostgreSQL
- Vault: Verschlüsselte Daten
- Config: .env (Secrets separat sichern)

## Backup Script

```bash
#!/bin/bash
set -e

BACKUP_DIR="./backups/$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Database
docker exec newpay-postgres-prod pg_dump -U newpay_prod newpay_prod | gzip > "$BACKUP_DIR/database.sql.gz"

# Vault
docker run --rm \
  -v newpay_vault-data:/source:ro \
  -v "$(pwd)/backups:/backup" \
  alpine tar czf "/backup/$(basename $BACKUP_DIR)/vault-data.tar.gz" -C /source .

# Config
cp .env "$BACKUP_DIR/.env.backup"

# Cleanup (7 Tage)
find ./backups -mindepth 1 -maxdepth 1 -type d -mtime +7 -exec rm -rf {} \;
```

### Ausführung

```bash
chmod +x backup.sh
./backup.sh

# Cron (täglich 2:00)
0 2 * * * cd /pfad/zu/docker/production && ./backup.sh >> ./logs/backup.log 2>&1
```

## Manuelle Backup-Befehle

### Datenbank Backup

```bash
# Backup erstellen
docker exec newpay-postgres-prod pg_dump -U newpay_prod newpay_prod | gzip > backup-$(date +%Y%m%d-%H%M%S).sql.gz

# Nur Schema (ohne Daten)
docker exec newpay-postgres-prod pg_dump -U newpay_prod newpay_prod --schema-only | gzip > schema-$(date +%Y%m%d).sql.gz

# Nur Daten (ohne Schema)
docker exec newpay-postgres-prod pg_dump -U newpay_prod newpay_prod --data-only | gzip > data-$(date +%Y%m%d).sql.gz
```

### Vault Daten Backup

```bash
# Volume als tar.gz sichern
docker run --rm \
  -v newpay_vault-data:/source:ro \
  -v $(pwd):/backup \
  alpine tar czf /backup/vault-backup-$(date +%Y%m%d-%H%M%S).tar.gz -C /source .
```

### Umgebungsvariablen Backup

```bash
# .env sichern (ACHTUNG: Enthält Secrets!)
cp .env .env.backup-$(date +%Y%m%d)

# Secrets sollten separat in einem Passwort-Manager gesichert werden
```

## Restore

### Datenbank

```bash
gunzip < backups/20231201-120000/database.sql.gz | docker exec -i newpay-postgres-prod psql -U newpay_prod newpay_prod
```

### Vault

Stack stoppen:

```bash
cd docker/production
docker compose down
docker volume rm newpay_vault-data
docker volume create newpay_vault-data
docker run --rm -v newpay_vault-data:/target -v $(pwd)/backups:/backup alpine sh -c "cd /target && tar xzf /backup/20231201-120000/vault-data.tar.gz"
docker compose up -d
# Vault unsealen
```

### Komplett-Restore

```bash
docker compose down -v
docker volume create newpay_postgres-data
docker run --rm -v newpay_postgres-data:/target -v $(pwd)/backups:/backup alpine sh -c "cd /target && tar xzf /backup/20231201/postgres.tar.gz"
# Vault restore (siehe oben)
docker compose up -d
```
- **Jahres-Backups**: Unbegrenzt (Compliance)

### Off-Site Backup

Backups sollten auch außerhalb des Produktions-Servers gespeichert werden:

```bash
# Backup auf Remote-Server kopieren (via rsync)
rsync -avz --progress ./backups/ backup-server:/pfad/zu/backups/newpay/

# Backup in Cloud-Storage (z.B. AWS S3)
aws s3 sync ./backups/ s3://newpay-backups/$(date +%Y%m%d)/

# Backup auf NAS
scp -r ./backups/20231201-120000 nas:/volume1/backups/newpay/
```

## Backup testen

**WICHTIG:** Backups regelmäßig testen!

```bash
# Test-Umgebung erstellen
cd docker/development
docker compose down -v
docker compose up -d

# Backup in Test-Umgebung wiederherstellen
gunzip < ../../production/backups/20231201-120000/database.sql.gz | \
  docker exec -i newpay-postgres-dev psql -U newpay newpay

# Funktionalität testen
curl http://localhost:3000/api/health

# Test-Umgebung wieder entfernen
docker compose down -v
```

## Strategie

Frequenz:
- Täglich: Automatisch (Cron)
- Vault Root Token: Nach Änderung
- Unseal Keys: Offline, physisch getrennt
- Config: Git

Aufbewahrung:
- 7 Tage: Täglich
- 4 Wochen: Wöchentlich
- 12 Monate: Monatlich
- Unbegrenzt: Jährlich (Compliance)

Off-Site:

```Test

```bash
cd docker/development
docker compose down -v
docker compose up -d
gunzip < ../../production/backups/20231201/database.sql.gz | docker exec -i newpay-postgres-dev psql -U newpay newpay
curl http://localhost:3000/api/health
docker compose down -v
```

## Monitoring

```bash
ls -lt ./backups/ | head -n 2
du -sh ./backups/*/
tail -f ./logs/backup.log

# Alert bei Fehler
if ./backup.sh; then
    echo "OK" | mail -s "Backup OK" admin@example.com
else
    echo "Failed" | mail -s "
export VAULT_TOKEN="s.xyz..."
```

## Auto-Unseal (Optional)

**⚠️ SICHERHEITSWARNUNG**: Auto-Unseal reduziert die Sicherheit erheblich!

Ein Auto-Unseal Script ist verfügbar unter [docker/production/auto-unseal.sh](../docker/production/auto-unseal.sh).

### Verwendung

**Option 1: Keys direkt im Script** (nicht empfohlen für Production):

```bash
# Script bearbeiten
cd docker/production
nano auto-unseal.sh

# Kommentare entfernen und Keys eintragen:
UNSEAL_KEY_1="dein-key-1-hier"
UNSEAL_KEY_2="dein-key-2-hier"
UNSEAL_KEY_3="dein-key-3-hier"

# Script absichern
chmod 600 auto-unseal.sh

# Ausführen
./auto-unseal.sh
```

```bash
# pg_dump: connection failed
docker compose ps
docker compose logs postgres
docker exec -it newpay-postgres-prod psql -U newpay_prod newpay_prod

# No space left
df -h
rm -rf ./backups/old-*
docker system prune -a --volumes

# Vault sealed
docker exec newpay-vault-prod vault status
docker exec -it newpay-vault-prod vault operator unseal
# 3x wiederholen
sudo systemctl daemon-reload
sudo systemctl enable vault-unseal.service
```

**Empfehlung**: Für Production manuelles Unseal verwenden. Auto-Unseal nur wenn Server häufig neu starten.

## Sicherheitshinweise

1. **Secrets im Backup**: Backups enthalten sensitive Daten (Passwörter, Tokens, verschlüsselte Daten)
   - Backups verschlüsselt speichern (z.B. mit GPG)
   - Zugriff auf Backup-Storage beschränken
   - Backup-Storage physisch sichern

2. **Vault Unseal Keys**:
   - Niemals alle 5 Unseal Keys zusammen speichern
   - Auf mindestens 3 verschiedene, sichere Orte verteilen
   - Physische Medien bevorzugen (USB-Sticks in Safe)

3. **Root Token**:
   - Nur für Notfälle verwenden
   - Nach Verwendung widerrufen und neu generieren
   - Niemals im Git-Repository speichern

4. **Backup-Verschlüsselung**:

   ```bash
   # Backup verschlüsseln
   gpg --symmetric --cipher-algo AES256 backup.sql.gz
   
   # Backup en

WARNUNG: Reduziert Sicherheit erheblich!

Script: [docker/production/auto-unseal.sh](../docker/production/auto-unseal.sh)

```bash
# Env-Variablen (besser)
export VAULT_UNSEAL_KEY_1="..."
export VAULT_UNSEAL_KEY_2="..."
export VAULT_UNSEAL_KEY_3="..."
cd docker/production
./auto-unseal.sh

# Im Script (unsicher)
nano auto-unseal.sh  # Keys eintragen
chmod 600 auto-unseal.sh
./auto-unseal.sh
```

Systemd:

```bash
sudo tee /etc/systemd/system/vault-unseal.service <<EOF
[Unit]
Description=Vault Auto Unseal
After=docker.service

[Service]
Type=oneshot
WorkingDirectory=/pfad/zu/docker/production
Environment=VAULT_UNSEAL_KEY_1=...
Environment=VAULT_UNSEAL_KEY_2=...
Environment=VAULT_UNSEAL_KEY_3=...
ExecStart=/pfad/zu/docker/production/auto-unseal.sh

[Install]
WantedBy=multi-user.target
EOF

Backups enthalten Secrets:
- GPG-Verschlüsselung: `gpg --symmetric --cipher-algo AES256 backup.sql.gz`
- Zugriff beschränken
- Physisch sichern

Vault Keys:
- Niemals alle 5 zusammen
- 3+ Orte, physisch getrennt
- USB-Sticks bevorzugen

Root Token:
- Nur Notfälle
- Nach Verwendung rotieren
- Niemals in Git
