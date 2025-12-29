# Logging System

## Übersicht

Das Backend verwendet ein strukturiertes Logging-System basierend auf `log/slog` mit einer zentralen Middleware, die automatisch alle HTTP-Requests auf verschiedenen Log-Levels erfasst.

## Log-Levels und Verhalten

### INFO-Level (Standard)

Jeder Request wird mit folgenden Informationen geloggt:

**Beim Eingang des Requests:**

- `remote_ip`: IP-Adresse des Clients
- `user_agent`: Browser/Client User-Agent
- `method`: HTTP-Methode (GET, POST, PUT, DELETE, etc.)
- `path`: Aufgerufener Pfad

**Nach Abschluss des Requests:**

- Alle obigen Felder plus:
- `status`: HTTP-Status-Code
- `duration_ms`: Dauer der Request-Verarbeitung in Millisekunden

**Beispiel:**

```json
{"time":"2025-12-24 10:15:30","level":"INFO","msg":"Incoming request","remote_ip":"127.0.0.1:54321","user_agent":"Mozilla/5.0","method":"GET","path":"/api/v1/users/profile"}
{"time":"2025-12-24 10:15:30","level":"INFO","msg":"Request completed","remote_ip":"127.0.0.1:54321","user_agent":"Mozilla/5.0","method":"GET","path":"/api/v1/users/profile","status":200,"duration_ms":45}
```

### DEBUG-Level

Bei aktivem DEBUG-Level werden zusätzliche Details geloggt:

**Beim Eingang (ersetzt den INFO-Log):**

- Alle INFO-Felder plus:
- `query_params`: Alle Query-Parameter als Key-Value-Map
- `request_body`: Vollständiger Request-Body (falls vorhanden)

**Nach Abschluss:**

- Alle INFO-Felder plus:
- `response_body`: Vollständiger Response-Body

**Wichtig:** Bei DEBUG-Level wird der initiale INFO-Log durch einen ausführlichen DEBUG-Log ersetzt, um Duplikate zu vermeiden.

**Beispiel:**

```json
{"time":"2025-12-24 10:15:30","level":"DEBUG","msg":"Incoming request","remote_ip":"127.0.0.1:54321","user_agent":"Mozilla/5.0","method":"POST","path":"/api/v1/auth/login","query_params":{},"request_body":"{\"email\":\"user@example.com\",\"password\":\"***\"}"}
{"time":"2025-12-24 10:15:30","level":"INFO","msg":"Request completed","remote_ip":"127.0.0.1:54321","user_agent":"Mozilla/5.0","method":"POST","path":"/api/v1/auth/login","status":200,"duration_ms":123,"response_body":"{\"access_token\":\"...\"}"}
```

### WARN-Level

Nur fehlgeschlagene Requests mit HTTP-Status 4xx werden geloggt:

- Client-Fehler (Bad Request, Unauthorized, Forbidden, Not Found, etc.)
- Enthält alle Standard-Felder (remote_ip, method, path, status, duration_ms)

**Beispiel:**

```json
{"time":"2025-12-24 10:15:30","level":"WARN","msg":"Request failed","remote_ip":"127.0.0.1:54321","user_agent":"curl/7.64.1","method":"POST","path":"/api/v1/auth/login","status":401,"duration_ms":12}
```

### ERROR-Level

Nur Serverfehler mit HTTP-Status 5xx werden geloggt:

- Internal Server Error, Bad Gateway, Service Unavailable, etc.
- Enthält alle Standard-Felder

**Beispiel:**

```json
{"time":"2025-12-24 10:15:30","level":"ERROR","msg":"Request failed with error","remote_ip":"127.0.0.1:54321","user_agent":"curl/7.64.1","method":"GET","path":"/api/v1/data","status":500,"duration_ms":234}
```

## Konfiguration

Das Log-Level wird über die Umgebungsvariable `LOG_LEVEL` konfiguriert:

```bash
LOG_LEVEL=INFO    # Standard (INFO, WARN, ERROR)
LOG_LEVEL=DEBUG   # Alle Logs inkl. Request/Response-Bodies
LOG_LEVEL=WARN    # Nur Warnungen und Fehler
LOG_LEVEL=ERROR   # Nur Fehler
```

Die Konfiguration erfolgt in [`internal/logger/logger.go`](../backend/internal/logger/logger.go).

## Implementierung

### Middleware

Die zentrale Logging-Middleware befindet sich in [`internal/middleware/logging.go`](../backend/internal/middleware/logging.go).

**Funktionsweise:**

1. Request-Body wird gepuffert für DEBUG-Logging
2. Response-Writer wird gewrapped, um Status-Code und Response-Body zu erfassen
3. Initial-Log beim Request-Eingang (INFO oder DEBUG)
4. Handler wird ausgeführt
5. Completion-Log nach Abschluss (Level abhängig vom Status-Code)

