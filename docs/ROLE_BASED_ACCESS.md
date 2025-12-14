# Role-Based Access Control

## Übersicht

Das System verwendet ein **unabhängiges** dreistufiges Rollensystem **ohne Hierarchie**:

- **admin** - Voller Zugriff auf Systemverwaltung (User, Rollen, Audit-Logs, Sessions)
- **reviewer** - Zugriff auf Review-Funktionen
- **user** - Zugriff auf Self-Assessment und Kataloge

**Wichtig:** Die Rollen sind **unabhängig** und **nicht hierarchisch**. Ein Admin hat nicht automatisch die Rechte eines Reviewers oder Users. Jede Rolle muss explizit zugewiesen werden.

## Rollensystem

### Keine Rollenhierarchie

Im Gegensatz zu vielen anderen Systemen gibt es **keine Vererbung** zwischen Rollen:

- Ein **Admin** ohne User-Rolle kann **keine** Self-Assessments durchführen
- Ein **Admin** ohne Reviewer-Rolle kann **keine** Reviews durchführen  
- Ein **User** hat **keine** Admin- oder Reviewer-Funktionen

Jede Funktion erfordert die entsprechende Rolle explizit.

## Zugriffskontrolle

### User ohne Rolle

User ohne zugewiesene Rolle haben nur Zugriff auf:

- `/api/v1/users/profile` - Eigenes Profil abrufen
- `/api/v1/users/profile/update` - Eigenes Profil aktualisieren
- `/api/v1/users/password/change` - Eigenes Passwort ändern
- `/api/v1/users/resend-verification` - Verifizierungsmail erneut senden
- `/api/v1/users/sessions/*` - Eigene Sessions verwalten

**Frontend-Verhalten:**

- Sieht nur den Link "Profil" in der Navigation
- Homepage zeigt Warnmeldung mit Hinweis, einen Admin zu kontaktieren

### User-Rolle

User mit der Rolle `user` haben Zugriff auf:

- **Katalog-Funktionen** (`/api/v1/catalogs/*`) - **Exklusiv für user-Rolle**
- **Self-Assessment-Funktionen** (`/api/v1/self-assessments/*`) - **Exklusiv für user-Rolle**
- Eigene Self-Assessments erstellen, bearbeiten und einreichen
- **Eigene Self-Assessments schließen** (nur im Status "draft", vor dem Einreichen)
- Kataloge einsehen und verwenden
- **Eigenes Profil** (wie User ohne Rolle)

**Frontend-Verhalten:**

- Navigation sichtbar mit allen User-Funktionen (Home, Kataloge, Self-Assessment, Einstufungen)
- Kann Self-Assessments durchführen und Kataloge einsehen
- Kann eigene Self-Assessments schließen, bevor sie eingereicht wurden
- **Kein** Zugriff auf Admin- oder Reviewer-Bereiche

**Wichtig:** Self-Assessments und Kataloge sind **ausschließlich** der `user`-Rolle vorbehalten. Admins und Reviewer ohne `user`-Rolle können weder Self-Assessments durchführen noch Kataloge einsehen.

### Reviewer-Rolle

Reviewer haben Zugriff auf:

- Review-Funktionen (spezifische Review-Endpunkte)
- Kann Assessments anderer User einsehen und bewerten
- **Eigenes Profil**

**Frontend-Verhalten:**

- Navigation sichtbar (sofern auch andere Rollen vorhanden)
- Reviewer-spezifische Menüpunkte
- **Kein** automatischer Zugriff auf User- oder Admin-Funktionen

**Hinweis:** Wenn ein Reviewer auch Self-Assessments durchführen oder Kataloge einsehen möchte, benötigt er **zusätzlich** die `user`-Rolle. Self-Assessments und Kataloge sind ausschließlich der `user`-Rolle vorbehalten.

### Admin-Rolle

Admins haben vollen Zugriff auf:

- User-Verwaltung (`/api/v1/admin/users/*`)
- Rollen-Verwaltung
- Katalog-Verwaltung (`/api/v1/admin/catalogs/*`)
- **Self-Assessment-Verwaltung** (`/api/v1/admin/self-assessments/*`)
  - Alle Self-Assessments einsehen (mit Filtern)
  - Self-Assessments löschen
  - Self-Assessments schließen
  - **Self-Assessments wieder öffnen** (innerhalb 24h nach Schließung)
