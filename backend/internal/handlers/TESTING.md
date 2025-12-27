# Integration Tests

Dieses Verzeichnis enthält Integration Tests für die Backend-API mit Testcontainers.

## Überblick

Die Integration Tests verwenden:

- **Testcontainers** für PostgreSQL und HashiCorp Vault
- **Test Fixtures** für konsistente Testdaten
- **JWT Auth Helper** für Authentifizierung in Tests

## Struktur

```plain
internal/
├── testutil/                    # Test-Hilfsfunktionen
│   ├── testutil.go             # Container-Setup (PostgreSQL + Vault)
│   ├── fixtures.go             # Testdaten-Erstellung
│   └── auth.go                 # JWT-Token-Generierung
└── handlers/
    └── security_test.go        # Security-Tests (Reviewer-Isolation, Status-Schutz)
```

## Verwendung

### Voraussetzungen

- Docker muss laufen (für Testcontainers)
- Go 1.25+

### Dependencies installieren

```bash
cd backend
go mod download
```

### Tests ausführen

Alle Tests ausführen:

```bash
cd backend
go test ./internal/handlers -v
```

Einzelnen Test ausführen:

```bash
go test ./internal/handlers -run TestReviewerIsolation -v
```

Alle Security-Tests ausführen:

```bash
go test ./internal/handlers -run "TestReviewer|TestUser|TestArchived|TestSubmitted|TestDiscussion" -v
```

Mit Race Detector:

```bash
go test ./internal/handlers -race -v
```

Mit Coverage:

