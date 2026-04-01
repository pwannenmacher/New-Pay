# Production Deployment

## Quick Start with Interactive Setup

**Recommended:** Use the interactive setup script that handles everything automatically:

```bash
cd docker/production
./setup.sh
```

The script will guide you through:

- ✅ Generating secure secrets (JWT key, AES-256 encryption master key)
- ✅ Configuring database credentials
- ✅ Setting up SMTP for email
- ✅ Configuring OAuth/SSO providers (optional)
- ✅ Configuring LLM/AI features (optional)
- ✅ Creating the production `.env` file

After setup completes, start the stack:

```bash
docker compose up -d
```

---

## Manual Setup (Alternative)

### Prerequisites

- Docker Compose V2
- 4 GB RAM minimum
- 2 GB disk space
- Domain with DNS record

### 1. Environment Variables

```bash
cd docker/production
cp .env.example .env
nano .env
```

Required changes:

- `DB_PASSWORD` — secure database password
- `JWT_SECRET` — generate with `go run scripts/generate-jwt-keys.go`
- `ENCRYPTION_MASTER_KEY` — **generate with `openssl rand -hex 32`** and store securely
- `SMTP_*` — mail server credentials

### 2. Generate Keys

```bash
# JWT key (ECDSA P-256)
openssl ecparam -genkey -name prime256v1 -noout | openssl ec -outform PEM

# Encryption master key (AES-256, 32 bytes)
openssl rand -hex 32
```

> ⚠️ **Back up the `ENCRYPTION_MASTER_KEY` in a password manager or secrets vault.**
> It cannot be recovered if lost, and losing it makes all encrypted review
> justifications permanently unreadable.

### 3. Start the Stack

```bash
docker compose up -d
```

Migrations run automatically on API startup.

---

## Operations

### Health Checks

```bash
curl http://localhost:8080/health
docker compose ps
docker compose logs -f api
```

### Updates

```bash
git pull
docker compose build --no-cache api frontend
docker compose up -d
docker image prune -f
```

### Backups

```bash
# Database
docker exec newpay-postgres-prod pg_dump -U newpay_prod newpay_prod | gzip > backup-$(date +%Y%m%d).sql.gz

# Encryption master key — already backed up outside Docker in your password manager
```

---

## Security Checklist

- [ ] `ENCRYPTION_MASTER_KEY` backed up to password manager
- [ ] `DB_PASSWORD` is strong and unique
- [ ] `JWT_SECRET` uses ECDSA key (not a plain string)
- [ ] `.env` has permissions `600`
- [ ] HTTPS via reverse proxy
- [ ] Firewall: only ports 80/443 open externally
- [ ] Log monitoring configured

## Troubleshooting

```bash
# API does not start
docker compose logs api
# → Check DB connection and ENCRYPTION_MASTER_KEY format (must be 64 hex chars)

# Frontend API errors
curl http://localhost:8080/health
docker compose logs frontend
```
