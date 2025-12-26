# Self-Assessment Workflow und Berechtigungen

Dieses Dokument beschreibt die verschiedenen Status-Phasen eines Self-Assessments und die Berechtigungen der verschiedenen Rollen.

## Status-Übersicht

Ein Self-Assessment durchläuft folgende Status in dieser Reihenfolge:

```plain
draft → submitted → in_review → review_consolidation → reviewed → discussion → archived
                                                                                   ↓
                                    ← ← ← ← ← ← closed (kann innerhalb 24h zurückgesetzt werden)
```

## Status-Definitionen

| Status | Beschreibung | Dauer/Trigger |
| -------- | -------------- | --------------- |
| **draft** | Initiale Erstellung, Mitarbeiter füllt Selbsteinschätzung aus | Bis zur Einreichung |
| **submitted** | Mitarbeiter hat Selbsteinschätzung eingereicht | Bis Reviewer starten |
| **in_review** | Reviewer bewerten die Selbsteinschätzung | Bis 3+ Reviewer fertig sind |
| **review_consolidation** | Mindestens 3 Reviewer haben bewertet, Team konsolidiert Ergebnisse | Bis Konsolidierung abgeschlossen |
| **reviewed** | Alle Kategorien wurden genehmigt, finaler Kommentar und Freigabe steht aus | Bis alle Reviewer freigegeben haben |
| **discussion** | Ergebnis ist eingefroren und wird dem Mitarbeiter zur Besprechung angezeigt | Bis Besprechung abgeschlossen |
| **archived** | Besprechung abgeschlossen, Assessment archiviert | Endstatus |
| **closed** | Vorzeitig geschlossen (kann innerhalb 24h rückgängig gemacht werden) | Temporär oder permanent |

## Berechtigungen nach Status und Rolle

### Legende

- ✅ **Erlaubt**: Diese Aktion kann ausgeführt werden
- ❌ **Verboten**: Diese Aktion ist nicht erlaubt
- 🔒 **Read-Only**: Nur Lesezugriff
- ⏰ **Zeitbegrenzt**: Nur innerhalb eines bestimmten Zeitraums

---

## 1. Status: **draft**

| Aktion | User (Owner) | Reviewer | Admin |
| -------- | -------------- | ---------- | ------- |
| Assessment bearbeiten | ✅ Vollzugriff | ❌ Kein Zugriff | 🔒 Nur lesen |
| Antworten hinzufügen/ändern | ✅ Ja | ❌ Nein | ❌ Nein |
| Status ändern → submitted | ✅ Ja | ❌ Nein | ❌ Nein |
| Status ändern → closed | ✅ Ja | ❌ Nein | ✅ Ja |
| Assessment löschen | ❌ Nein | ❌ Nein | ❌ Nein |
| Assessment anzeigen | ✅ Ja | ❌ Nein | ✅ Ja |

**Hinweise:**

- Nur der Mitarbeiter (Owner) kann sein eigenes Assessment im Draft-Status bearbeiten
- Admins können das Assessment schließen, aber nicht für den User einreichen

---

## 2. Status: **submitted**

| Aktion | User (Owner) | Reviewer | Admin |
| -------- | -------------- | ---------- | ------- |
| Assessment bearbeiten | ❌ Gesperrt | ❌ Nein | ❌ Nein |
| Antworten ändern | ❌ Nein | ❌ Nein | ❌ Nein |
| Assessment anzeigen | 🔒 Read-only | 🔒 Vorbereitung | ✅ Ja |
| Status ändern → in_review | ❌ Nein | ✅ Ja | ❌ Nein |
| Status ändern → closed | ❌ Nein | ❌ Nein | ✅ Ja |
| Review starten | ❌ Nein | ✅ Ja | ❌ Nein |

**Hinweise:**

- Nach Einreichung kann der Mitarbeiter nichts mehr ändern
- Reviewer können das Assessment sehen und den Review-Prozess starten
- Nur Admins können das Assessment schließen

---

## 3. Status: **in_review**

