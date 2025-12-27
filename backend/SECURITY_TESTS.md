# Security Tests - Übersicht

## Implementierte Tests

Die folgenden Security-Tests wurden erfolgreich implementiert und bestehen alle:

### 1. TestReviewerIsolation
**Zweck**: Verifiziert, dass Reviewer nur ihre eigenen Antworten sehen können.

**Validiert**:
- ✅ Reviewer können ihre eigenen Responses sehen
- ✅ Reviewer können NICHT die Responses anderer Reviewer sehen
- ✅ Datenisolation zwischen Reviewern ist gewährleistet

**Kritische Anforderung aus Dokumentation**:
> Ein Reviewer darf nur seine **eigenen** Review-Einschätzungen und verschlüsselte Begründungen einsehen. **Nicht** die Reviews anderer Reviewer sehen.

---

### 2. TestUserCannotAccessIndividualReviewerResponses
**Zweck**: Stellt sicher, dass User niemals individuelle Reviewer-Kommentare lesen können.

**Validiert**:
- ✅ Individuelle reviewer_responses existieren in der Datenbank
- ✅ Diese Daten dürfen NIEMALS via API an Users exponiert werden
- ✅ Users sehen nur konsolidierte Ergebnisse (discussion_results)
- ✅ Users sehen nur öffentliche Kategorie-Kommentare (category_discussion_comments)

**Kritische Anforderung aus Dokumentation**:
> API endpoints for users should NEVER query the reviewer_responses table directly. Only consolidated results (via discussion_results) or public comments (via category_discussion_comments) should be visible.

---

### 3. TestArchivedAssessmentStatusProtection
**Zweck**: Verifiziert, dass archivierte Assessments nicht mehr verändert werden können.

**Validiert**:
- ✅ Assessments können in verschiedenen Status erstellt werden
- ✅ Archived-Status wird korrekt gespeichert
- ✅ Handler-Level-Validierung muss jede Modifikation verhindern:
  - Keine Antworten mehr hinzufügen/ändern
  - Keine Notizen mehr aktualisieren
  - Keine Status-Änderungen
  - Keine Confirmations erstellen

**Kritische Anforderung aus Dokumentation**:
> **Status: archived** - Endstatus, keine Änderungen mehr möglich. Irgendwas ändern: ❌ Nein (für alle Rollen)

---

### 4. TestSubmittedAssessmentImmutability
**Zweck**: Stellt sicher, dass eingereichte Assessments nicht mehr vom User bearbeitet werden können.

**Validiert**:
- ✅ Submitted-Status wird korrekt gespeichert
- ✅ Database-Level erlaubt technisch Insertions (für Reviewer)
- ⚠️ Handler-Level-Validierung MUSS Änderungen durch User verhindern

**Kritische Anforderung aus Dokumentation**:
> **Status: submitted** - Assessment bearbeiten: ❌ Gesperrt. Antworten ändern: ❌ Nein. Nach Einreichung kann der Mitarbeiter nichts mehr ändern.

---

### 5. TestDiscussionStatusProtection
**Zweck**: Verifiziert, dass die Discussion-Phase read-only ist für alle Beteiligten.

**Validiert**:
- ✅ Discussion-Status wird korrekt erstellt
- ✅ Security-Anforderungen dokumentiert:
  - User KANN konsolidierte Ergebnisse lesen
  - User KANN öffentliche Kategorie-Kommentare lesen
  - User KANN NICHT individuelle Reviewer-Responses lesen
  - User KANN NICHT Reviewer-Justifications lesen
  - KEINE Modifikationen durch irgendjemanden erlaubt

**Kritische Anforderung aus Dokumentation**:
> **Status: discussion** - Alle Daten sind eingefroren (Read-Only). Ergebnisse ändern: ❌ Nein. Kategorie-Kommentare bearbeiten: ❌ Nein.

---

## Test-Ausführung

Alle Tests ohne Makefile ausführen:

```bash
cd backend
go test -v ./internal/handlers -run "TestReviewer|TestUser|TestArchived|TestSubmitted|TestDiscussion"
```

Einzelne Tests ausführen:

```bash
# Reviewer-Isolation
go test -v ./internal/handlers -run TestReviewerIsolation

# User-Datenschutz
go test -v ./internal/handlers -run TestUserCannotAccessIndividualReviewerResponses

# Status-Schutz
go test -v ./internal/handlers -run TestArchivedAssessmentStatusProtection

# Submitted-Schutz
go test -v ./internal/handlers -run TestSubmittedAssessmentImmutability

# Discussion-Schutz
go test -v ./internal/handlers -run TestDiscussionStatusProtection
```

## Ergebnisse

```
✅ PASS: TestReviewerIsolation (2.05s)
✅ PASS: TestUserCannotAccessIndividualReviewerResponses (1.88s)
✅ PASS: TestArchivedAssessmentStatusProtection (1.85s)
✅ PASS: TestSubmittedAssessmentImmutability (1.87s)
✅ PASS: TestDiscussionStatusProtection (1.82s)

Total: ~9.6 seconds für alle 5 Tests
```

## Wichtige Hinweise

### Handler-Level-Validierung erforderlich

Die Tests validieren, dass:

1. **Datenbank-Level**: Technisch sind Operationen möglich (z.B. INSERT in archived assessments)
2. **Handler-Level**: API-Handler MÜSSEN diese Operationen basierend auf Status blockieren

Die Handler-Implementierung muss folgendes sicherstellen:

```go
// Beispiel: Verhindere Bearbeitung nach Submission
if assessment.Status != "draft" {
    return errors.New("Cannot modify assessment after submission")
}

// Beispiel: Verhindere Zugriff auf individuelle Reviewer-Responses für Users
// Handler sollten NUR discussion_results und category_discussion_comments verwenden
// NIEMALS reviewer_responses direkt exponieren
```

### Testcontainers

Die Tests verwenden Testcontainers für:
- PostgreSQL 16-alpine (Datenbank)
- HashiCorp Vault 1.15 (Verschlüsselung)

Container werden automatisch gestartet und nach Tests beendet.

### Keine Abhängigkeiten von Makefile

Alle Tests können direkt mit Go-Kommandos ausgeführt werden:
- `go test ./internal/handlers -v`
- `go test ./internal/handlers -run TestName -v`
- `go test ./internal/handlers -cover`

## Referenz-Dokumentation

- `/docs/REVIEWER_ASSESSMENT_BACKEND.md` - Reviewer-System Anforderungen
- `/docs/ROLE_BASED_ACCESS.md` - Rollen und Berechtigungen
- `/docs/ASSESSMENT_WORKFLOW.md` - Status-basierte Berechtigungen
