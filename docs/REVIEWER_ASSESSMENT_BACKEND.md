# Reviewer Assessment System - Backend Requirements

## Überblick

Dieses Dokument beschreibt die Backend-Anforderungen für das Reviewer-Assessment-System, das es Reviewern ermöglicht, Selbsteinschätzungen von Benutzern zu bewerten.

## Datenmodell

### Neue Tabelle: `reviewer_responses`

```sql
CREATE TABLE reviewer_responses (
    id SERIAL PRIMARY KEY,
    assessment_id INTEGER NOT NULL REFERENCES self_assessments(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    reviewer_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    path_id INTEGER NOT NULL REFERENCES paths(id) ON DELETE CASCADE,
    level_id INTEGER NOT NULL REFERENCES levels(id) ON DELETE CASCADE,
    encrypted_justification_id BIGINT REFERENCES encrypted_records(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(assessment_id, category_id, reviewer_user_id)
);

CREATE INDEX idx_reviewer_responses_assessment ON reviewer_responses(assessment_id);
CREATE INDEX idx_reviewer_responses_reviewer ON reviewer_responses(reviewer_user_id);
CREATE INDEX idx_reviewer_responses_encrypted_justification ON reviewer_responses(encrypted_justification_id);
```

### Felder

- `id`: Primärschlüssel
- `assessment_id`: Referenz zur Selbsteinschätzung
- `category_id`: Referenz zur bewerteten Kategorie
- `reviewer_user_id`: Benutzer-ID des Reviewers
- `path_id`: Vom Reviewer gewählter Entwicklungspfad (kann vom User-Pfad abweichen)
- `level_id`: Vom Reviewer gewähltes Level
- `encrypted_justification_id`: Referenz zu `encrypted_records` für verschlüsselte Begründung (erforderlich wenn abweichend vom User-Level ODER User-Pfad)
- `created_at`: Erstellungszeitpunkt
- `updated_at`: Zeitpunkt der letzten Änderung

## Verschlüsselung

### Überblick

Reviewer-Begründungen werden mit demselben Verschlüsselungssystem wie Self-Assessment-Begründungen geschützt:

- **3-stufige Key-Hierarchie**: System-Key + User-Key (Reviewer) + Process-Key (Assessment)
- **Verschlüsselung**: AES-256-GCM
- **Digitale Signaturen**: Ed25519
- **Speicherung**: `encrypted_records` Tabelle

### Process-Key

Reviewer-Responses verwenden denselben Process-Key wie die zugehörige Self-Assessment:

```
Process-ID: "assessment-{assessment_id}"
```

### Verschlüsselungs-Flow

1. **Beim Erstellen/Aktualisieren einer Reviewer-Response:**
   - Process-Key für Assessment sicherstellen
   - User-Key für Reviewer sicherstellen
   - Begründung über `securestore.CreateRecord()` verschlüsseln
   - Record-ID in `encrypted_justification_id` speichern
   - Plaintext-`justification` leer lassen

2. **Beim Abrufen einer Reviewer-Response:**
   - Record über `securestore.GetRecord()` entschlüsseln
   - Begründung in Response-Objekt einfügen (nur für Display)

### Service-Integration

```go
// Beispiel in reviewer_service.go
func (s *ReviewerService) CreateResponse(response *models.ReviewerResponse) error {
    if response.Justification != "" {
        // Encrypt justification
        processID := fmt.Sprintf("assessment-%d", response.AssessmentID)
        
        data := &securestore.PlainData{
            Fields: map[string]interface{}{
                "justification": response.Justification,
            },
            Metadata: map[string]string{
                "assessment_id": fmt.Sprintf("%d", response.AssessmentID),
                "category_id":   fmt.Sprintf("%d", response.CategoryID),
                "reviewer_id":   fmt.Sprintf("%d", response.ReviewerUserID),
            },
        }
        
        record, err := s.secureStore.CreateRecord(
            processID,
            int64(response.ReviewerUserID),
            "REVIEWER_JUSTIFICATION",
            data,
            "",
        )
        if err != nil {
            return err
        }
        
        response.EncryptedJustificationID = &record.ID
        response.Justification = "" // Clear plaintext
    }
    
    return s.repo.CreateOrUpdate(response)
}

func (s *ReviewerService) GetResponseWithDecryption(assessmentID, categoryID uint) (*models.ReviewerResponse, error) {
    response, err := s.repo.GetByCategoryID(assessmentID, categoryID)
    if err != nil {
        return nil, err
    }
    
    if response.EncryptedJustificationID != nil {
        record, err := s.secureStore.GetRecord(*response.EncryptedJustificationID)
        if err != nil {
            return nil, err
        }
        
        response.Justification = record.Data.Fields["justification"].(string)
    }
    
    return response, nil
}
```

