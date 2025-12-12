# Verschlüsselungs-Prozess für Tabellenspalten

## Übersicht

Dieser Prozess beschreibt, wie sensible Tabellenspalten verschlüsselt werden, um die Daten end-to-end zu schützen.

## Architektur

### 3-Tier Key Hierarchy

1. **System Master Key**: In Vault Transit Engine (AES-256-GCM)
2. **Process Keys**: Pro Assessment, verschlüsselt mit System Key
3. **User Keys**: Pro User (Ed25519 Keypair), Private Key verschlüsselt mit System Key

### Data Encryption Key (DEK)

Der DEK wird aus allen drei Schlüsselebenen abgeleitet:

```plain
DEK = SHA256(ProcessKey || UserKey.Seed() || "process:{processID}:user:{userID}")
```

### Append-Only Encrypted Records

- Alle verschlüsselten Daten werden in `encrypted_records` Tabelle gespeichert
- Records sind append-only (durch DB-Trigger geschützt)
- Jeder Record enthält: verschlüsselte Daten, Nonce, Tag, Ed25519 Signatur
- Hash-Chain über alle Records eines Prozesses für Tamper-Detection

## Implementierungs-Schritte

### Schritt 1: Migration erstellen

Füge eine Spalte hinzu, die auf `encrypted_records` verweist:

```sql
-- Beispiel: migrations/012_encrypt_justification.up.sql
ALTER TABLE assessment_responses 
ADD COLUMN encrypted_justification_id BIGINT 
REFERENCES encrypted_records(id);

-- Optional: Alte Spalte nullable machen für Migration
ALTER TABLE assessment_responses 
ALTER COLUMN justification DROP NOT NULL;
```

### Schritt 2: Model erweitern

```go
// In internal/models/models.go
type AssessmentResponse struct {
    // ... existing fields
    Justification            string    `json:"justification" db:"-"` // Nur für Display, nicht in DB
    EncryptedJustificationID *int64    `json:"encrypted_justification_id,omitempty" db:"encrypted_justification_id"`
}
```

### Schritt 3: Encrypted Service erstellen

```go
// Beispiel: internal/service/encrypted_response_service.go
type EncryptedResponseService struct {
    db           *sql.DB
    responseRepo *repository.AssessmentResponseRepository
    keyManager   *keymanager.KeyManager
    secureStore  *securestore.SecureStore
}

func (s *EncryptedResponseService) CreateResponse(response *models.AssessmentResponse, userID uint) error {
    // 1. Ensure keys exist
    processID := fmt.Sprintf("assessment-%d", response.AssessmentID)
    s.keyManager.ensureUserKey(int64(userID))
    s.keyManager.ensureProcessKey(processID)
    
    // 2. Encrypt data via SecureStore
    data := &securestore.PlainData{
        Fields: map[string]interface{}{
            "justification": response.Justification,
        },
        Metadata: map[string]string{
            "assessment_id": fmt.Sprintf("%d", response.AssessmentID),
            "category_id":   fmt.Sprintf("%d", response.CategoryID),
        },
    }
    
    record, err := s.secureStore.CreateRecord(
        processID,
        int64(userID),
        "JUSTIFICATION",
        data,
        "",
    )
    
    // 3. Store reference
    response.EncryptedJustificationID = &record.ID
    response.Justification = "" // Clear plaintext
    
    // 4. Insert into main table
    query := `INSERT INTO assessment_responses (..., encrypted_justification_id) VALUES (..., $N)`
    // Execute query
    
    return nil
}

func (s *EncryptedResponseService) DecryptJustification(encryptedJustificationID int64) (string, error) {
    plainData, err := s.secureStore.DecryptRecord(encryptedJustificationID)
    if err != nil {
        return "", err
    }
    
    if justification, ok := plainData.Fields["justification"].(string); ok {
        return justification, nil
    }
    
    return "", fmt.Errorf("justification field not found")
}
```

### Schritt 4: Business Logic anpassen

```go
// In SaveResponse
if s.encryptedResponseSvc == nil {
    return nil, fmt.Errorf("encryption service not available")
}

// Create/Update via EncryptedResponseService
if existing != nil {
    response.ID = existing.ID
    err = s.encryptedResponseSvc.UpdateResponse(response, userID)
} else {
    err = s.encryptedResponseSvc.CreateResponse(response, userID)
}

// In GetResponses
for i := range responses {
    if responses[i].EncryptedJustificationID != nil {
        decrypted, err := s.encryptedResponseSvc.DecryptJustification(*responses[i].EncryptedJustificationID)
        if err != nil {
            slog.Error("Failed to decrypt", "error", err)
            responses[i].Justification = "[Decryption failed]"
        } else {
            responses[i].Justification = decrypted
        }
    }
}
```