| Aktion | User (Owner) | Reviewer | Admin |
| -------- | -------------- | ---------- | ------- |
| Assessment anzeigen | 🔒 Read-only | ✅ Ja | ✅ Ja |
| Eigene Review-Antworten erstellen | ❌ Nein | ✅ Ja | ❌ Nein |
| Eigene Review-Antworten bearbeiten | ❌ Nein | ✅ Ja (nur eigene) | ❌ Nein |
| Andere Reviews anzeigen | ❌ Nein | ❌ Nein | ❌ Nein |
| Status ändern → review_consolidation | ❌ Nein | ✅ Ja (wenn 3+ Reviews) | ❌ Nein |
| Status ändern → reviewed | ❌ Nein | ✅ Ja | ❌ Nein |
| Status ändern → closed | ❌ Nein | ❌ Nein | ✅ Ja |

**Hinweise:**

- Reviewer sehen nur ihre eigenen Antworten, nicht die anderer Reviewer
- Mindestens 3 vollständige Reviews werden für die Konsolidierung empfohlen
- Reviewer können direkt zu "reviewed" springen, wenn keine Konsolidierung nötig ist

---

## 4. Status: **review_consolidation**

| Aktion | User (Owner) | Reviewer | Admin |
| -------- | -------------- | ---------- | ------- |
| Assessment anzeigen | 🔒 Read-only | ✅ Ja | ✅ Ja |
| Alle Reviews anzeigen | ❌ Nein | ✅ Ja | ❌ Nein |
| Gemittelte Ergebnisse sehen | ❌ Nein | ✅ Ja | ❌ Nein |
| Override erstellen | ❌ Nein | ✅ Ja | ❌ Nein |
| Override bearbeiten | ❌ Nein | ✅ Ja (nur eigene) | ❌ Nein |
| Override/Averaged approven | ❌ Nein | ✅ Ja (nicht eigene) | ❌ Nein |
| Override-Approval zurücknehmen | ❌ Nein | ⏰ Ja (1h nach "reviewed") | ❌ Nein |
| Kategorie-Kommentare verfassen | ❌ Nein | ❌ Nein | ❌ Nein |
| Finalen Kommentar verfassen | ❌ Nein | ❌ Nein | ❌ Nein |
| Status ändern → in_review | ❌ Nein | ✅ Ja | ❌ Nein |
| Status ändern → reviewed | ❌ Nein | ✅ Ja (wenn alle genehmigt) | ❌ Nein |
| Status ändern → closed | ❌ Nein | ❌ Nein | ✅ Ja |

**Hinweise:**

- Jeder Override/Averaged Response benötigt 2 Approvals
- Reviewer können ihre eigenen Overrides nicht approven
- Status wechselt automatisch zu "reviewed", wenn alle Kategorien approved sind

---

## 5. Status: **reviewed**

| Aktion | User (Owner) | Reviewer | Admin |
| -------- | -------------- | ---------- | ------- |
| Assessment anzeigen | 🔒 Read-only | ✅ Ja | ✅ Ja |
| Konsolidierung anzeigen | ❌ Nein | ✅ Ja | ❌ Nein |
| **Kategorie-Kommentare verfassen** | ❌ Nein | ✅ Ja | ❌ Nein |
| **Kategorie-Kommentare bearbeiten** | ❌ Nein | ✅ Ja | ❌ Nein |
| Finalen Kommentar verfassen | ❌ Nein | ✅ Ja | ❌ Nein |
| Finalen Kommentar approven | ❌ Nein | ✅ Ja | ❌ Nein |
| Approval zurücknehmen | ❌ Nein | ⏰ Ja (1h) | ❌ Nein |
| Status ändern → discussion | ❌ Nein | ✅ Ja (wenn final approved) | ❌ Nein |
| Status ändern → closed | ❌ Nein | ❌ Nein | ✅ Ja |

**Hinweise:**

- **Wichtig**: Kategorie-Kommentare können im Status "reviewed" verfasst werden
- Diese Kommentare werden später in der Discussion-Ansicht dem Mitarbeiter angezeigt
- Finaler Kommentar benötigt Approval von allen Reviewern, die am Review beteiligt waren
- Approvals können innerhalb 1 Stunde zurückgenommen werden

---

## 6. Status: **discussion**