Siehe auch:
- [docs/ENCRYPTION.md](ENCRYPTION.md) für vollständige Verschlüsselungs-Architektur
- [docs/ENCRYPTION_PROCESS.md](ENCRYPTION_PROCESS.md) für Implementierungs-Details
- Migration `012_encrypt_justification.up.sql` für Referenz-Implementierung

## API-Endpunkte

### 1. Reviewer-Antworten abrufen

**Endpoint:** `GET /api/v1/review/assessment/:id/responses`

**Beschreibung:** Lädt die Reviewer-Antworten des aktuellen Reviewers für eine Selbsteinschätzung

**Authentifizierung:** JWT (Rolle: reviewer oder admin)

**Wichtig:** Reviewer sehen **nur ihre eigenen** Review-Antworten. Antworten anderer Reviewer sind niemals sichtbar. Admins können alle Reviews sehen.

**Response:**

```json
[
  {
    "id": 1,
    "assessment_id": 123,
    "category_id": 5,
    "reviewer_user_id": 10,
    "level_id": 3,
    "justification": "Der Benutzer hat die Anforderungen von Level 4 noch nicht vollständig erfüllt...",
    "created_at": "2025-12-15T10:30:00Z",
    "updated_at": "2025-12-15T10:30:00Z"
  }
]
```

### 2. Reviewer-Antwort speichern/aktualisieren

**Endpoint:** `POST /api/v1/review/assessment/:id/responses`

**Beschreibung:** Speichert oder aktualisiert die Reviewer-Bewertung für eine Kategorie

**Authentifizierung:** JWT (Rolle: reviewer oder admin)

**Request Body:**

```json
{
  "category_id": 5,
  "path_id": 12,
  "level_id": 3,
  "justification": "Begründung..."
}
```

**Validierung:**

- `category_id`: Erforderlich, muss zur Assessment gehören
- `path_id`: Erforderlich, muss gültiger Pfad in der Kategorie sein
- `level_id`: Erforderlich, muss gültiges Level sein
- `justification`:
  - Optional wenn `level_id` == User-Level UND `path_id` == User-Pfad
  - Erforderlich (min. 50 Zeichen) wenn `level_id` != User-Level ODER `path_id` != User-Pfad

**Response:**

```json
{
  "id": 1,
  "assessment_id": 123,
  "category_id": 5,
  "reviewer_user_id": 10,
  "path_id": 12,
  "level_id": 3,
  "justification": "...",
  "created_at": "2025-12-15T10:30:00Z",
  "updated_at": "2025-12-15T10:30:00Z"
}
```

**Fehler:**

- `400`: Validierungsfehler (z.B. Begründung zu kurz bei Abweichung von Level oder Pfad)
- `403`: Keine Berechtigung (nicht reviewer/admin) ODER Versuch eigenes Assessment zu prüfen
- `404`: Assessment nicht gefunden
- `403`: Keine Berechtigung (nicht reviewer/admin)
- `404`: Assessment nicht gefunden

### 3. Reviewer-Antwort löschen

**Endpoint:** `DELETE /api/v1/review/assessment/:id/responses/:category_id`

**Beschreibung:** Löscht die Reviewer-Bewertung für eine Kategorie

**Authentifizierung:** JWT (Rolle: reviewer oder admin)

**Response:**

```json
{
  "message": "Reviewer response deleted successfully"
}
```

