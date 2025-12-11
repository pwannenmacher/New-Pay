# Verschlüsselung und Key Management

## Überblick

Die New Pay Plattform implementiert ein sicheres Verschlüsselungssystem für sensible Daten wie Begründungen (justifications), Kommentare und Ergebnisdokumentationen. Das System kombiniert mehrere kryptografische Techniken:

- **3-stufige Key-Hierarchie**: System-Key + User-Key + Process-Key
- **Verschlüsselung**: AES-256-GCM (Authenticated Encryption)
- **Digitale Signaturen**: Ed25519 für Authentizität
- **Hash-Chain**: SHA-256 für Manipulationserkennung
- **Append-Only**: Unveränderbare Audit-Trail

## Architektur

```plain
┌─────────────────────────────────────────────────────────────┐
│                    HashiCorp Vault (KMS)                     │
│  - System Master Key (AES-256)                               │
│  - Transit Encryption Engine                                 │
│  - Key Rotation Support                                      │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                     Key Manager Service                      │
│  - User Keys (Ed25519 Keypairs, encrypted with System Key)  │
│  - Process Keys (AES-256, encrypted with System Key)        │
│  - Key Derivation (HKDF)                                     │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                    Secure Store Service                      │
│  - Data Encryption (AES-256-GCM)                            │
│  - Digital Signatures (Ed25519)                              │
│  - Hash Chain Audit Trail                                    │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   PostgreSQL Database                        │
│  - user_keys: Verschlüsselte User-Keypairs                  │
│  - process_keys: Verschlüsselte Process-Keys                │
│  - encrypted_records: Verschlüsselte Daten + Signaturen     │
└─────────────────────────────────────────────────────────────┘
```

## Key-Hierarchie

### 1. System Master Key

- **Speicherort**: HashiCorp Vault
- **Typ**: AES-256-GCM
- **Verwendung**: Verschlüsselung aller User- und Process-Keys
- **Rotation**: Unterstützt durch Vault Key Versioning

### 2. User Keys

- **Typ**: Ed25519 Keypair (Public + Private)
- **Speicherort**: PostgreSQL (Private Key verschlüsselt mit System Key)
- **Verwendung**:
  - Private Key: Digitale Signaturen erstellen
  - Public Key: Signaturen verifizieren
  - Seed: Teil der Key Derivation für Data Encryption Key

### 3. Process Keys

- **Typ**: 256-bit symmetrischer Schlüssel
- **Speicherort**: PostgreSQL (verschlüsselt mit System Key)
- **Verwendung**: Isolierung von Datenströmen (z.B. pro Self-Assessment)
- **Expiration**: Optional, für zeitlich begrenzte Vorgänge

### 4. Data Encryption Key (DEK)

- **Ableitung**: `SHA256(Process-Key || User-Key-Seed || "process:ID:user:ID")`
- **Verwendung**: Einmalig für jede Verschlüsselungsoperation
- **Speicherung**: Nicht gespeichert, wird bei Bedarf neu abgeleitet

## Datenbank-Schema

### user_keys

```sql
CREATE TABLE user_keys (
    user_id BIGINT PRIMARY KEY,
    public_key TEXT NOT NULL,              -- Ed25519 Public Key (hex)
    encrypted_private_key TEXT NOT NULL,   -- Ed25519 Private Key (encrypted)
    key_version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL
);
```

### process_keys

```sql
CREATE TABLE process_keys (
    process_id VARCHAR(100) PRIMARY KEY,
    encrypted_key_material TEXT NOT NULL,  -- AES-256 Key (encrypted)
    key_hash VARCHAR(64) NOT NULL,         -- SHA-256 Hash für Verifikation
    created_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP                   -- Optional: Key Expiration
);
```

### encrypted_records

