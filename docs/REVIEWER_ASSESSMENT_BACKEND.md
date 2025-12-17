# Reviewer Assessment System - Backend Requirements

## Überblick

Dieses Dokument beschreibt die Backend-Anforderungen für das Reviewer-Assessment-System, das es Reviewern ermöglicht, Selbsteinschätzungen von Benutzern zu bewerten.

## Rollentrennung & Datenschutz

### Strikte Rollentrennung

Das System implementiert eine **strikte Trennung** zwischen den Rollen:

- **User**: Zugriff auf eigene Selbsteinschätzungen und deren verschlüsselte Begründungen
- **Reviewer**: Zugriff auf Review-Prozess und eigene verschlüsselte Review-Begründungen
- **Admin**: System- und Nutzerverwaltung - **KEIN Zugriff** auf verschlüsselte Begründungen

**Kritisch**: Ein Admin-User ohne Reviewer- oder User-Rolle hat **weder** Zugriff auf `justifications` im Selbsteinschätzungs-Kontext **noch** auf `justifications` im Rahmen des Review-Prozesses.

### Datenisolation für Reviewer

Ein Reviewer darf:
- ✅ Nur seine **eigenen** Review-Einschätzungen und verschlüsselte Begründungen einsehen
- ✅ Die Angaben des Users zu Pfad/Zielstufe einsehen (aber **nicht** dessen Begründung)
- ❌ **Nicht** die Reviews anderer Reviewer sehen
- ❌ **Nicht** die Begründung des Users sehen
- ❌ **Nicht** eigene Assessments reviewen (Self-Review-Prevention)

### Implementierungsdetails

- Alle Reviewer-Endpunkte verwenden `RequireRole("reviewer")` (nicht `RequireAnyRole("reviewer", "admin")`)
- Keine Admin-Override-Funktionalität im Service-Layer
- Self-Review-Prevention für alle Benutzer (keine Ausnahmen)

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

**Authentifizierung:** JWT (Rolle: reviewer)

**Rollentrennung:** Nur Reviewer haben Zugriff. Admins **ohne** Reviewer-Rolle dürfen **keine** Begründungen einsehen.

**Datenisolation:** Reviewer sehen **nur ihre eigenen** Review-Antworten. Antworten anderer Reviewer sind niemals sichtbar.

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

**Authentifizierung:** JWT (Rolle: reviewer)

**Rollentrennung:** Nur Reviewer haben Zugriff. Admins **ohne** Reviewer-Rolle dürfen **keine** Begründungen erstellen.

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
- `403`: Keine Berechtigung (nicht reviewer) ODER Versuch eigenes Assessment zu prüfen
- `404`: Assessment nicht gefunden

### 3. Reviewer-Antwort löschen

**Endpoint:** `DELETE /api/v1/review/assessment/:id/responses/:category_id`

**Beschreibung:** Löscht die Reviewer-Bewertung für eine Kategorie

**Authentifizierung:** JWT (Rolle: reviewer)

**Rollentrennung:** Nur Reviewer haben Zugriff.

**Response:**

```json
{
  "message": "Reviewer response deleted successfully"
}
```

### 4. Review abschließen

**Endpoint:** `POST /api/v1/review/assessment/:id/complete`

**Beschreibung:** Markiert das Review als abgeschlossen und ändert den Assessment-Status

**Authentifizierung:** JWT (Rolle: reviewer)

**Rollentrennung:** Nur Reviewer können Reviews abschließen und Status ändern.

**Validierung:**

- Alle Kategorien müssen eine Reviewer-Antwort haben
- Alle erforderlichen Begründungen müssen vorhanden sein (min. 50 Zeichen bei Abweichung)

**Request Body:**

```json
{
  "new_status": "review_consolidation"  // oder "reviewed", "discussion"
}
```