```bContainer Setup

Jeder Test verwendet Testcontainers, die automatisch:

1. PostgreSQL Container startet (postgres:16-alpine)
2. Vault Container startet (hashicorp/vault:1.15)
3. Datenbankmigrationen ausführt
4. Testdaten (Fixtures) erstellt

```go
func TestExample(t *testing.T) {
    containers := testutil.SetupTestContainers(t)
    defer containers.Cleanup(t)
    
    fixtures := testutil.SetupFixtures(t, containers.DB
4. Testdaten (Fixtures) erstellt
5. Services und Handler initialisiert

```go
func TestExample(t *testing.T) {
    suite := setupTestSuite(t)
    defer suite.teardownTestSuite(t)
    
    // Test-Code hier
}
```

### Fixtures

Vordefinierte Testdaten:

- `AdminUser` - Benutzer mit admin, reviewer, user Rollen
- `ReviewerUser` - Benutzer mit reviewer, user Rollen
- `RegularUser` - Benutzer nur mit user Rolle
- `Catalog` - Aktiver Test-Katalog
- `Categories` - 3 Kategorien mit Gewichtungen
- `Paths` - 2 Entwicklungspfade
- `Levels` - 4 Level (Beginner bis Expert)

Zusätzliche Testdaten erstellen:

// Benutzer erstellen
user := &models.User{
    Email:     "test@example.com",
    FirstName: "Test",
    LastName:  "User",
}
err := containers.DB.QueryRow(`
    INSERT INTO users (email, password_hash, first_name, last_name, email_verified)
    VALUES ($1, $2, $3, $4, true)
    RETURNING id, created_at, updated_at
authHelper := testutil.NewAuthHelper()

// Token generieren
token, err := authHelper.GenerateToken(userID, email, []string{"user", "reviewer"})

// Oder direkt zu Request hinzufügen
authHelper.AddAuthHeader(t, req, user, []string{"user", "reviewer"})
```

## Getestete Funktionen

### Security-Tests (security_test.go)

#### ✅ TestReviewerIsolation
Verifiziert, dass Reviewer nur ihre eigenen Antworten sehen können.

**Validiert:**
- Reviewer können ihre eigenen Responses sehen
- Reviewer können NICHT die Responses anderer Reviewer sehen
- Datenisolation zwischen Reviewern ist gewährleistet

#### ✅ TestUserCannotAccessIndividualReviewerResponses
Stellt sicher, dass User niemals individuelle Reviewer-Kommentare lesen können.

**Validiert:**
- Individuelle reviewer_responses existieren in der Datenbank
- Diese Daten dürfen NIEMALS via API an Users exponiert werden
- Users sehen nur konsolidierte Ergebnisse (discussion_results)
- Users sehen nur öffentliche Kategorie-Kommentare

#### ✅ TestArchivedAssessmentStatusProtection
Verifiziert, dass archivierte Assessments nicht mehr verändert werden können.

**Validiert:**
- Assessments können in verschiedenen Status erstellt werden
- Archived-Status wird korrekt gespeichert
- Handler-Level-Validierung muss jede Modifikation verhindern
Security-Anforderungen

Die Tests basieren auf den Anforderungen aus der Dokumentation:

### Reviewer-Isolation
**Quelle:** `/docs/REVIEWER_ASSESSMENT_BACKEND.md`

> Ein Reviewer darf nur seine **eigenen** Review-Antworten sehen. **Nicht** die Reviews anderer Reviewer.

### User-Datenschutz
**Quelle:** `/docs/REVIEWER_ASSESSMENT_BACKEND.md`

> API endpoints for users should NEVER query the reviewer_responses table directly. Only consolidated results (via discussion_results) or public comments (via category_discussion_comments) should be visible.

### Status-basierte Berechtigungen
**Quelle:** `/docs/ASSESSMENT_WORKFLOW.md`

- **draft**: User kann bearbeiten
- **submitted**: Keine Änderungen mehr durch User
- **in_review**: Nur Reviewer können Reviews erstellen
- **discussion**: Read-only für alle
- **archived**: Endstatus, keine Änderungen `PUT /api/v1/self-assessments/{id}/status` - Status ändern
- ✅ `GET /api/v1/admin/self-assessments` - Admin: Alle Assessments
- ✅ `GET /api/v1/self-assessments/{id}/responses` - Antworten abrufen
- ✅ `GET /api/v1/self-assessments/{id}/completeness` - Vollständigkeit prüfen

### Review-Endpunkte (self_assessment_review_test.go)

- ✅ `GET /api/v1/review/open-assessments` - Offene Assessments für Review
- ✅ `GET /api/v1/review/completed-assessments` - Abgeschlossene Assessments
- ✅ `DELETE /api/v1/admin/self-assessments/{id}` - Assessment löschen (Admin)

## Rollenbasierte Tests

Jeder Endpunkt wird mit verschiedenen Rollen getestet:

1. **Admin** - Vollzugriff auf alle Funktionen
2. **Reviewer** - Zugriff auf Review-Funktionen
3. **User** - Zugriff nur auf eigene Daten
4. **Unauthenticated** - Kein Zugriff (401)

Beispiel:

```go
tests := []struct {
    name           string
    user           *models.User
    roles          []string
    expectedStatus int
}{
    {
        name:           "Admin can access",
        user:           suite.fixtures.AdminUser,
        roles:          []string{"admin"},
        expectedStatus: http.StatusOK,
    },
    {
        name:           "User cannot access",
        user:           suite.fixtures.RegularUser,
        roles:          []string{"user"},
        expectedStatus: http.StatusForbidden,
    },
}
```

## Best Practices
containers.Cleanup(t)` um Container zu beenden:

```go
func TestExample(t *testing.T) {
    containers := testutil.SetupTestContainers(t)
    defer containers.Cleanup(t)
    
    // Test-Code
}
```
### Testdaten isolieren

Jeder Test sollte seine eigenen Testdaten erstellen, um Abhängigkeiten zu vermeiden.

### Cleanup

Verwende immer `defer suite.teardownTestSuite(t)` um Container zu beenden.

### Parallele Tests

Tests können parallel laufen, da jede Suite eigene Container verwendet:
r Test eigene Container verwendet:

```go
func TestParallel(t *testing.T) {
    t.Parallel()
    containers := testutil.SetupTestContainers(t)
    defer containers.Cleanup(t)
    
    fixtures := testutil.SetupFixtures(t, containers.DB)
    // Test-Code
```

### Test-Namen

Verwende beschreibende Namen: im `handlers` Verzeichnis
2. Testcontainers und Fixtures verwenden
3. Tests für verschiedene Szenarien schreiben

```go
package handlers_test

import (
    "testing"
    "new-pay/internal/testutil"
)

func TestNewHandler(t *testing.T) {
    containers := testutil.SetupTestContainers(t)
    defer containers.Cleanup(t)
    
    fixtures := testutil.SetupFixtures(t, containers.DB

1. Neue Test-Datei erstellen: `*_test.go`
2. `setupTestSuite` wiederverwenden
3. Tests mit verschiedenen Rollen schreiben

```go
package handlers_test

func TestNewHandler(t *testing.T) {
    suite := setupTestSuite(t)
    defer suite.teardownTestSuite(t)
    
    // Tests hier
}
```

### Neue Fixtures hinzufügen

In `testutil/fixtures.go`:

```go
func (f *Fixtures) CreateCustomData(t *testing.T, params ...) *Model {
    t.Helper()
    
    var model Model
    err := f.DB.QueryRow(`INSERT INTO ...`).Scan(...)
    if err != nil {
        t.Fatalf("Failed to create: %v", err)
    }
    
    return &model
}
```

## Troubleshooting

### Container starten nicht

- Stelle sicher, dass Docker läuft
- Prüfe Docker-Logs: `docker logs <container-id>`
- Erhöhe Timeout in `testutil.go`

### Tests hängen

- Überprüfe auf Deadlocks in Datenbank-Transaktionen
- Stelle sicher, dass alle Connections geschlossen werden

### Migrations-Fehle2-5s (pro Test)
- Einzelner Test: ~1-3s (inkl. Container-Setup)
- Alle 5 Security-Tests: ~10s

Container werden nach Tests automatisch beendet und aufgeräumt.

## Weitere Dokumentation

- [SECURITY_TESTS.md](/Users/paul/frachtwerk/new-pay-gh/backend/SECURITY_TESTS.md) - Detaillierte Security-Test-Dokumentation
- [TEST_QUICKSTART.md](/Users/paul/frachtwerk/new-pay-gh/backend/TEST_QUICKSTART.md) - Schnellstart-Guide
- [/docs/REVIEWER_ASSESSMENT_BACKEND.md](/Users/paul/frachtwerk/new-pay-gh/docs/REVIEWER_ASSESSMENT_BACKEND.md) - Reviewer-System Anforderungen
- [/docs/ASSESSMENT_WORKFLOW.md](/Users/paul/frachtwerk/new-pay-gh/docs/ASSESSMENT_WORKFLOW.md) - Status-basierte Berechtigungen
## Performance

Typische Testlaufzeiten:

- Container-Setup: ~5-10s (einmalig pro Suite)
- Einzelner Test: ~50-200ms
- Gesamte Test-Suite: ~30-60s

Container werden nach Tests automatisch beendet und aufgeräumt.