- Audit-Logs (`/api/v1/admin/audit-logs/*`)
- Session-Verwaltung (`/api/v1/admin/sessions/*`)
- **Eigenes Profil**

**Frontend-Verhalten:**

- Admin-Bereich vollständig sichtbar
- Kann Self-Assessments anderer User einsehen und verwalten
- Kann geschlossene Self-Assessments innerhalb 24h wieder öffnen
- **Kann keine eigenen Self-Assessments durchführen** (benötigt `user`-Rolle)
- **Kann keine Kataloge einsehen** (benötigt `user`-Rolle)

**Wichtig:** Admins können Self-Assessments **verwalten** (schließen, wieder öffnen, löschen), aber nicht **selbst durchführen** oder **für andere submitten**. Für eigene Self-Assessments wird zusätzlich die `user`-Rolle benötigt.

### Mehrfach-Rollen

Ein User kann mehrere Rollen gleichzeitig haben:

**Beispiel-Kombinationen:**

- `admin` alleine - Kann System verwalten und Self-Assessments anderer einsehen/löschen, aber keine eigenen erstellen
- `user` alleine - Kann Self-Assessments und Kataloge nutzen, aber keine Admin- oder Review-Funktionen
- `admin` + `user` - Admin, der auch eigene Self-Assessments durchführt und Kataloge einsieht
- `admin` + `reviewer` - Admin, der auch Reviews durchführt  
- `admin` + `reviewer` + `user` - Admin mit vollem Zugriff auf alle Funktionen
- `reviewer` + `user` - Reviewer, der auch Self-Assessments durchführt und Kataloge einsieht

## OAuth Default-Rolle

### Konfiguration

Bei OAuth-Login kann automatisch eine Default-Rolle vergeben werden, falls der User keine Gruppe hat, die auf eine Rolle gemappt ist:

```bash
# Für jeden OAuth-Provider separat
OAUTH_KEYCLOAK_DEFAULT_ROLE=user
OAUTH_AZURE_DEFAULT_ROLE=user
```

### Verhalten

1. User meldet sich via OAuth an
2. System prüft OAuth-Gruppen gegen Gruppenmapping
3. Wenn keine Gruppe auf eine Rolle gemappt ist:
   - Falls `DEFAULT_ROLE` konfiguriert ist: Rolle wird zugewiesen
   - Sonst: User hat keine Rolle und sieht nur sein Profil
4. Änderungen werden im Audit-Log protokolliert

### Beispiel-Konfiguration

```bash
# Gruppenmapping (primär)
OAUTH_KEYCLOAK_GROUP_MAPPING=app-admin:admin,app-reviewer:reviewer,app-user:user
OAUTH_KEYCLOAK_GROUPS_CLAIM=groups

# Fallback für User ohne Gruppe
OAUTH_KEYCLOAK_DEFAULT_ROLE=user
```

**Szenario 1:** User ist in Gruppe "app-user"

- Ergebnis: Erhält `user`-Rolle durch Gruppenmapping

**Szenario 2:** User ist in keiner Gruppe

- Ergebnis: Erhält `user`-Rolle durch DEFAULT_ROLE

**Szenario 3:** User ist in keiner Gruppe, DEFAULT_ROLE nicht konfiguriert

- Ergebnis: Keine Rolle, nur Profilzugriff

## Frontend-Implementierung

### Navigation (MainLayout.tsx)

Die Navigation basiert auf den User-Rollen:

```typescript
const hasUserRole = user?.roles?.some(role => role.name === 'user');
const hasAnyRole = user?.roles && user.roles.length > 0;

// Navigation nur für User mit 'user'-Rolle oder höher
{hasUserRole && (
  <NavLink
    component={Link}
    to="/"
    label="Home"
    leftSection={<IconHome size="1rem" />}
  />
  // ... weitere Nav-Links
)}

// Profil-Link für alle authentifizierten User
<NavLink
  component={Link}
  to="/profile"
  label="Profil"
  leftSection={<IconUser size="1rem" />}
/>
```

