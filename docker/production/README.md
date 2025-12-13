# Production Deployment

## Voraussetzungen

- Docker Engine 20.10+
- Docker Compose V2
- 4GB RAM
- 20GB Speicherplatz
- Domain mit DNS-Eintrag

## Setup

### 1. Umgebungsvariablen

```bash
cd docker/production
cp .env.example .env
nano .env
```

Ändern:
- `DB_PASSWORD`
- `JWT_SECRET` (siehe unten)
- `SMTP_*`
- `VAULT_TOKEN` (nach Vault-Init)
- `VITE_API_URL`

### 2. JWT Key

```bash
openssl ecparam -genkey -name prime256v1 -noout | openssl ec -outform PEM > jwt-key.pem
cat jwt-key.pem | base64  # in .env eintragen
```

### 3. Stack starten

Reverse-Proxy muss SSL-Termination übernehmen und zu Port 80 forwarden.

```bash
docker compose up -d
docker compose logs -f vault

# Init
docker exec -it newpay-vault-prod vault operator init
# Keys und Root Token sichern!

# Unseal (3 Keys)
docker exec -it newpay-vault-prod vault operator unseal
# Wiederholen mit Key 2 und Key 3

# Token in .env
nano .env  # VAULT_TOKEN=s.xxx...

# Transit Engine
export VAULT_TOKEN=<ROOT_TOKEN>
docker exec -e VAULT_TOKEN=$VAULT_TOKEN newpay-vault-prod vault secrets enable transit

# API neu starten
docker compose restart api
```

Nach Neustarts: Vault mit 3 Keys unsealen.

## Betrieb

### Vault Unseal

```bash
docker exec -it newpay-vault-prod vault operator unseal
# Wiederholen mit Key 2 und Key 3

docker exec newpay-vault-prod vault status
```

### Health Checks

```bash
curl http://localhost:8080/health
docker compose logs -f
docker compose logs -f api
```

## Updates

```bash
git pull
docker compose build --no-cache api frontend
docker compose up -d
docker image prune -f
```

Migrationen laufen automatisch beim API-Start.

## Backups

Details: [docs/BACKUP.md](../../docs/BACKUP.md)

```bash
# DB
docker exec newpay-postgres-prod pg_dump -U newpay_prod newpay_prod | gzip > backup-$(date +%Y%m%d).sql.gz

# Vault
docker run --rm -v newpay_vault-data:/source:ro -v $(pwd):/backup alpine tar czf /backup/vault-$(date +%Y%m%d).tar.gz -C /source .
```

## Sicherheit

```bash
# Firewall
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

Checkliste:
- Sichere Passwörter (DB, SMTP)
- JWT Key sicher aufbewahren
- Vault Keys an verschiedenen Orten
- Root Token rotieren
- Tägliche Backups
- HTTPS via Reverse-Proxy
- Log-Monitoring
- Updates

## Wartung

```bash
# /etc/docker/daemon.json
{"log-driver":"json-file","log-opts":{"max-size":"10m","max-file":"3"}}

sudo systemctl restart docker
docker volume prune -f
docker image prune -a -f
```

## Troubleshooting

```bash
# API startet nicht
docker compose logs api
# → DB erreichbar? Vault unsealed?

# Vault sealed
docker exec -it newpay-vault-prod vault operator unseal
# 3x wiederholen

# Frontend API-Fehler
curl http://localhost:8080/health
docker compose logs frontend
```