| Aktion | User (Owner) | Reviewer | Admin |
| -------- | -------------- | ---------- | ------- |
| Discussion Result anzeigen | ✅ Ja | ✅ Ja | ✅ Ja |
| Kategorie-Ergebnisse sehen | ✅ Ja | ✅ Ja | ✅ Ja |
| **Kategorie-Kommentare lesen** | ✅ Ja | ✅ Ja | ✅ Ja |
| Kategorie-Kommentare bearbeiten | ❌ Nein | ❌ Nein | ❌ Nein |
| Gewichtete Gesamtbewertung sehen | ✅ Ja | ✅ Ja | ✅ Ja |
| Finalen Kommentar lesen | ✅ Ja | ✅ Ja | ✅ Ja |
| Konsolidierungs-Details sehen | ❌ Nein | ✅ Ja | ❌ Nein |
| Ergebnisse ändern | ❌ Nein | ❌ Nein | ❌ Nein |
| Status ändern → archived | ❌ Nein | ✅ Ja | ❌ Nein |
| Status ändern → closed | ❌ Nein | ❌ Nein | ✅ Ja |

**Hinweise:**

- Alle Daten sind eingefroren (Read-Only)
- **Kategorie-Kommentare**: Mitarbeiter sieht nun die öffentlichen Erklärungen pro Kategorie
- Discussion Result wird beim ersten Status-Wechsel zu "discussion" erstellt
- Mitarbeiter kann seine ursprüngliche Selbsteinschätzung mit dem Review-Ergebnis vergleichen

---

## 7. Status: **archived**

| Aktion | User (Owner) | Reviewer | Admin |
| -------- | -------------- | ---------- | ------- |
| Assessment anzeigen | ✅ Read-only | ✅ Read-only | ✅ Read-only |
| Discussion Result anzeigen | ✅ Ja | ✅ Ja | ✅ Ja |
| Kategorie-Kommentare lesen | ✅ Ja | ✅ Ja | ✅ Ja |
| Irgendwas ändern | ❌ Nein | ❌ Nein | ❌ Nein |
| Status ändern | ❌ Nein | ❌ Nein | ❌ Nein |

**Hinweise:**

- Endstatus, keine Änderungen mehr möglich
- Dient als historische Aufzeichnung

---

## 8. Status: **closed**

| Aktion | User (Owner) | Reviewer | Admin |
| -------- | -------------- | ---------- | ------- |
| Assessment anzeigen | ✅ Ja | ✅ Ja | ✅ Ja |
| Status wiederherstellen | ❌ Nein | ❌ Nein | ⏰ Ja (24h) |
| Assessment löschen | ❌ Nein | ❌ Nein | ✅ Ja (nur wenn nie submitted) |

**Hinweise:**

- Admin kann Assessment innerhalb 24h nach Schließung zum vorherigen Status zurücksetzen
- Assessment kann nur gelöscht werden, wenn es nie eingereicht wurde (submitted_at = NULL)
- Nach 24h ist der Closed-Status permanent

---

## Status-Übergänge Matrix

| Von / Nach | draft | submitted | in_review | review_consolidation | reviewed | discussion | archived | closed |
| ------------ | ------- | ----------- | ----------- | --------------------- | ---------- | ------------ | ---------- | -------- |
| **draft** | - | ✅ Owner | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ Owner/Admin |
| **submitted** | ❌ | - | ✅ Reviewer | ❌ | ❌ | ❌ | ❌ | ✅ Admin |
| **in_review** | ❌ | ❌ | - | ✅ Reviewer | ✅ Reviewer | ❌ | ❌ | ✅ Admin |
| **review_consolidation** | ❌ | ❌ | ✅ Reviewer | - | ✅ Reviewer | ❌ | ❌ | ✅ Admin |
| **reviewed** | ❌ | ❌ | ❌ | ❌ | - | ✅ Reviewer | ❌ | ✅ Admin |
| **discussion** | ❌ | ❌ | ❌ | ❌ | ❌ | - | ✅ Reviewer | ✅ Admin |
| **archived** | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | - | ❌ |
| **closed** | ⏰ Admin (24h) | ⏰ Admin (24h) | ⏰ Admin (24h) | ⏰ Admin (24h) | ⏰ Admin (24h) | ⏰ Admin (24h) | ❌ | - |

---

## Wichtige Workflows

### 1. Normaler Review-Workflow

```plain
draft (Mitarbeiter) 
  → submitted (Mitarbeiter) 
  → in_review (Reviewer) 
  → review_consolidation (Reviewer, 3+ Reviews) 
  → reviewed (Reviewer, alle approvals) 
  → discussion (Reviewer, final approved) 
  → archived (Reviewer)
```

### 2. Schneller Review-Workflow (ohne Konsolidierung)

```plain
draft → submitted → in_review → reviewed → discussion → archived
```

### 3. Abbruch-Workflow