### Homepage (HomePage.tsx)

Zeigt Warnmeldung für User ohne Rolle:

```typescript
{!hasAnyRole && (
  <Alert
    icon={<IconAlertCircle size="1rem" />}
    title="Keine Rolle zugewiesen"
    color="yellow"
  >
    Sie haben aktuell keine Rolle zugewiesen. Bitte kontaktieren Sie einen
    Administrator, um die 'user'-Rolle zu erhalten und auf die Self-Assessment-Funktionen
    zuzugreifen.
  </Alert>
)}
```

## Backend-Implementierung

### RBAC Middleware

Die Middleware prüft auf **exakte Rollenübereinstimmung** ohne Hierarchie:

```go
// RequireRole checks if the user has the exact required role
func (m *RBACMiddleware) RequireRole(roleName string) func(http.Handler) http.Handler {
 // ...
 // Check if user has the exact required role
 hasRole := false
 for _, role := range roles {
  if role.Name == roleName {
   hasRole = true
   break
  }
 }
 // ...
}
```

### Route-Konfiguration

**Nur eine spezifische Rolle:**

```go
// Nur für Admins
mux.Handle("/api/v1/admin/users/list",
    authMw.Authenticate(
        rbacMw.RequireRole("admin")(
            http.HandlerFunc(userHandler.ListUsers),
        ),
    ),
)
```

**Mehrere Rollen akzeptieren:**

```go (z.B. für gemeinsame Ressourcen)
mux.Handle("GET /api/v1/some-resource",
    authMw.Authenticate(
        rbacMw.RequireAnyRole("admin", "reviewer", "user")(
            http.HandlerFunc(handler.GetResource),
        ),
    ),
)
```

**Nur user-Rolle:**

```go
// Exklusiv für user (Self-Assessments, Kataloge)
mux.Handle("GET /api/v1/catalogs",
    authMw.Authenticate(
        rbacMw.RequireRole(
        rbacMw.RequireAnyRole("admin", "reviewer", "user")(
            http.HandlerFunc(catalogHandler.GetAllCatalogs),
        ),
    ),
)
```

**Nur authentifiziert (keine Rolle erforderlich):**

```go
mux.Handle("/api/v1/users/profile", 
    authMw.Authenticate(http.HandlerFunc(userHandler.GetProfile)))
```

## Admin-Schutz

Das System verhindert das Entfernen der letzten Admin-Rolle:

1. Beim Sync von OAuth-Gruppen wird geprüft, ob mindestens ein Admin verbleibt
2. Die Methode `CanRemoveAdminRole()` prüft vor dem Entfernen einer Admin-Rolle
3. Falls es der letzte Admin ist, wird die Rolle nicht entfernt

## Best Practices

### Für Entwickler

1. **Neue Routen immer schützen:** Verwende entweder `authMw.Authenticate()` oder zusätzlich `rbacMw.RequireRole()` / `rbacMw.RequireAnyRole()`
2. **Keine Hierarchie:** `RequireRole("user")` erlaubt **nur** User, **nicht** Admin oder Reviewer
3. **RequireAnyRole verwenden:** Für Endpunkte, die mehrere Rollen akzeptieren sollen
4. **Spezifisch sein:** Überlege genau, welche Rollen Zugriff haben sollen
5. **Admin-Endpunkte:** Immer mit `RequireRole("admin")` schützen

### Für Administratoren

1. **Mehrfach-Rollen vergeben:** Admins, die auch Self-Assessments durchführen sollen, benötigen zusätzlich die `user`-Rolle
2. **Default-Rolle setzen:** Konfiguriere `OAUTH_X_DEFAULT_ROLE=user` für OAuth-Provider
3. **Gruppenmapping:** Mappe OAuth-Gruppen auf die entsprechenden Rollen
4. **Audit-Logs prüfen:** Alle Rollenänderungen werden protokolliert
5. **Letzten Admin schützen:** System verhindert automatisch Entfernung des letzten Admins

### Für User