```sql
CREATE TABLE encrypted_records (
    id BIGSERIAL PRIMARY KEY,
    process_id VARCHAR(100) NOT NULL,
    user_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    
    -- Verschlüsselte Daten (AES-256-GCM)
    encrypted_data BYTEA NOT NULL,
    encryption_nonce BYTEA NOT NULL,
    encryption_tag BYTEA NOT NULL,
    
    -- Key Metadata
    key_version INT NOT NULL DEFAULT 1,
    system_key_id VARCHAR(50) NOT NULL,
    process_key_hash VARCHAR(64) NOT NULL,
    
    -- Digitale Signatur (Ed25519)
    data_signature TEXT NOT NULL,
    signature_public_key TEXT NOT NULL,
    
    -- Metadata (unverschlüsselt für Queries)
    record_type VARCHAR(50),
    status VARCHAR(50),
    
    -- Hash Chain für Audit Trail
    prev_record_hash VARCHAR(64),
    chain_hash VARCHAR(64) NOT NULL UNIQUE
);
```

**Trigger**: Verhindert UPDATE/DELETE (Append-Only)

## Verschlüsselungsablauf

### Daten speichern

1. **Key Derivation**

   ```go
   dek := SHA256(processKey || userKeySeed || contextInfo)
   ```

2. **Verschlüsselung**

   ```go
   ciphertext, nonce, tag := AES-256-GCM.Encrypt(plaintext, dek, additionalData)
   ```

3. **Signatur**

   ```go
   signature := Ed25519.Sign(userPrivateKey, ciphertext || nonce || tag)
   ```

4. **Hash Chain**

   ```go
   chainHash := SHA256(prevHash || signature || userID || processID || timestamp)
   ```

5. **Speicherung**
   - Alle Komponenten werden in `encrypted_records` gespeichert
   - Record ist unveränderlich (Trigger verhindert Änderungen)

### Daten entschlüsseln

1. **Signatur verifizieren**

   ```go
   valid := Ed25519.Verify(publicKey, ciphertext || nonce || tag, signature)
   ```

2. **Key Derivation** (identisch wie beim Verschlüsseln)

3. **Entschlüsselung**

   ```go
   plaintext := AES-256-GCM.Decrypt(ciphertext, dek, nonce, tag, additionalData)
   ```

## Sicherheitsmerkmale

### ✅ Authenticated Encryption (AEAD)

- AES-256-GCM garantiert Vertraulichkeit UND Integrität
- Authentication Tag verhindert unbemerkte Manipulation
- Nonce verhindert Replay-Angriffe

### ✅ Digitale Signaturen

- Ed25519: Schnell, klein, sicher (128-bit Sicherheit)
- Authentizität: Nur der User mit Private Key kann signieren
- Non-Repudiation: User kann Erstellung nicht abstreiten

### ✅ Hash Chain Audit Trail

- Jeder Record verlinkt auf vorherigen via `prev_hash`
- Manipulation bricht die Kette → sofort erkennbar
- Genesis Block: `0000...0000` (64 Nullen)

### ✅ Append-Only

- PostgreSQL Trigger verhindert UPDATE/DELETE
- Vollständige Historie bleibt erhalten
- Compliance-ready (DSGVO, GoBD)

### ✅ Key Separation

- System-Key: Nur in Vault, niemals in DB oder App-Code
- User-Keys: Pro User isoliert
- Process-Keys: Pro Vorgang isoliert
- DEK: Wird nicht gespeichert, nur abgeleitet

### ✅ Automatische Entschlüsselung

- Keine User-Interaktion nötig
- Backend kann Daten entschlüsseln und weitergeben
- Vault-Token wird von Backend verwaltet

## HashiCorp Vault Integration

### Konfiguration

**.env**

```bash
VAULT_ENABLED=true
VAULT_ADDR=http://vault:8200
VAULT_TOKEN=dev-root-token  # Produktion: Vault AppRole oder Kubernetes Auth
VAULT_TRANSIT_MOUNT=transit
```

### Vault Services

