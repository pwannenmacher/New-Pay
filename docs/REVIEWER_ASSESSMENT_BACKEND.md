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
    level_id INTEGER NOT NULL REFERENCES levels(id) ON DELETE CASCADE,
    justification TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(assessment_id, category_id, reviewer_user_id)
);

CREATE INDEX idx_reviewer_responses_assessment ON reviewer_responses(assessment_id);
CREATE INDEX idx_reviewer_responses_reviewer ON reviewer_responses(reviewer_user_id);
```

### Felder
- `id`: Primärschlüssel
- `assessment_id`: Referenz zur Selbsteinschätzung
- `category_id`: Referenz zur bewerteten Kategorie
- `reviewer_user_id`: Benutzer-ID des Reviewers
- `level_id`: Vom Reviewer gewähltes Level
- `justification`: Begründung des Reviewers (erforderlich wenn abweichend vom User-Level)
- `created_at`: Erstellungszeitpunkt
- `updated_at`: Zeitpunkt der letzten Änderung

## API-Endpunkte

### 1. Reviewer-Antworten abrufen
**Endpoint:** `GET /api/v1/review/assessment/:id/responses`

**Beschreibung:** Lädt alle Reviewer-Antworten für eine Selbsteinschätzung

**Authentifizierung:** JWT (Rolle: reviewer oder admin)

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
  "level_id": 3,
  "justification": "Begründung..."
}
```

**Validierung:**
- `category_id`: Erforderlich, muss zur Assessment gehören
- `level_id`: Erforderlich, muss gültiges Level sein
- `justification`: 
  - Optional wenn `level_id` == User-Level
  - Erforderlich (min. 50 Zeichen) wenn `level_id` != User-Level

**Response:**
```json
{
  "id": 1,
  "assessment_id": 123,
  "category_id": 5,
  "reviewer_user_id": 10,
  "level_id": 3,
  "justification": "...",
  "created_at": "2025-12-15T10:30:00Z",
  "updated_at": "2025-12-15T10:30:00Z"
}
```

**Fehler:**
- `400`: Validierungsfehler (z.B. Begründung zu kurz bei Abweichung)
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
    GetResponsesByAssessmentID(assessmentID uint) ([]ReviewerResponse, error)
    GetResponseByCategoryID(assessmentID, categoryID uint) (*ReviewerResponse, error)
    DeleteResponse(assessmentID, categoryID, reviewerUserID uint) error
    GetResponsesWithUserComparison(assessmentID uint) ([]ResponseComparison, error)
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
    level_id INTEGER NOT NULL REFERENCES levels(id) ON DELETE CASCADE,
    justification TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(assessment_id, category_id, reviewer_user_id)
);

CREATE INDEX idx_reviewer_responses_assessment ON reviewer_responses(assessment_id);
CREATE INDEX idx_reviewer_responses_reviewer ON reviewer_responses(reviewer_user_id);
```

### Migration File: `015_reviewer_responses.down.sql`
```sql
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
3. Reviewer können nicht ihre eigenen Assessments bewerten (prüfen: reviewer_user_id != assessment.user_id)

### Datenintegrität
1. Reviewer-Antworten können nur für existierende Assessments und Kategorien erstellt werden
2. Level-ID muss zu einem gültigen Level im Katalog gehören
3. Bei Abweichung vom User-Level ist eine Begründung von mindestens 50 Zeichen erforderlich

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
