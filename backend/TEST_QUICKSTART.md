# Integration Tests - Quickstart

## Schnellstart

### 1. Docker starten
Stelle sicher, dass Docker Desktop läuft.

### 2. Dependencies installieren
```bash
go mod download
```

### 3. Tests ausführen
```bash
# Alle Tests
go test ./internal/handlers/... -v
```

## Beispiel-Output

```
=== RUN   TestGetActiveCatalogs
=== RUN   TestGetActiveCatalogs/Admin_can_get_active_catalogs
=== RUN   TestGetActiveCatalogs/Reviewer_can_get_active_catalogs  
=== RUN   TestGetActiveCatalogs/Regular_user_can_get_active_catalogs
--- PASS: TestGetActiveCatalogs (5.23s)
    --- PASS: TestGetActiveCatalogs/Admin_can_get_active_catalogs (0.05s)
    --- PASS: TestGetActiveCatalogs/Reviewer_can_get_active_catalogs (0.03s)
    --- PASS: TestGetActiveCatalogs/Regular_user_can_get_active_catalogs (0.03s)
```

## Häufige Kommandos

```bash
# Einzelnen Test ausführen
go test ./internal/handlers -run TestGetActiveCatalogs -v

# Mit Coverage
go test ./internal/handlers/... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Bestimmte Tests
go test ./internal/handlers/... -run TestCreate -v

# Docker-Status prüfen
docker ps
```

## Troubleshooting

**Problem**: Tests hängen
- Lösung: Docker neustarten

**Problem**: "Cannot connect to Docker daemon"
- Lösung: Docker Desktop starten

**Problem**: "Port already in use"
- Lösung: Andere Container stoppen oder Tests erneut ausführen (Ports werden automatisch gewählt)

## Weitere Informationen

Siehe [TESTING.md](internal/handlers/TESTING.md) für Details.