1. **Keine Rolle:** Kontaktiere einen Administrator zur Rollenvergabe
2. **Fehlende Funktionen:** Wenn du Self-Assessments durchführen möchtest, benötigst du die `user`-Rolle
3. **Mehrere Funktionen:** Für Admin + Self-Assessment benötigst du beide Rollen: `admin` **und** `user`
4. **Rollenänderung:** Änderungen werden beim nächsten OAuth-Login synchronisiert
Rolle, nur `admin`-Rolle. Kataloge sind exklusiv für `user`-Rolle.

**Lösung:**
Admin benötigt zusätzlich die `user`-Rolle, um auf Kataloge und Self-Assessments zuzugreifen

**Beispiel:** Admin möchte Self-Assessments durchführen, hat aber nur `admin`-Rolle.

**Lösung:**

1. Gruppenmapping erweitern, um mehrere Rollen zu vergeben
2. Oder: Manuell über Admin-Interface zusätzlich `user`-Rolle zuweisen

### Problem: Admin kann Kataloge nicht sehen

**Ursache:** Admin hat keine `user`-Rolle, nur `admin`-Rolle. Kataloge sind exklusiv für `user`-Rolle.

**Lösung:**
Admin benötigt zusätzlich die `user`-Rolle, um Kataloge einzusehen.

```sql
-- Admin zusätzlich user-Rolle geben
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u
CROSS JOIN roles r
WHERE u.email = 'admin@example.com'
AND r.name = 'user'
AND NOT EXISTS (
    SELECT 1 FROM user_roles ur
    WHERE ur.user_id = u.id AND ur.role_id = r.id
);
```

**Hinweis:** Admins können Self-Assessments anderer User **verwalten** (einsehen, löschen, schließen) ohne `user`-Rolle, aber keine eigenen erstellen.

### Problem: Admin kann sich nicht mehr einloggen

**Ursache:** Letzte Admin-Rolle wurde entfernt (sollte durch System verhindert werden)

**Lösung:**

1. Datenbankzugriff verwenden
2. Direkt Admin-Rolle in `user_roles`-Tabelle eintragen
3. Oder neuen Admin-User über Migration/Seed erstellen

### Problem: User sieht Navigation nicht

**Prüfen:**

1. Hat der User die `user`-Rolle? (Profil → Rollen prüfen)
2. Frontend aktualisieren (Hard Refresh: Cmd+Shift+R / Ctrl+Shift+R)
3. Token ist möglicherweise veraltet → Neu einloggen

### Problem: "Insufficient permissions" trotz richtiger Rolle

**Prüfen:**

1. Hat der User **exakt** die erforderliche Rolle? (Keine Hierarchie!)
2. Route verwendet `RequireRole()` oder `RequireAnyRole()` korrekt?
3. Mehrere Rollen erforderlich? User braucht **alle** benötigten Rollen
4. Audit-Logs prüfen auf Rollenänderungen
5. Session-Token aktualisieren (neu einloggen)

## Migration

hierarchischem zu flachem System

Falls Sie von einem System mit Rollenhierarchie migrieren:

1. **Backend:** RBAC-Middleware jetzt ohne Hierarchie
2. **Rollen prüfen:** Admins/Reviewer brauchen evtl. zusätzlich `user`-Rolle
3. **Gruppenmapping:** Kann mehrere Rollen gleichzeitig mappen
4. **Testen:** Alle Rollenkombinationen durchprüfen

```sql
-- Allen Admins und Reviewern zusätzlich die 'user'-Rolle geben
INSERT INTO user_roles (user_id, role_id)
SELECT DISTINCT ur.user_id, r_user.id
FROM user_roles ur
JOIN roles r ON ur.role_id = r.id
CROSS JOIN roles r_user
WHERE r.name IN ('admin', 'reviewer')
AND r_user.name = 'user'
AND NOT EXISTS (
    SELECT 1 FROM user_roles ur2
    WHERE ur2.user_id = ur.user_id AND ur2.role_id = r_use
    WHERE ur.user_id = u.id AND ur.role_id = r.id
);
```

## Siehe auch

- [OAuth Configuration](OAUTH_CONFIGURATION.md)
- [OAuth Group Mapping](OAUTH_GROUP_MAPPING.md)
- [Session Management](SESSION_MANAGEMENT.md)