```go
// Vault Client initialisieren
vaultClient, err := vault.NewClient(&vault.Config{
    Address:      cfg.Vault.Address,
    Token:        cfg.Vault.Token,
    TransitMount: cfg.Vault.TransitMount,
})

// System Master Key erstellen
err = vaultClient.CreateKey("system-master-key", "aes256-gcm96")

// Daten verschlüsseln via Transit Engine
ciphertext, err := vaultClient.Encrypt("system-master-key", plaintext, context)

// Daten entschlüsseln via Transit Engine
plaintext, err := vaultClient.Decrypt("system-master-key", ciphertext, context)
```

### Transit Engine Vorteile

- ✅ **Key Rotation**: Alte Versionen bleiben funktionsfähig
- ✅ **Key Deletion**: Sichere Vernichtung von Keys
- ✅ **Audit Log**: Alle Operationen werden protokolliert
- ✅ **HSM Support**: Optional Hardware Security Module
- ✅ **High Availability**: Vault Clustering

## Verwendungsbeispiel

### Setup

```go
import (
    "github.com/pwannenmacher/new-pay-gh/backend/internal/vault"
    "github.com/pwannenmacher/new-pay-gh/backend/internal/keymanager"
    "github.com/pwannenmacher/new-pay-gh/backend/internal/securestore"
)

// 1. Vault Client
vaultClient, err := vault.NewClient(&vault.Config{
    Address:      "http://localhost:8200",
    Token:        "dev-root-token",
    TransitMount: "transit",
})

// 2. Key Manager
keyManager, err := keymanager.NewKeyManager(db, vaultClient)

// 3. Secure Store
store := securestore.NewSecureStore(db, keyManager)
```

### User Key erstellen

```go
publicKey, err := keyManager.CreateUserKey(userID)
// Public Key: für Signatur-Verifikation
// Private Key: verschlüsselt in DB gespeichert
```

### Process Key erstellen

```go
err := keyManager.CreateProcessKey(processID, nil) // nil = kein Expiration
```

### Daten verschlüsseln und speichern

```go
data := &securestore.PlainData{
    Fields: map[string]interface{}{
        "justification": "Begründung für die Selbsteinschätzung...",
        "details": "Weitere Details...",
    },
    Metadata: map[string]string{
        "assessment_id": "12345",
    },
}

record, err := store.CreateRecord(
    processID,
    userID,
    "JUSTIFICATION",
    data,
    "active",
)

// Record enthält:
// - Verschlüsselte Daten
// - Signatur
// - Hash Chain Verlinkung
```

### Daten entschlüsseln

```go
// Automatisch ohne User-Interaktion
plainData, err := store.DecryptRecord(recordID)

// plainData.Fields["justification"] -> "Begründung für..."
```

### Hash Chain verifizieren

```go
valid, messages, err := store.VerifyChain(processID)
if !valid {
    for _, msg := range messages {
        log.Warn(msg) // z.B. "chain broken at record 42"
    }
}
```

## Migration von bestehenden Daten

Für bestehende unverschlüsselte `justification`-Felder:

```sql
-- 1. Backup erstellen
CREATE TABLE justifications_backup AS 
SELECT * FROM assessment_responses;

-- 2. Verschlüsselung über Go-Script
-- (für jeden Record: lesen, verschlüsseln, in encrypted_records speichern)

-- 3. Referenz aktualisieren
ALTER TABLE assessment_responses 
ADD COLUMN encrypted_record_id BIGINT REFERENCES encrypted_records(id);

-- 4. justification auf NULL setzen (oder Spalte droppen)
UPDATE assessment_responses SET justification = NULL;
```

## Deployment

### Docker Compose

```yaml
vault:
  image: hashicorp/vault:1.18
  ports:
    - "8200:8200"
  environment:
    VAULT_DEV_ROOT_TOKEN_ID: dev-root-token
    VAULT_DEV_LISTEN_ADDRESS: 0.0.0.0:8200
  volumes:
    - vault-file:/vault/file
    - vault-logs:/vault/logs
  command: server -dev
```

