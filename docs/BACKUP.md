# Backup & Restore

## Komponenten

- Datenbank: PostgreSQL
- Konfiguration: `.env`
- Secret: `ENCRYPTION_MASTER_KEY` (separat und sicher aufbewahren)

## Grundsaetze

- Backups enthalten sensible Daten und muessen als Secret behandelt werden.
- Der `ENCRYPTION_MASTER_KEY` gehoert nicht in Git und nicht unverschluesselt ins Backup-Verzeichnis.
- Ohne gueltigen Master-Key koennen verschluesselte Inhalte nicht wiederhergestellt werden.

## Backup-Skript (Beispiel)

```bash
#!/bin/bash
set -euo pipefail

BACKUP_DIR="./backups/$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Datenbank
DockerContainer="newpay-postgres-prod"
docker exec "$DockerContainer" pg_dump -U newpay_prod newpay_prod | gzip > "$BACKUP_DIR/database.sql.gz"

# Laufzeitkonfiguration (enthaelt Secrets)
cp .env "$BACKUP_DIR/.env.backup"

# Optionale Integritaetsdatei
shasum -a 256 "$BACKUP_DIR/database.sql.gz" > "$BACKUP_DIR/database.sql.gz.sha256"

# Aufraeumen alter Backups (7 Tage)
find ./backups -mindepth 1 -maxdepth 1 -type d -mtime +7 -exec rm -rf {} \;
```

## Backup ausfuehren

```bash
chmod +x backup.sh
./backup.sh
```

## Cronjob (taeglich 02:00)

```bash
0 2 * * * cd /pfad/zu/docker/production && ./backup.sh >> ./logs/backup.log 2>&1
```

## Manuelle Backups

### Datenbank

```bash
# Vollbackup
docker exec newpay-postgres-prod pg_dump -U newpay_prod newpay_prod | gzip > backup-$(date +%Y%m%d-%H%M%S).sql.gz

# Nur Schema
docker exec newpay-postgres-prod pg_dump -U newpay_prod newpay_prod --schema-only | gzip > schema-$(date +%Y%m%d).sql.gz

# Nur Daten
docker exec newpay-postgres-prod pg_dump -U newpay_prod newpay_prod --data-only | gzip > data-$(date +%Y%m%d).sql.gz
```

### Konfiguration

```bash
cp .env .env.backup-$(date +%Y%m%d)
```

## Restore

### Datenbank-Restore

```bash
gunzip < backups/20231201-120000/database.sql.gz | docker exec -i newpay-postgres-prod psql -U newpay_prod newpay_prod
```

### Komplett-Restore (DB + App)

```bash
cd docker/production
docker compose down

gunzip < backups/20231201-120000/database.sql.gz | docker exec -i newpay-postgres-prod psql -U newpay_prod newpay_prod

docker compose up -d
```

### Master-Key beim Restore

- Stelle sicher, dass `ENCRYPTION_MASTER_KEY` im Zielsystem korrekt gesetzt ist.
- Nutze exakt den Key der beim Verschluesseln der Daten aktiv war.

## Off-Site-Backups

```bash
# Remote-Server
rsync -avz --progress ./backups/ backup-server:/pfad/zu/backups/newpay/

# S3
aws s3 sync ./backups/ s3://newpay-backups/$(date +%Y%m%d)/
```

## Backup-Tests (dringend empfohlen)

```bash
cd docker/development
docker compose down -v
docker compose up -d

gunzip < ../production/backups/20231201-120000/database.sql.gz | \
  docker exec -i newpay-postgres-dev psql -U newpay newpay

# Beispiel-Healthcheck
curl http://localhost:8080/health

docker compose down -v
```

## Aufbewahrungsstrategie (Beispiel)

- Taeglich: 7 Tage
- Woechentlich: 4 Wochen
- Monatlich: 12 Monate
- Jaehrlich: unbegrenzt (Compliance)

## Monitoring

```bash
ls -lt ./backups/ | head -n 5
du -sh ./backups/*/
tail -f ./logs/backup.log
```

## Sicherheitshinweise

1. **Backups verschluesseln**

```bash
gpg --symmetric --cipher-algo AES256 backup-20231201.sql.gz
```

2. **Zugriff beschraenken**
- Backup-Storage nur fuer notwendige Accounts freigeben.
- Rechte und Schluessel regelmaessig rotieren.

3. **Master-Key-Schutz**
- `ENCRYPTION_MASTER_KEY` in Passwort-Manager/Secret-Store sichern.
- Nie im Repository, nie im Klartext via Chat/Ticket teilen.

4. **Restore-Probe**
- Restore-Prozess regelmaessig in isolierter Testumgebung ueben.
- Erfolgskriterien dokumentieren (Start, Login, Entschluesselung, Kernflows).
