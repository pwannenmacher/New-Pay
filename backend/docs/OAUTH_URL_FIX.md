# OAuth URL Fix und Erste User Registrierung

## Datum: 7. Dezember 2025

## Problembeschreibung

1. **Hardcodierte Frontend-URL**: Bei OAuth-Login Fehlern wurde immer auf `http://localhost:5173/login?error=...` weitergeleitet, auch wenn das Frontend auf Port 3001 läuft
2. **Erste Registrierung blockiert**: Wenn die Datenbank komplett leer war (keine User), konnte sich der erste User nicht registrieren, wenn `ENABLE_REGISTRATION=false` oder `ENABLE_OAUTH_REGISTRATION=false` gesetzt war

## Implementierte Lösung

### 1. Konfigurierbare Frontend-URL für OAuth-Fehler

**Neue Hilfsmethode in `auth_handler.go`:**

```go
func (h *AuthHandler) getBaseLoginURL() string {
    // Extrahiert die Base-URL aus der konfigurierten OAuth Frontend Callback URL
    // z.B. "http://localhost:3001/oauth/callback" → "http://localhost:3001"
    callbackURL := h.config.OAuth.FrontendCallbackURL
    
    parsedURL, err := url.Parse(callbackURL)
    if err != nil {
        log.Printf("Failed to parse frontend callback URL: %v, using default", err)
        return "http://localhost:3001"
    }
    
    return fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
}
```

**Alle OAuth-Fehler-Redirects verwenden jetzt:**

```go
redirectURL := fmt.Sprintf("%s/login?error=...", h.getBaseLoginURL())
http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
```

Dies gilt für folgende Fehler:
- `invalid_provider` - Provider ungültig oder nicht aktiviert
- `invalid_state` - State-Cookie fehlt oder stimmt nicht überein
- `no_code` - Kein Authorization Code vom Provider
- `token_exchange_failed` - Token-Austausch fehlgeschlagen
- `userinfo_failed` - User-Info konnte nicht abgerufen werden
- `no_email` - Keine E-Mail in User-Info gefunden
- `user_creation_failed` - User-Erstellung fehlgeschlagen
- `registration_disabled` - Registrierung deaktiviert

### 2. Erste User Registrierung immer erlauben

**Neue Methode im AuthService:**

```go
func (s *AuthService) CountAllUsers() (int, error) {
    return s.userRepo.CountAll()
}
```

**Geänderte Logik in `Register()` Handler:**

```go
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    // Check if registration is enabled (allow if no users exist)
    if !h.config.App.EnableRegistration {
        // Check if any users exist - allow registration if database is empty
        userCount, err := h.authService.CountAllUsers()
        if err != nil || userCount > 0 {
            respondWithError(w, http.StatusForbidden, "Registration is disabled")
            return
        }
    }
    // ... rest of registration logic
}
```

**Geänderte Logik in `OAuthCallback()` Handler:**

```go
// If it's a new user and OAuth registration is disabled, deny access (unless it's the first user)
if isNewUser && !h.config.App.EnableOAuthRegistration {
    // Check if any users exist - allow registration if database is empty
    userCount, err := h.authService.CountAllUsers()
    if err != nil || userCount > 1 { // userCount > 1 because the user was just created
        log.Printf("OAuth callback: registration disabled, rejecting new user %s", email)
        _ = h.auditMw.LogAction(nil, "user.oauth.registration.disabled", "users", 
            fmt.Sprintf("OAuth registration blocked for %s via %s (registration disabled)", 
            email, providerConfig.Name), getIP(r), r.UserAgent())
        redirectURL := fmt.Sprintf("%s/login?error=registration_disabled", h.getBaseLoginURL())
        http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
        return
    }
}
```

## Verhalten

### Szenario 1: Komplett leere Datenbank

**Fall A: Email/Password Registrierung**
- `ENABLE_REGISTRATION=false`
- Datenbank hat 0 User
- ✅ **Registrierung erlaubt** (erste User kann sich registrieren)
- Nach der Registrierung: `userCount = 1`
- Weitere Registrierungen: ❌ **Blockiert**

**Fall B: OAuth Registrierung**
- `ENABLE_OAUTH_REGISTRATION=false`
- Datenbank hat 0 User
- ✅ **OAuth-Login erlaubt** (erste User kann sich via OAuth registrieren)
- User wird erstellt: `userCount = 1`
- Prüfung: `userCount > 1` → false (1 ist nicht > 1)
- ✅ **OAuth-Login erfolgreich**
- Weitere OAuth-Registrierungen: ❌ **Blockiert**

### Szenario 2: Mindestens ein User existiert

