# Kriterienkatalog-System

## Überblick

Das Kriterienkatalog-System verwaltet Bewertungskriterien für Selbsteinschätzungen. Kataloge durchlaufen verschiedene Phasen und haben Gültigkeitszeiträume.

## Katalog-Phasen

Ein Katalog durchläuft folgende Phasen:

- **draft**: Entwurf, nur von Admins editierbar
- **active**: Aktiv und für Selbsteinschätzungen nutzbar
- **archived**: Archiviert, nicht mehr für neue Selbsteinschätzungen nutzbar

### Phasen-Übergänge

Erlaubte Übergänge:

- `draft` → `active`: Katalog aktivieren (nur wenn vollständig)
- `active` → `archived`: Katalog archivieren
- `active` → `draft`: Zurück zu Entwurf (nur wenn keine Selbsteinschätzungen existieren)

Archivierte Kataloge können nicht mehr geändert werden.

## Gültigkeitszeitraum

Jeder Katalog hat:
- `valid_from`: Startdatum der Gültigkeit
- `valid_until`: Enddatum der Gültigkeit

Admins können `valid_until` für aktive Kataloge verkürzen (z.B. bei vorzeitiger Ablösung).

## Rollenbasierte Sichtbarkeit

### GET /api/v1/catalogs

Die API liefert katalogspezifische Daten basierend auf der Benutzerrolle:

#### Regular User (keine Admin/Reviewer-Rolle)

Sieht folgende Kataloge:
- Alle Kataloge in Phase `active`
- Archivierte Kataloge (`archived`), bei denen der User eine Selbsteinschätzung abgegeben hat

**Implementierung**: 
- Backend: `CatalogService.GetVisibleCatalogs()` filtert basierend auf `userID`
- Repository: `SelfAssessmentRepository.GetCatalogIDsByUserID()` liefert Katalog-IDs mit User-Beteiligung

#### Reviewer

Sieht folgende Kataloge:
- Alle Kataloge in Phase `active`
- Alle Kataloge in Phase `archived`
- Alle Kataloge in Phase `draft` (Einblick in zukünftige Kataloge)

#### Administrator

Sieht alle Kataloge unabhängig von Phase oder Gültigkeitszeitraum.

## Berechtigungen

### Kataloge anzeigen

| Rolle | draft | active | archived |
|-------|-------|--------|----------|
| User | ❌ | ✅ | ✅ (nur mit eigener Selbsteinschätzung) |
| Reviewer | ✅ | ✅ | ✅ |
| Admin | ✅ | ✅ | ✅ |

### Kataloge bearbeiten

| Aktion | draft | active | archived |
|--------|-------|--------|----------|
| Basisdaten ändern | Admin | ❌ | ❌ |
| `valid_until` verkürzen | ❌ | Admin | ❌ |
| Struktur ändern (Kategorien, Level, Pfade) | Admin | ❌ | ❌ |
| Phase ändern | Admin | Admin | ❌ |
| Löschen | Admin | ❌ | ❌ |

## Navigation

### Frontend-Struktur

Die Katalog-Navigation ist zweigeteilt:

**Allgemeine Navigation** (`/catalogs`):
- Für alle authentifizierten User sichtbar
- Zeigt rollenbasiert gefilterte Katalogliste
- Nur Ansicht, keine Verwaltungsfunktionen

**Admin-Navigation** (`/admin/catalogs`):
- Nur für Admins sichtbar
- Vollständige Katalogverwaltung (CRUD)
- Phasen-Übergänge, Strukturbearbeitung

## Code-Referenzen

### Backend

- `internal/service/catalog_service.go`: Geschäftslogik für Kataloge
  - `GetVisibleCatalogs(userRoles, userID)`: Rollenbasierte Filterung
  - `TransitionToActive()`, `TransitionToArchived()`: Phasen-Übergänge
- `internal/repository/catalog_repository.go`: Datenbankoperationen
- `internal/repository/self_assessment_repository.go`: 
  - `GetCatalogIDsByUserID()`: Katalog-IDs mit User-Beteiligung
- `internal/handlers/catalog_handler.go`: HTTP-Handler

### Frontend

- `pages/CatalogsPage.tsx`: User-facing Katalogübersicht
- `pages/admin/CatalogManagementPage.tsx`: Admin-Katalogverwaltung
- `pages/admin/CatalogEditorPage.tsx`: Katalog-Editor
- `components/layout/MainLayout.tsx`: Navigation mit Rollen-Checks

## Validierung

### Katalog-Vollständigkeit

Ein Katalog kann nur von `draft` zu `active` übergehen, wenn:
- Mindestens eine Kategorie existiert
- Mindestens ein Level existiert
- Jede Kategorie hat mindestens einen Pfad (Level-Zuordnung)

### Überlappungsprüfung

Beim Erstellen/Bearbeiten wird geprüft, dass sich Gültigkeitszeiträume nicht-archivierter Kataloge nicht überschneiden.

## Beispiel-Workflow

1. Admin erstellt neuen Katalog (Phase: `draft`)
2. Admin fügt Kategorien, Level und Pfade hinzu
3. Admin aktiviert Katalog (Phase: `active`)
4. User erstellen Selbsteinschätzungen basierend auf diesem Katalog
5. Admin archiviert Katalog bei Ablauf (Phase: `archived`)
6. User können ihren archivierten Katalog weiterhin einsehen