### Produktion

**Wichtig**: Dev-Mode ist NICHT für Produktion geeignet!

1. **Vault in Produktion deployen**:

   ```bash
   vault operator init
   vault operator unseal (3x mit verschiedenen Unseal Keys)
   ```

2. **AppRole Auth einrichten**:

   ```bash
   vault auth enable approle
   vault write auth/approle/role/newpay-api \
       secret_id_ttl=24h \
       token_ttl=1h \
       token_max_ttl=4h
   ```

3. **Backend mit AppRole Token**:

   ```go
   VAULT_TOKEN=$(vault write -field=token auth/approle/login \
       role_id=$ROLE_ID secret_id=$SECRET_ID)
   ```

4. **TLS aktivieren**:

   ```bash
   VAULT_ADDR=https://vault.example.com:8200
   ```

## Monitoring

### Vault Health Check

```go
if err := vaultClient.Health(); err != nil {
    log.Error("Vault is unhealthy: %v", err)
    // Fallback oder Alert
}
```

### Audit Logs

Vault speichert alle Operationen:

```bash
vault audit enable file file_path=/vault/logs/audit.log
```

### Hash Chain Verifikation (Cronjob)

```go
// Täglich alle Process-Chains verifizieren
for _, processID := range allProcessIDs {
    valid, errs, err := store.VerifyChain(processID)
    if !valid {
        alert.Send("Chain verification failed for " + processID)
    }
}
```

## Performance

### Key Caching

KeyManager cached System-Keys im Memory:

```go
systemKeyCache map[string][]byte
```

### Batch-Operations

Für große Datenmengen:

```go
// TODO: Batch-Encrypt API
records, err := store.CreateRecordsBatch(processID, userID, dataSlice)
```

### Index-Optimierung

```sql
CREATE INDEX idx_encrypted_records_process_user 
ON encrypted_records(process_id, user_id);

CREATE INDEX idx_encrypted_records_created 
ON encrypted_records(created_at DESC);
```

## Troubleshooting

### Vault nicht erreichbar

```plain
Error: failed to connect to Vault: connection refused
```

**Lösung**: Prüfe `VAULT_ADDR` und Vault Container Status:

```bash
docker-compose ps vault
curl http://localhost:8200/v1/sys/health
```

### Signatur-Verifikation fehlgeschlagen

```plain
Error: signature verification failed - data may be tampered
```

**Ursachen**:

- Daten wurden manipuliert (KRITISCH!)
- Falscher Public Key
- Falscher User Key verwendet

**Lösung**: Hash Chain verifizieren, Log prüfen

### Key nicht gefunden

```plain
Error: user key not found
```

**Lösung**: User Key erstellen:

```go
publicKey, err := keyManager.CreateUserKey(userID)
```

## Weitere Entwicklung

Geplante Features:

- [ ] **Batch-Operationen**: Mehrere Records gleichzeitig ver-/entschlüsseln
- [ ] **Key Rotation**: Automatische Rotation von Process-Keys
- [ ] **External Timestamping**: RFC 3161 für rechtssichere Zeitstempel
- [ ] **Read-Only Replicas**: Zusätzliche Datensicherheit
- [ ] **Merkle Tree**: Effiziente Batch-Verifikation der Hash-Chain
- [ ] **Access Control**: Granulare Berechtigungen für Process-Key-Zugriff

## Referenzen

- [HashiCorp Vault Documentation](https://www.vaultproject.io/docs)
- [AES-GCM (NIST)](https://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-38d.pdf)
- [Ed25519 (RFC 8032)](https://datatracker.ietf.org/doc/html/rfc8032)
- [Go crypto/ed25519](https://pkg.go.dev/crypto/ed25519)
- [Go crypto/cipher#NewGCM](https://pkg.go.dev/crypto/cipher#NewGCM)