### Schritt 5: Repository Queries anpassen

```go
// Entferne die alte Spalte aus allen SELECT Queries
// ALTE Version:
SELECT id, ..., justification, encrypted_justification_id FROM table

// NEUE Version:
SELECT id, ..., encrypted_justification_id FROM table

// Wichtig: Entferne auch aus Scan() calls!
```

### Schritt 6: Migration für Spalten-Entfernung

```sql
-- migrations/013_remove_justification_column.up.sql
-- Safety check: Fail if unencrypted data exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM assessment_responses 
        WHERE justification IS NOT NULL 
        AND encrypted_justification_id IS NULL
    ) THEN
        RAISE EXCEPTION 'Cannot drop column: unencrypted justifications exist';
    END IF;
END $$;

ALTER TABLE assessment_responses DROP COLUMN justification;
```

### Schritt 7: main.go Initialisierung

```go
// Initialize encryption services if Vault is enabled
if config.GetBool("VAULT_ENABLED") {
    vaultClient := vault.NewClient(&vault.Config{...})
    keyManager := keymanager.NewKeyManager(db.DB, vaultClient)
    secureStore := securestore.NewSecureStore(db.DB, keyManager)
    encryptedResponseSvc := service.NewEncryptedResponseService(
        db.DB, responseRepo, keyManager, secureStore
    )
    
    // Pass to business service
    selfAssessmentSvc := service.NewSelfAssessmentService(
        ..., encryptedResponseSvc
    )
}
```

## Wichtige Hinweise

### ⚠️ Slice Memory Issue

Beim Aufteilen von Ciphertext in Data und Tag IMMER kopieren, nicht slicen:

```go
// FALSCH - Slices teilen sich das zugrunde liegende Array
encryptedData := ciphertext[:len-16]
tag := ciphertext[len-16:]

// RICHTIG - Explizit kopieren
encryptedData := make([]byte, len-16)
copy(encryptedData, ciphertext[:len-16])
tag := make([]byte, 16)
copy(tag, ciphertext[len-16:])
```

### 🔐 Sicherheit

- System Master Key nie im Code/Logs ausgeben
- DEK, Nonces, Tags nie loggen
- Plaintext Daten nie in Logs ausgeben
- Vault Token sicher speichern (Umgebungsvariable)

### 🔄 Key Rotation

Process Keys können rotiert werden:

```go
keyManager.RotateProcessKey(processID)
```

Alte Records bleiben mit altem Key verschlüsselt (append-only).

### 📊 Monitoring

Wichtige Logs mit slog:

```go
slog.Error("Failed to decrypt", "error", err, "record_id", id)
slog.Info("Encryption service initialized", "vault_addr", addr)
```

## Testing

### Manuell testen

1. Response mit Justification speichern
2. In DB prüfen: `justification` leer, `encrypted_justification_id` gesetzt
3. Response abrufen: Justification sollte entschlüsselt erscheinen
4. In `encrypted_records`: Record mit korrekten Längen (Nonce: 12, Tag: 16)

### Datenbank-Prüfung

```sql
-- Check encrypted record
SELECT id, length(encrypted_data), length(encryption_nonce), length(encryption_tag)
FROM encrypted_records WHERE id = X;

-- Should return: nonce=12, tag=16
```

## Vault Setup

Vault muss im Dev-Modus oder mit persistentem Storage laufen:

```bash
docker-compose up -d vault
# Vault läuft auf localhost:8200
# Transit Engine wird automatisch beim Start gemountet
```

## Fehlerbehebung

### "cipher: message authentication failed"

- DEK stimmt nicht → Keys prüfen (User/Process)
- Tag korrumpiert → Slice-Kopie Problem (siehe oben)
- Additional Data unterschiedlich → Format prüfen

### "column does not exist"

- Repository Queries noch nicht angepasst
- Migration noch nicht ausgeführt

### "encryption service not available"

- Vault nicht erreichbar
- VAULT_ENABLED=false in .env
- KeyManager nicht initialisiert in main.go