**Fall A: Email/Password Registrierung**
- `ENABLE_REGISTRATION=false`
- Datenbank hat >= 1 User
- ❌ **Registrierung blockiert**

**Fall B: OAuth Registrierung**
- `ENABLE_OAUTH_REGISTRATION=false`
- Datenbank hat >= 1 User
- Neuer OAuth-User versucht sich zu registrieren
- User wird erstellt: `userCount = n+1`
- Prüfung: `userCount > 1` → true
- ❌ **OAuth-Login blockiert**, User wird gelöscht (implizit durch Rollback)
- Redirect: `http://localhost:3001/login?error=registration_disabled`

### Szenario 3: Registrierung aktiviert

**Fall A: Email/Password Registrierung**
- `ENABLE_REGISTRATION=true`
- ✅ **Registrierung immer erlaubt**, egal wie viele User existieren

**Fall B: OAuth Registrierung**
- `ENABLE_OAUTH_REGISTRATION=true`
- ✅ **OAuth-Login immer erlaubt**, egal wie viele User existieren

## Frontend URL Konfiguration

Die Frontend-URL wird aus der Umgebungsvariable `OAUTH_FRONTEND_CALLBACK_URL` extrahiert:

```bash
# In .env
OAUTH_FRONTEND_CALLBACK_URL=http://localhost:3001/oauth/callback
```

Die `getBaseLoginURL()` Methode extrahiert automatisch:
- Scheme: `http`
- Host: `localhost:3001`
- Ergebnis: `http://localhost:3001`

Dies wird dann für alle Fehler-Redirects verwendet:
- `http://localhost:3001/login?error=invalid_provider`
- `http://localhost:3001/login?error=registration_disabled`
- etc.

## Betroffene Dateien

### Backend

1. **`backend/internal/handlers/auth_handler.go`**
   - Neue Methode: `getBaseLoginURL()`
   - Geändert: `Register()` - erste User Registrierung erlauben
   - Geändert: `OAuthCallback()` - alle Redirects + erste OAuth User erlauben
   - Geändert: 11 OAuth-Fehler-Redirects verwenden jetzt `getBaseLoginURL()`

2. **`backend/internal/service/auth_service.go`**
   - Neue Methode: `CountAllUsers()` - gibt Anzahl aller User zurück

## Testing

### Test 1: Erste Registrierung mit leerem System

1. Datenbank leeren (oder frisches System)
2. `ENABLE_REGISTRATION=false` setzen
3. Registrierung über Frontend-Formular versuchen
4. ✅ Erste Registrierung sollte erfolgreich sein
5. Zweite Registrierung versuchen
6. ❌ Sollte mit "Registration is disabled" blockiert werden

### Test 2: Erste OAuth-Registrierung mit leerem System

1. Datenbank leeren
2. `ENABLE_OAUTH_REGISTRATION=false` setzen
3. OAuth-Login versuchen (z.B. mit Authentik)
4. ✅ Erste OAuth-Registrierung sollte erfolgreich sein
5. Zweite OAuth-Registrierung (anderer User) versuchen
6. ❌ Sollte mit `registration_disabled` Fehler blockiert werden
7. ✅ Redirect sollte auf `http://localhost:3001/login?error=registration_disabled` gehen (nicht mehr 5173!)

### Test 3: OAuth-Fehler Redirects

1. OAuth-Login mit ungültigem Provider versuchen
2. ✅ Sollte auf `http://localhost:3001/login?error=invalid_provider` redirecten
3. Andere OAuth-Fehler provozieren
4. ✅ Alle sollten auf Port 3001 redirecten, nicht mehr auf 5173

## Vorteile

1. **Flexibilität**: Frontend kann auf beliebigem Port/Domain laufen
2. **Konsistenz**: Eine zentrale Konfiguration (`OAUTH_FRONTEND_CALLBACK_URL`) steuert alle Redirects
3. **Sicherheit**: Erste User kann System initialisieren, danach ist Registrierung kontrolliert
4. **Usability**: Admins können System mit gesperrter Registrierung ausliefern, erster User kann sich trotzdem registrieren
5. **Docker-Kompatibilität**: Funktioniert sowohl in Docker (Port 3001) als auch lokal (Port 5173)

## Hinweise

- Die erste User Registrierung wird automatisch erkannt, keine manuelle Konfiguration nötig
- Der erste User wird standardmäßig zum Admin (existierende Logik bleibt unverändert)
- Die `getBaseLoginURL()` Methode hat einen Fallback auf `http://localhost:3001` falls das Parsing fehlschlägt
- OAuth-User werden erst nach der Registrierungsprüfung erstellt, daher `userCount > 1` statt `userCount > 0`
