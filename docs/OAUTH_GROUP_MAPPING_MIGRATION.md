# Migration zu OAuth Group Mapping

Dieser Leitfaden hilft beim Aktivieren des OAuth Group Mappings für bestehende Installationen.

## Überblick der Änderungen

### Neue Features

1. **Gruppen-zu-Rollen-Mapping**: OAuth-Gruppen werden auf interne Rollen gemappt
2. **Automatische Synchronisierung**: Rollen werden bei jedem Login aktualisiert
3. **Audit-Logging**: Alle Rollenänderungen werden protokolliert
4. **Admin-Schutz**: Verhindert Entfernung des letzten Admins

### Neue Umgebungsvariablen

Pro OAuth-Provider:
- `OAUTH_X_GROUP_MAPPING` - Mapping-Konfiguration
- `OAUTH_X_GROUPS_CLAIM` - Name des Claims mit Gruppen (default: "groups")

## Migrationsschritte

### Schritt 1: Backend aktualisieren

```bash
cd backend
git pull
go mod tidy
```

### Schritt 2: Umgebungsvariablen hinzufügen

**Ohne Mapping (Status Quo beibehalten):**

```bash
# Nichts tun - System verhält sich wie vorher
# Rollen bleiben manuell verwaltet
```

**Mit Mapping aktivieren:**

```bash
# In backend/.env oder docker-compose.yml
OAUTH_1_GROUP_MAPPING=admins:admin,users:user,reviewers:reviewer
OAUTH_1_GROUPS_CLAIM=groups  # optional, default ist "groups"
```

### Schritt 3: OAuth-Provider konfigurieren

#### Keycloak

1. Client Mapper für "groups" hinzufügen (siehe [Quickstart](OAUTH_GROUP_MAPPING_QUICKSTART.md))
2. Gruppen in Keycloak erstellen
3. Benutzer zu Gruppen hinzufügen

#### Azure AD

1. App-Manifest bearbeiten
2. `groupMembershipClaims`: `"SecurityGroup"` setzen
3. API-Permissions: `Group.Read.All` hinzufügen

#### GitLab

1. In GitLab sind Gruppen standardmäßig im Token enthalten
2. Claim-Name prüfen (oft `groups_direct`)

### Schritt 4: Test mit einem Testbenutzer

**Wichtig**: Teste zuerst mit einem nicht-produktiven Account!

1. Testbenutzer in OAuth-Provider erstellen
2. Zu Testgruppe hinzufügen
3. Login testen
4. Logs prüfen:
   ```bash
   docker logs newpay-backend | grep "OAuth groups"
   docker logs newpay-backend | grep "Roles added"
   ```
5. Audit-Log in der Datenbank prüfen

### Schritt 5: Produktiv aktivieren

Wenn Tests erfolgreich:

1. Produktive Umgebungsvariablen setzen
2. Backend neu starten
3. Benutzer informieren (beim nächsten Login werden Rollen aktualisiert)

## Verhalten für bestehende Benutzer

### Ohne Group Mapping konfiguriert

- **Keine Änderung**: Alles funktioniert wie vorher
- Rollen bleiben manuell verwaltet
- OAuth-Login funktioniert weiterhin

### Mit Group Mapping konfiguriert

**Beim nächsten OAuth-Login:**

1. System extrahiert Gruppen aus OAuth-Token
2. Vergleicht mit konfiguriertem Mapping
3. Synchronisiert nur Rollen, die im Mapping definiert sind
4. Andere Rollen bleiben unberührt

**Beispiel:**

```bash
# Konfiguration
OAUTH_1_GROUP_MAPPING=admins:admin,users:user

# Benutzer A:
# - Aktuell: admin, reviewer (manuell vergeben)
# - OAuth-Gruppen: admins
# - Nach Login: admin, reviewer (reviewer bleibt, da nicht im Mapping)

# Benutzer B:
# - Aktuell: admin, user (manuell vergeben)
# - OAuth-Gruppen: users
# - Nach Login: user (admin entfernt, da im Mapping aber nicht in Gruppen)
```

## Rollback

Falls Probleme auftreten:

### Option 1: Mapping deaktivieren

```bash
# Umgebungsvariable leer setzen
OAUTH_1_GROUP_MAPPING=
```

Nach Neustart: Keine Synchronisierung mehr, bestehende Rollen bleiben

### Option 2: Feature komplett deaktivieren

```bash
# OAuth-Provider temporär deaktivieren
OAUTH_1_ENABLED=false
```

Benutzer können dann mit Passwort (falls vorhanden) oder per Admin-Reset einloggen.

## Best Practices

### 1. Schrittweise Einführung

1. **Woche 1**: Mit Testbenutzer testen
2. **Woche 2**: Mit kleiner Benutzergruppe testen
3. **Woche 3**: Für alle aktivieren

### 2. Kommunikation

Informiere Benutzer:
> "Bei eurem nächsten Login werden eure Rollen automatisch aus euren OAuth-Gruppen synchronisiert. Bitte meldet Probleme sofort."

### 3. Monitoring

Überwache nach Aktivierung:

```sql
-- Rollenänderungen der letzten 24h
SELECT * FROM audit_logs 
WHERE action IN ('user.roles.added', 'user.roles.removed')
  AND created_at > NOW() - INTERVAL '24 hours'
ORDER BY created_at DESC;
```

### 4. Admin-Schutz

- Stelle sicher, dass mehrere Admins existieren
- Wenn möglich: Erstelle einen "Emergency Admin" Account ohne OAuth

## Häufige Fragen

### Werden bestehende Rollen gelöscht?

Nein, nur Rollen die im Group Mapping definiert sind, werden synchronisiert. Andere bleiben unberührt.

### Was passiert mit manuell vergebenen Rollen?

- **Im Mapping enthalten**: Werden bei jedem Login synchronisiert
- **Nicht im Mapping**: Bleiben unverändert

### Kann ich das Mapping für einzelne Benutzer deaktivieren?

Nein, aber du kannst:
1. Rollen manuell vergeben, die nicht im Mapping sind
2. Benutzer aus OAuth-Gruppen nehmen (dann keine Synchronisierung)

### Was ist mit dem ersten Admin?

Der erste Benutzer wird automatisch Admin, unabhängig vom Mapping. Danach greift das Mapping.

### Kann ich verschiedene Mappings für verschiedene Provider haben?

Ja! Jeder Provider hat eigene `OAUTH_X_GROUP_MAPPING` Konfiguration.

## Weitere Informationen

- [Vollständige Dokumentation](OAUTH_GROUP_MAPPING.md)
- [Quickstart Guide](OAUTH_GROUP_MAPPING_QUICKSTART.md)
- [OAuth-Konfiguration](OAUTH_CONFIGURATION.md)