```plain
Jeder Status → closed (Admin oder Owner bei draft)
  → innerhalb 24h zurück zum vorherigen Status (Admin)
```

---

## Spezielle Berechtigungen

### Kategorie-Kommentare (Neu implementiert)

- **Wann verfassbar**: Im Status "reviewed" (Abschluss-Tab)
- **Wer kann verfassen**: Nur Reviewer
- **Zweck**: Öffentliche Erklärung der Bewertung pro Kategorie für den Mitarbeiter
- **Wann sichtbar**: Für Mitarbeiter ab Status "discussion"
- **Unterschied**: Kategorie-Kommentare sind öffentlich, Review-Begründungen (Justifications) sind intern

### Konsolidierungs-Approvals

- **Override-Approvals**: Benötigt 2 Approvals von Reviewern (nicht vom Ersteller)
- **Averaged-Approvals**: Benötigt 2 Approvals von beliebigen Reviewern
- **Final-Approval**: Benötigt Approval von allen Reviewern, die am Review beteiligt waren
- **Rücknahme**: Innerhalb 1 Stunde nach Status-Wechsel zu "reviewed" möglich

### Admin-Sonderrechte

- Kann Assessments in jedem Status schließen
- Kann geschlossene Assessments innerhalb 24h wiederherstellen
- Kann nie eingereichte Assessments löschen (submitted_at = NULL)
- Kann KEINE Status-Übergänge im Review-Prozess durchführen (nur Reviewer)
- Kann NICHT für andere Users einreichen oder reviewen

---

## Zeitliche Einschränkungen

| Aktion | Zeitlimit | Rolle |
| -------- | ----------- | ------- |
| Override/Averaged Approval zurücknehmen | 1 Stunde nach "reviewed" | Reviewer |
| Final Approval zurücknehmen | 1 Stunde nach "reviewed" | Reviewer |
| Closed Status rückgängig machen | 24 Stunden nach "closed" | Admin |

---

## Datenschutz und Sichtbarkeit

| Daten | User (Owner) | Reviewer (eigene) | Reviewer (andere) | Admin |
| ------- | -------------- | ------------------- | ------------------- | ------- |
| Eigene Antworten (draft-discussion) | ✅ Vollzugriff | 🔒 Read-only | 🔒 Read-only | 🔒 Read-only |
| Review-Antworten (in_review) | ❌ Nicht sichtbar | ✅ Nur eigene | ❌ Nicht sichtbar | ❌ Nicht sichtbar |
| Alle Reviews (consolidation) | ❌ Nicht sichtbar | ✅ Alle sichtbar | ✅ Alle sichtbar | ❌ Nicht sichtbar |
| Review-Begründungen (intern) | ❌ Nie sichtbar | ✅ Ja | ✅ Ja | ❌ Nicht sichtbar |
| **Kategorie-Kommentare** (öffentlich) | ✅ Ab "discussion" | ✅ Immer | ✅ Immer | ✅ Ja |
| Discussion Result | ✅ Ab "discussion" | ✅ Immer | ✅ Immer | ✅ Ja |

---

## API-Endpunkte und Berechtigungen

Siehe [REVIEWER_ASSESSMENT_BACKEND.md](./REVIEWER_ASSESSMENT_BACKEND.md) für eine detaillierte Übersicht aller API-Endpunkte.

### Wichtige Endpunkte nach Status

**Status: in_review**

- `POST /api/v1/review/responses` - Reviewer erstellen Antworten
- `GET /api/v1/self-assessments/:id` - Reviewer + Owner können Assessment sehen

**Status: review_consolidation & reviewed**

- `GET /api/v1/review/consolidation/:id` - Konsolidierungsdaten abrufen
- `POST /api/v1/review/consolidation/:id/override` - Override erstellen
- `POST /api/v1/review/consolidation/:id/override/:categoryId/approve` - Override approven
- `POST /api/v1/review/consolidation/:id/category/:categoryId/comment` - **Kategorie-Kommentar erstellen**
- `POST /api/v1/review/consolidation/:id/final` - Finalen Kommentar speichern

**Status: discussion**

- `GET /api/v1/discussion/:id` - Discussion Result abrufen (Owner + Reviewer)

---

## Änderungshistorie

- **26.12.2025**: Dokumentation erstellt
- **26.12.2025**: Kategorie-Kommentare (category_discussion_comments) hinzugefügt - werden im Status "reviewed" verfasst und sind ab "discussion" für Mitarbeiter sichtbar