**Middleware-Reihenfolge in `cmd/api/main.go`:**

```go
handler := middleware.LoggingMiddleware(
    middleware.SecurityHeaders(
        corsMw.Handler(
            rateLimiter.Limit(mux),
        ),
    ),
)
```

### Handler-Logging

**Grundregel:** Handler sollten `http.Error()` oder die Helper-Funktionen `respondWithError()` / `respondWithJSON()` verwenden, um Fehler an das Frontend zu senden. Die Middleware loggt diese automatisch.

**Business-Event-Logs:** Wichtige Business-Events (z.B. "User registered successfully", "User logged in successfully") werden weiterhin in den Handlern geloggt, da sie zusätzlichen fachlichen Kontext bieten.

**Beispiel aus `auth_handler.go`:**

```go
user, err := h.authService.Register(req.Email, req.Password, req.FirstName, req.LastName)
if err != nil {
    slog.Error("Registration failed", "email", req.Email, "error", err)
    respondWithError(w, http.StatusBadRequest, err.Error())
    return
}

slog.Info("User registered successfully", "user_id", user.ID, "email", user.Email)
```

## Best Practices

### 1. Fehlerbehandlung

✅ **Richtig:**

```go
if err != nil {
    slog.Error("Failed to process request", "error", err, "user_id", userID)
    http.Error(w, "Internal server error", http.StatusInternalServerError)
    return
}
```

❌ **Falsch:** Fehler nicht loggen (Frontend bekommt keine Details):

```go
if err != nil {
    slog.Error("Failed to process request", "error", err)
    // http.Error fehlt - Frontend erhält keine Fehlermeldung!
    return
}
```

### 2. Sensitive Daten

**Achtung:** Bei DEBUG-Level werden Request- und Response-Bodies vollständig geloggt!

Sensitive Daten wie Passwörter, Tokens, etc. werden im Klartext in den Logs erscheinen. DEBUG-Level sollte daher nur in Entwicklungsumgebungen oder für gezieltes Debugging verwendet werden.

### 3. Redundante Logs vermeiden

Die Middleware loggt bereits:

- Alle eingehenden Requests (INFO/DEBUG)
- Alle abgeschlossenen Requests (INFO/WARN/ERROR)
- Request/Response-Bodies (DEBUG)

**Vermeiden:** Redundante "Handler called" oder "Request received" Logs in den Handlern.

**Behalten:** Business-Events und Fehler mit zusätzlichem Kontext.

## Log-Analyse

### Erfolgreiche Requests filtern

```bash
# Nur erfolgreiche Requests (Status 2xx)
grep '"level":"INFO"' logs.json | grep '"status":2'
```

### Fehler analysieren

```bash
# Alle Fehler (4xx und 5xx)
grep -E '"level":"(WARN|ERROR)"' logs.json

# Nur Serverfehler (5xx)
grep '"level":"ERROR"' logs.json
```

### Performance-Analyse

```bash
# Langsame Requests (> 1000ms)
grep '"duration_ms"' logs.json | awk '$NF > 1000'
```

### DEBUG-Logging für spezifische Requests

```bash
# Backend mit DEBUG-Level starten
LOG_LEVEL=DEBUG go run .

# Oder in Docker:
docker-compose exec backend sh -c "LOG_LEVEL=DEBUG ./api"
```

## Troubleshooting

### Zu viele Logs

**Problem:** Log-Dateien werden zu groß.

**Lösung:** Log-Level auf WARN oder ERROR erhöhen in Production:

```bash
LOG_LEVEL=WARN
```

### Fehlende Request-Details

**Problem:** Request-Bodies werden nicht geloggt.

**Lösung:** DEBUG-Level aktivieren:

```bash
LOG_LEVEL=DEBUG
```

### Doppelte Logs

**Problem:** Requests werden mehrfach geloggt.

**Lösung:** Dies sollte nicht mehr auftreten. Bei DEBUG-Level ersetzt der ausführliche DEBUG-Log den INFO-Log. Falls doch Duplikate auftreten, prüfen ob Log-Level korrekt gesetzt ist.

## Audit-Logging

Zusätzlich zum Request-Logging gibt es ein separates Audit-System für sicherheitsrelevante Aktionen:

- Login/Logout
- Benutzer-Verwaltung
- Rollen-Zuweisungen
- OAuth-Authentifizierung

Siehe [`internal/middleware/audit.go`](../backend/internal/middleware/audit.go) für Details.

## Weiterführende Dokumentation

- [Go Best Practices](GO_BEST_PRACTICES.md)
- [Session Management](SESSION_MANAGEMENT.md)
- [OAuth Configuration](OAUTH_CONFIGURATION.md)