### 4. Review abschließen

**Endpoint:** `POST /api/v1/review/assessment/:id/complete`

**Beschreibung:** Markiert das Review als abgeschlossen und ändert den Assessment-Status

**Authentifizierung:** JWT (Rolle: reviewer oder admin)

**Validierung:**

- Alle Kategorien müssen eine Reviewer-Antwort haben
- Alle erforderlichen Begründungen müssen vorhanden sein (min. 50 Zeichen bei Abweichung)

**Request Body:**

```json
{
  "new_status": "reviewed"  // oder "discussion"
}
```

**Response:**

```json
{
  "message": "Review completed successfully",
  "assessment": {
    "id": 123,
    "status": "reviewed",
    "reviewed_at": "2025-12-15T10:30:00Z"
  }
}
```

**Fehler:**

- `400`: Unvollständiges Review oder Validierungsfehler
- `403`: Keine Berechtigung
- `404`: Assessment nicht gefunden

## Backend-Logik

### Service Layer (`reviewer_service.go`)

#### Validierung

```go
func ValidateReviewerResponse(userLevelID, reviewerLevelID uint, justification string) error {
    if userLevelID != reviewerLevelID && len(justification) < 50 {
        return errors.New("justification must be at least 50 characters when deviating from user's level")
    }
    return nil
}
```

#### Prüfung der Vollständigkeit

```go
func IsReviewComplete(assessmentID uint) (bool, error) {
    // Prüfe ob für alle Kategorien mit User-Antworten auch Reviewer-Antworten existieren
    // Prüfe ob alle erforderlichen Begründungen vorhanden sind
}
```

### Repository Layer (`reviewer_repository.go`)

#### Methoden

```go
type ReviewerRepository interface {
    CreateOrUpdateResponse(response *ReviewerResponse) error
    // GetResponsesByAssessmentID lädt nur Responses des spezifischen Reviewers (außer für Admins)
    GetResponsesByAssessmentID(assessmentID, reviewerUserID uint) ([]ReviewerResponse, error)
    // GetResponseByCategoryID lädt nur Response des spezifischen Reviewers (außer für Admins)
    GetResponseByCategoryID(assessmentID, categoryID, reviewerUserID uint) (*ReviewerResponse, error)
    DeleteResponse(assessmentID, categoryID, reviewerUserID uint) error
    // GetResponsesWithUserComparison vergleicht User-Antworten mit den Antworten des aktuellen Reviewers
    GetResponsesWithUserComparison(assessmentID, reviewerUserID uint) ([]ResponseComparison, error)
}
```

### Handler Layer (`reviewer_handler.go`)

#### Berechtigungsprüfung

- Nur Benutzer mit Rolle "reviewer" oder "admin" dürfen Reviews durchführen
- Jeder Reviewer kann nur seine eigenen Reviews bearbeiten (außer Admins)

## Migration

### Migration File: `015_reviewer_responses.up.sql`

```sql
CREATE TABLE reviewer_responses (
    id SERIAL PRIMARY KEY,
    assessment_id INTEGER NOT NULL REFERENCES self_assessments(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    reviewer_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    path_id INTEGER NOT NULL REFERENCES paths(id) ON DELETE CASCADE,
    level_id INTEGER NOT NULL REFERENCES levels(id) ON DELETE CASCADE,
    encrypted_justification_id BIGINT REFERENCES encrypted_records(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(assessment_id, category_id, reviewer_user_id)
);

CREATE INDEX idx_reviewer_responses_assessment ON reviewer_responses(assessment_id);
CREATE INDEX idx_reviewer_responses_reviewer ON reviewer_responses(reviewer_user_id);
CREATE INDEX idx_reviewer_responses_encrypted_justification ON reviewer_responses(encrypted_justification_id);

COMMENT ON COLUMN reviewer_responses.encrypted_justification_id IS 'Reference to encrypted justification in encrypted_records table';
```

### Migration File: `015_reviewer_responses.down.sql`