**Hinweis:** Der Status `review_consolidation` kann nur gesetzt werden, wenn mindestens 3 vollständige Reviews vorliegen (siehe Abschnitt "Review Consolidation" weiter unten).

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
    // GetResponsesByAssessmentID lädt nur Responses des spezifischen Reviewers
    GetResponsesByAssessmentID(assessmentID, reviewerUserID uint) ([]ReviewerResponse, error)
    // GetResponseByCategoryID lädt nur Response des spezifischen Reviewers
    GetResponseByCategoryID(assessmentID, categoryID, reviewerUserID uint) (*ReviewerResponse, error)
    DeleteResponse(assessmentID, categoryID, reviewerUserID uint) error
    // GetResponsesWithUserComparison vergleicht User-Antworten mit den Antworten des aktuellen Reviewers
    GetResponsesWithUserComparison(assessmentID, reviewerUserID uint) ([]ResponseComparison, error)
}
```

### Handler Layer (`reviewer_handler.go`)

#### Berechtigungsprüfung

- **Nur Benutzer mit Rolle "reviewer"** dürfen Reviews durchführen (nicht "admin")
- **Jeder Reviewer kann nur seine eigenen Reviews bearbeiten** (keine Admin-Override-Funktion)
- **Self-Review-Prevention**: Reviewer können nicht ihre eigenen Assessments reviewen

## Migration

### Migration File: `016_reviewer_responses.up.sql`

```sql
CREATE TABLE reviewer_responses (
    id SERIAL PRIMARY KEY,
    assessment_id INTEGER NOT NULL REFERENCES self_assessments(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    reviewer_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    path_id INTEGER NOT NULL REFERENCES paths(id) ON DELETE CASCADE,
    level_id INTEGER NOT NULL REFERENCES levels(id) ON DELETE CASCADE,
    encrypted_justification_id BIGINT REFERENCES encrypted_records(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(assessment_id, category_id, reviewer_user_id)
);

CREATE INDEX idx_reviewer_responses_assessment ON reviewer_responses(assessment_id);
CREATE INDEX idx_reviewer_responses_reviewer ON reviewer_responses(reviewer_user_id);
CREATE INDEX idx_reviewer_responses_encrypted_justification ON reviewer_responses(encrypted_justification_id);

COMMENT ON TABLE reviewer_responses IS 'Individual reviewer assessments for self-assessments. Each reviewer creates their own independent review.';
COMMENT ON COLUMN reviewer_responses.encrypted_justification_id IS 'Reference to encrypted justification in encrypted_records table';
COMMENT ON COLUMN reviewer_responses.path_id IS 'Reviewer-selected path, may differ from user selection';
COMMENT ON COLUMN reviewer_responses.level_id IS 'Reviewer-selected level, may differ from user selection';
```

### Migration File: `016_reviewer_responses.down.sql`

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
2. Reviewer können nur offene Assessments (status: submitted, in_review, review_consolidation, reviewed, discussion) bewerten
3. **✅ IMPLEMENTIERT** - Reviewer können nicht ihre eigenen Assessments bewerten (prüfen: reviewer_user_id != assessment.user_id)
   - Diese Prüfung ist in ALLEN Reviewer-Endpunkten implementiert:
     - ✅ `GET /api/v1/review/assessment/:id/responses`
     - ✅ `POST /api/v1/review/assessment/:id/responses`
     - ✅ `DELETE /api/v1/review/assessment/:id/responses/:category_id`
     - ✅ `POST /api/v1/review/assessment/:id/complete`
   - Fehlercode: `403 Forbidden` mit Meldung "Cannot review your own assessment"
4. **✅ IMPLEMENTIERT: Datenisolierung zwischen Reviewern**
   - Reviewer dürfen **nur ihre eigenen** Review-Antworten sehen und bearbeiten
   - Review-Antworten anderer Reviewer sind **niemals** zugänglich (auch nicht lesend)
   - Admins können alle Review-Antworten aller Reviewer einsehen
   - Implementierung: Alle GET/POST/DELETE Endpunkte filtern `reviewer_user_id = current_user.id` (außer für Admins)

### Datenintegrität

1. Reviewer-Antworten können nur für existierende Assessments und Kategorien erstellt werden
2. Level-ID muss zu einem gültigen Level im Katalog gehören
3. ✅ Bei Abweichung vom User-Level ist eine Begründung von mindestens 50 Zeichen erforderlich
4. ✅ Bei Abweichung vom User-Pfad (path_id) ist ebenfalls eine Begründung von mindestens 50 Zeichen erforderlich
5. ✅ Alle Begründungen werden verschlüsselt in `encrypted_records` gespeichert

### Verschlüsselungs-Sicherheit

1. Reviewer-Begründungen verwenden den Process-Key der zugehörigen Self-Assessment
2. Signatur erfolgt mit dem Ed25519-Key des Reviewers
3. Plaintext-Begründungen dürfen **nie** in der `justification` Spalte gespeichert werden
4. Beim Löschen einer Reviewer-Response muss auch der zugehörige `encrypted_records` Eintrag gelöscht werden (CASCADE)
5. Obwohl alle Reviewer-Responses denselben Process-Key verwenden, sind sie durch Zugriffskontrolle auf Anwendungsebene isoliert
   - Jeder Reviewer kann nur seine eigenen verschlüsselten Records entschlüsseln und lesen
   - Die Signatur mit dem individuellen Reviewer-Key stellt Authentizität sicher

## Erweiterungen (Zukünftig)

### Review Consolidation (Status-Übergang)

Der Status `review_consolidation` wird zwischen `in_review` und `reviewed` eingeführt:

**Zweck:**
- Mehrere Reviewer erstellen unabhängig voneinander ihre Einzel-Reviews
- Sobald mindestens 3 vollständige Reviews vorliegen, kann das Review-Team in den Status `review_consolidation` wechseln
- In diesem Status trifft sich das Team, um die einzelnen Reviews zu besprechen und zu einem gemeinsamen Ergebnis zusammenzuführen

**Anforderungen:**

1. **Minimum 3 vollständige Reviews:**
   - Ein Review gilt als vollständig, wenn für alle Kategorien des Assessments eine Reviewer-Response existiert
   - Der Übergang zu `review_consolidation` ist nur möglich, wenn >= 3 verschiedene Reviewer vollständige Reviews abgegeben haben
   - Backend-Validierung erforderlich: `COUNT(DISTINCT reviewer_user_id WHERE all_categories_reviewed) >= 3`

2. **API-Endpoint:**
   ```
   GET /api/v1/review/assessment/:id/completion-status
   ```
   **Authentifizierung:** JWT (Rolle: reviewer)
   
   **Rollentrennung:** Nur Reviewer haben Zugriff.
   
   **Response:**
   ```json
   {
     "total_reviewers": 5,
     "complete_reviews": 3,
     "can_consolidate": true,
     "reviewers_with_complete_reviews": [
       { "reviewer_id": 10, "reviewer_name": "Max Mustermann", "completed_at": "2025-12-16T10:00:00Z" },
       { "reviewer_id": 15, "reviewer_name": "Anna Schmidt", "completed_at": "2025-12-16T11:00:00Z" },
       { "reviewer_id": 20, "reviewer_name": "Tom Weber", "completed_at": "2025-12-16T12:00:00Z" }
     ]
   }
   ```

3. **Frontend-Anforderungen:**
   - Anzeige der Anzahl vorliegender vollständiger Reviews (z.B. Badge: "3/5 Reviews")
   - Button "In Konsolidierung überführen" wird erst ab 3 vollständigen Reviews aktiviert
   - Liste der Reviewer mit vollständigen Reviews anzeigen
   - Status-Badge mit Icon für `review_consolidation` (bereits implementiert: cyan, IconUsers)

4. **Status-Übergänge:**
   ```
   submitted → in_review (Review-Prozess startet)
   in_review → review_consolidation (mindestens 3 vollständige Reviews liegen vor)
   review_consolidation → reviewed (Konsolidierung abgeschlossen, Ergebnis steht fest)
   reviewed → discussion (Besprechung mit User kann beginnen)
   ```

5. **Datenbank-Schema:**
   - Neue Spalte: `review_consolidation_at TIMESTAMP` (in Migration 015 hinzugefügt)
   - Status CHECK constraint erweitert um 'review_consolidation' (Migration 015)

**TODO (Backend-Implementierung erforderlich):**
- [x] Tabelle `reviewer_responses` erstellt (Migration 016)
- [x] Repository Layer implementiert
- [x] Service Layer mit Verschlüsselung implementiert
- [x] Handler Layer mit allen Endpunkten implementiert
- [x] Routen registriert in main.go
- [ ] Endpoint `/api/v1/review/assessment/:id/completion-status` vollständig getestet
- [ ] Frontend-Komponente zur Anzeige der Review-Statistik und Konsolidierungs-Button

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
