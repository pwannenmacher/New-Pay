# Go Backend Best Practices

## JSON Response Handling

### ⚠️ WICHTIG: Nil Slices Problem

**Problem:**  
In Go werden `nil` Slices als JSON `null` encodiert, nicht als leere Arrays `[]`. Dies führt zu Fehlern im Frontend, da TypeScript/JavaScript Arrays erwartet.

**Beispiel des Problems:**

```go
var items []string  // nil slice
json.NewEncoder(w).Encode(items)  // ❌ Ergibt: null
```

**Lösung:**  
Verwende immer den `JSONResponse` Helper aus `cmd/api/json_helpers.go`:

```go
// ✅ RICHTIG - Verwendet JSONResponse Helper
func GetItems(w http.ResponseWriter, r *http.Request) {
    items, err := repo.GetItems()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    JSONResponse(w, items)  // nil slices werden automatisch zu []
}

// ❌ FALSCH - Direktes JSON Encoding
func GetItems(w http.ResponseWriter, r *http.Request) {
    items, err := repo.GetItems()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(items)  // nil wird zu null!
}
```

**Alternative Lösungen:**

1. **Slice Initialisierung im Repository:**

```go
func (r *Repository) GetItems() ([]Item, error) {
    items := []Item{}  // Leeres Array statt nil
    rows, err := r.db.Query("SELECT * FROM items")
    // ...
    return items, nil
}
```

2. **Check im Handler:**

```go
items, err := repo.GetItems()
if items == nil {
    items = []Item{}
}
json.NewEncoder(w).Encode(items)
```

3. **Beste Lösung - JSONResponse Helper:**

```go
JSONResponse(w, items)  // Handled alles automatisch
```

### Wann tritt das Problem auf?

- `[]Struct{}` ist nil nach `var items []Struct`
- SQL Queries ohne Ergebnisse (`rows.Next()` wird nie aufgerufen)
- Uninitialisierte Slice-Felder in Structs
- Jede Funktion die `([]T, error)` zurückgibt ohne Daten

### Symptome im Frontend

```plain
TypeError: can't access property "length", items is null
TypeError: items.map is not a function
```

### Checklist für neue Handler

- [ ] Verwende `JSONResponse(w, data)` statt `json.NewEncoder(w).Encode(data)`
- [ ] Bei SQL Queries: Initialisiere Slice mit `items := []T{}`
- [ ] Bei Struct-Feldern: Verwende `json:"field,omitempty"` Tag
- [ ] Frontend: Defensive Checks hinzufügen (`items && items.length > 0`)

## Weitere Best Practices

### Error Handling

Verwende konsistente Error Messages und HTTP Status Codes:

- 400 Bad Request - Ungültige Eingabe
- 401 Unauthorized - Nicht eingeloggt
- 403 Forbidden - Keine Berechtigung
- 404 Not Found - Ressource nicht gefunden
- 500 Internal Server Error - Serverfehler

### SQL Null Handling

```go
var name sql.NullString
err := row.Scan(&name)
if name.Valid {
    // name.String enthält den Wert
}
```

### Validation

Prüfe immer Eingabedaten bevor sie in die DB gehen:

- Required Fields
- String Lengths
- Foreign Key Existenz
- Business Logic Constraints