```sql
DROP INDEX IF EXISTS idx_reviewer_responses_encrypted_justification;
DROP INDEX IF EXISTS idx_reviewer_responses_reviewer;
DROP INDEX IF EXISTS idx_reviewer_responses_assessment;
DROP TABLE IF EXISTS reviewer_responses;
```

## Audit Logging

Alle Reviewer-Aktionen sollten im Audit-Log protokolliert werden:

- `reviewer.response.create`: Neue Reviewer-Antwort erstellt
- `reviewer.response.update`: Reviewer-Antwort aktualisiert
- `reviewer.response.delete`: Reviewer-Antwort gelöscht
- `reviewer.assessment.complete`: Review abgeschlossen

## Sicherheit

### Zugriffskontrolle

1. Nur Benutzer mit Rolle "reviewer" oder "admin" dürfen auf Review-Endpunkte zugreifen
2. Reviewer können nur offene Assessments (status: submitted, in_review, reviewed, discussion) bewerten
3. **TODO: WICHTIG** - Reviewer können nicht ihre eigenen Assessments bewerten (prüfen: reviewer_user_id != assessment.user_id)
   - Diese Prüfung muss in ALLEN Reviewer-Endpunkten implementiert werden:
     - `GET /api/v1/review/assessment/:id/responses`
     - `POST /api/v1/review/assessment/:id/responses`
     - `DELETE /api/v1/review/assessment/:id/responses/:category_id`
     - `POST /api/v1/review/assessment/:id/complete`
   - Fehlercode: `403 Forbidden` mit Meldung "Cannot review your own assessment"
   - Im Frontend bereits implementiert, Backend-Validierung fehlt noch
4. **WICHTIG: Datenisolierung zwischen Reviewern**
   - Reviewer dürfen **nur ihre eigenen** Review-Antworten sehen und bearbeiten
   - Review-Antworten anderer Reviewer sind **niemals** zugänglich (auch nicht lesend)
   - Admins können alle Review-Antworten aller Reviewer einsehen
   - Implementierung: Alle GET/POST/DELETE Endpunkte müssen `reviewer_user_id = current_user.id` filtern (außer für Admins)

### Datenintegrität

1. Reviewer-Antworten können nur für existierende Assessments und Kategorien erstellt werden
2. Level-ID muss zu einem gültigen Level im Katalog gehören
3. Bei Abweichung vom User-Level ist eine Begründung von mindestens 50 Zeichen erforderlich
4. **TODO:** Bei Abweichung vom User-Pfad (path_id) ist ebenfalls eine Begründung von mindestens 50 Zeichen erforderlich
5. Alle Begründungen müssen verschlüsselt in `encrypted_records` gespeichert werden

### Verschlüsselungs-Sicherheit

1. Reviewer-Begründungen verwenden den Process-Key der zugehörigen Self-Assessment
2. Signatur erfolgt mit dem Ed25519-Key des Reviewers
3. Plaintext-Begründungen dürfen **nie** in der `justification` Spalte gespeichert werden
4. Beim Löschen einer Reviewer-Response muss auch der zugehörige `encrypted_records` Eintrag gelöscht werden (CASCADE)
5. Obwohl alle Reviewer-Responses denselben Process-Key verwenden, sind sie durch Zugriffskontrolle auf Anwendungsebene isoliert
   - Jeder Reviewer kann nur seine eigenen verschlüsselten Records entschlüsseln und lesen
   - Die Signatur mit dem individuellen Reviewer-Key stellt Authentizität sicher

## Erweiterungen (Zukünftig)

### Mehrfach-Reviews

- Erlauben mehrerer Reviewer pro Assessment
- Aggregation von Reviewer-Bewertungen
- Konfliktlösung bei unterschiedlichen Bewertungen

### Review-Historie

- Versionierung von Reviewer-Antworten
- Nachverfolgung von Änderungen

### Benachrichtigungen

- E-Mail an Benutzer wenn Review abgeschlossen
- Benachrichtigung bei Status-Änderung

### Statistiken

- Durchschnittliche Review-Zeit
- Abweichungsanalyse (User vs. Reviewer)
- Reviewer-Performance-Metriken
