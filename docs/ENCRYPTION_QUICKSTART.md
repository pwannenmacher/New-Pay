# Verschluesselungssystem - Quick Start

## Was ist implementiert?

New Pay nutzt ein lokales Key-Management ohne externen Schluessel-Dienst:

- `ENCRYPTION_MASTER_KEY` (32 Bytes / 64 Hex-Zeichen) als Root-Key
- AES-256-GCM fuer sensible Inhalte
- Ed25519 fuer Signaturen
- Hash-Chain fuer manipulationssicheren Audit-Trail

## Komponenten

1. **Backend Services**
   - `internal/crypto/`: AES-256-GCM und HKDF-SHA256
   - `internal/keymanager/`: User- und Process-Keys
   - `internal/securestore/`: Encrypt/Decrypt + Signatur + Hash-Chain

2. **Datenbank**
   - Migration `011_encryption_tables`: `user_keys`, `process_keys`, `encrypted_records`

## Verwendung

### 1. Master-Key erzeugen

```bash
openssl rand -hex 32
```

Wert als `ENCRYPTION_MASTER_KEY` in `.env` eintragen.

### 2. Services starten

```bash
cd docker/development
docker compose up -d
```

### 3. Anwendung starten

```bash
cd backend
go run main.go
```

Migrationen laufen beim Start automatisch.

## Environment Variables

```bash
ENCRYPTION_MASTER_KEY=<64-hex-chars>
```

Hinweise:
- Den Key niemals committen.
- Verlust des Keys macht verschluesselte Daten dauerhaft unlesbar.

## Im Code verwenden

```go
import (
    "new-pay/internal/keymanager"
    "new-pay/internal/securestore"
)

keyManager, _ := keymanager.NewKeyManager(db, masterKey)
store := securestore.NewSecureStore(db, keyManager)

publicKey, _ := keyManager.CreateUserKey(userID)
_ = publicKey

_ = keyManager.CreateProcessKey(processID, nil)

data := &securestore.PlainData{
    Fields: map[string]interface{}{
        "justification": "Meine Begruendung...",
    },
}
record, _ := store.CreateRecord(processID, userID, "JUSTIFICATION", data, "")

plainData, _ := store.DecryptRecord(record.ID)
_ = plainData
```

## Sicherheitsmerkmale

- AES-256-GCM (AEAD)
- HKDF-SHA256 fuer DEK-Ableitung
- Ed25519-Signaturen
- Hash-Chain zur Manipulationserkennung
- Append-Only Storage (`encrypted_records`)

## Troubleshooting

**`ENCRYPTION_MASTER_KEY` fehlt oder ungueltig?**

- Muss exakt 64 Hex-Zeichen haben.
- Beispielgenerator: `openssl rand -hex 32`

**Tests ausfuehren**

```bash
cd backend
go test ./internal/keymanager/...
go test ./internal/securestore/...
```

## Dokumentation

Vollstaendige Dokumentation: [docs/ENCRYPTION.md](../docs/ENCRYPTION.md)
