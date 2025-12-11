# Verschlüsselungssystem - Quick Start

## Was wurde implementiert?

Ein **Key Management System** basierend auf HashiCorp Vault für sichere Verschlüsselung sensibler Daten.

## Komponenten

1. **HashiCorp Vault** (Docker Container)
   - Port: 8200
   - Dev Token: `dev-root-token`
   - Transit Engine für Verschlüsselung

2. **Backend Services**
   - `internal/vault/`: Vault Client
   - `internal/keymanager/`: 3-stufige Key-Hierarchie
   - `internal/securestore/`: Verschlüsselung + Signaturen

3. **Datenbank**
   - Migration `011_encryption_tables`: user_keys, process_keys, encrypted_records

## Verwendung

### 1. Services starten

```bash
cd docker
docker-compose up -d vault postgres
```

Vault UI: http://localhost:8200 (Token: `dev-root-token`)

### 2. Migrationen ausführen

```bash
cd backend
# Migrationen werden automatisch beim Start ausgeführt
```

### 3. Im Code verwenden

```go
import (
    "github.com/pwannenmacher/new-pay-gh/backend/internal/vault"
    "github.com/pwannenmacher/new-pay-gh/backend/internal/keymanager"
    "github.com/pwannenmacher/new-pay-gh/backend/internal/securestore"
)

// Setup (in main.go oder Handler)
vaultClient, _ := vault.NewClient(&vault.Config{
    Address:      cfg.Vault.Address,
    Token:        cfg.Vault.Token,
    TransitMount: cfg.Vault.TransitMount,
})

keyManager, _ := keymanager.NewKeyManager(db, vaultClient)
store := securestore.NewSecureStore(db, keyManager)

// User Key erstellen (einmalig pro User)
publicKey, _ := keyManager.CreateUserKey(userID)

// Process Key erstellen (z.B. pro Self-Assessment)
_ = keyManager.CreateProcessKey(assessmentID, nil)

// Daten verschlüsseln
data := &securestore.PlainData{
    Fields: map[string]interface{}{
        "justification": "Meine Begründung...",
    },
}
record, _ := store.CreateRecord(assessmentID, userID, "JUSTIFICATION", data, "")

// Daten entschlüsseln (automatisch)
plainData, _ := store.DecryptRecord(record.ID)
```

## Environment Variables

Zu `.env` hinzufügen:

```bash
VAULT_ENABLED=true
VAULT_ADDR=http://vault:8200
VAULT_TOKEN=dev-root-token
VAULT_TRANSIT_MOUNT=transit
```

## Sicherheitsmerkmale

- ✅ **3-Stufen-Keys**: System + User + Process
- ✅ **AES-256-GCM**: Authenticated Encryption
- ✅ **Ed25519**: Digitale Signaturen
- ✅ **Hash-Chain**: Manipulationserkennung
- ✅ **Append-Only**: Unveränderbare Records

## Nächste Schritte

1. **Integration in Assessment-Responses**:
   - `justification` verschlüsselt speichern
   - `encrypted_record_id` Referenz hinzufügen

2. **Handler erweitern**:
   - CreateRecord beim Speichern von Begründungen
   - DecryptRecord beim Abrufen

3. **Testing**:
   ```bash
   go test ./internal/securestore/...
   go test ./internal/keymanager/...
   ```

## Dokumentation

Vollständige Dokumentation: [docs/ENCRYPTION.md](../docs/ENCRYPTION.md)

## Troubleshooting

**Vault nicht erreichbar?**
```bash
docker-compose ps vault
docker-compose logs vault
curl http://localhost:8200/v1/sys/health
```

**Migrationen fehlgeschlagen?**
```bash
docker-compose logs api
# Oder manuell:
cd backend
go run cmd/api/main.go
```

**Tests ausführen**
```bash
cd backend
go test ./internal/vault/...
go test ./internal/keymanager/...
go test ./internal/securestore/...
```
