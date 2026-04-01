# Verschluesselungs-Prozess fuer Tabellenspalten

## Uebersicht

Dieser Leitfaden beschreibt, wie sensible Felder (z. B. `justification`) ueber `encrypted_records` abgesichert werden.

## Architektur

### Key-Hierarchie

1. **Master Key**: `ENCRYPTION_MASTER_KEY` (32 Byte, 64 Hex-Zeichen)
2. **Process Keys**: pro Prozess, in DB verschluesselt gespeichert
3. **User Keys**: pro User (Ed25519), Private Key verschluesselt gespeichert

### Data Encryption Key (DEK)

Der DEK wird mit HKDF-SHA256 aus Kontext und Key-Material abgeleitet.

### Append-Only Encrypted Records

- Verschluesselte Nutzdaten liegen in `encrypted_records`
- Records sind append-only (DB-Trigger)
- Pro Record: ciphertext, nonce, tag, Ed25519-Signatur
- Hash-Chain pro Prozess fuer Tamper-Detection

## Implementierungs-Schritte

### Schritt 1: Migration vorbereiten

```sql
ALTER TABLE assessment_responses
ADD COLUMN encrypted_justification_id BIGINT
REFERENCES encrypted_records(id);

ALTER TABLE assessment_responses
ALTER COLUMN justification DROP NOT NULL;
```

### Schritt 2: Modell erweitern

```go
type AssessmentResponse struct {
    Justification            string `json:"justification" db:"-"`
    EncryptedJustificationID *int64 `json:"encrypted_justification_id,omitempty" db:"encrypted_justification_id"`
}
```

### Schritt 3: Service fuer verschluesselte Felder

```go
type EncryptedResponseService struct {
    db           *sql.DB
    responseRepo *repository.AssessmentResponseRepository
    keyManager   *keymanager.KeyManager
    secureStore  *securestore.SecureStore
}
```

Ablauf beim Speichern:
1. User- und Process-Key sicherstellen
2. `justification` als PlainData an `SecureStore.CreateRecord(...)`
3. erzeugte Record-ID in `encrypted_justification_id` speichern
4. Klartextfeld leeren

### Schritt 4: Business Logic anpassen

- Beim Schreiben immer ueber `EncryptedResponseService` gehen.
- Beim Lesen bei gesetzter `encrypted_justification_id` entschluesseln.
- Bei Entschluesselungsfehlern keine geheimen Inhalte loggen.

### Schritt 5: Repository-Queries bereinigen

- Klartextspalte aus SELECT/INSERT/UPDATE entfernen.
- `Scan()`-Signaturen entsprechend anpassen.

### Schritt 6: Klartextspalte sicher entfernen

```sql
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

### Schritt 7: Initialisierung in `main.go`

- `ENCRYPTION_MASTER_KEY` aus Config laden
- `keymanager.NewKeyManager(db, masterKey)` initialisieren
- `securestore.NewSecureStore(db, keyManager)` initialisieren
- Services mit aktivem Encryption-Stack verdrahten

## Wichtige Hinweise

### Slice Memory Issue vermeiden

```go
// Richtig: Kopieren statt geteilte Slices
encryptedData := make([]byte, len(ciphertext)-16)
copy(encryptedData, ciphertext[:len(ciphertext)-16])

tag := make([]byte, 16)
copy(tag, ciphertext[len(ciphertext)-16:])
```

### Sicherheit

- Master-Key niemals loggen
- Plaintext, DEK, Nonce, Tag nicht in Logs schreiben
- Secrets nur ueber sichere Secret-Distribution verteilen

### Monitoring

- Fehler mit Kontext loggen (record/process), aber ohne sensitive Daten
- Regelmaessige Hash-Chain-Validierung einplanen

## Testing

### Manuell

1. Response mit Begruendung speichern
2. In DB pruefen: Klartext leer, `encrypted_justification_id` gesetzt
3. Response lesen: Begruendung wird entschluesselt angezeigt
4. In `encrypted_records`: nonce=12 Bytes, tag=16 Bytes

### SQL-Check

```sql
SELECT id, length(encrypted_data), length(encryption_nonce), length(encryption_tag)
FROM encrypted_records
WHERE id = $1;
```

## Fehlerbehebung

### `cipher: message authentication failed`

- falscher Kontext/DEK
- korrumpierte Daten
- AAD stimmt nicht mit Encrypt-Aufruf ueberein

### `column does not exist`

- Migration oder Query-Anpassung unvollstaendig

### `encryption service not available`

- Service nicht verdrahtet
- `ENCRYPTION_MASTER_KEY` fehlt/ungueltig
