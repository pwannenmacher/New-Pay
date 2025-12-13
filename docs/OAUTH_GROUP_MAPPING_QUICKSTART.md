# OAuth Gruppen-zu-Rollen Mapping - Beispielkonfiguration

## Schnellstart-Beispiel für Keycloak

```bash
# In backend/.env oder als Docker-Umgebungsvariablen:

OAUTH_1_ENABLED=true
OAUTH_1_NAME=keycloak
OAUTH_1_CLIENT_ID=newpay-app
OAUTH_1_CLIENT_SECRET=your-client-secret-here
OAUTH_1_AUTH_URL=https://keycloak.example.com/realms/newpay/protocol/openid-connect/auth
OAUTH_1_TOKEN_URL=https://keycloak.example.com/realms/newpay/protocol/openid-connect/token
OAUTH_1_USER_INFO_URL=https://keycloak.example.com/realms/newpay/protocol/openid-connect/userinfo

# Gruppen-Mapping: OAuth-Gruppe -> Interne Rolle
OAUTH_1_GROUP_MAPPING=newpay-admins:admin,newpay-users:user,newpay-reviewers:reviewer
OAUTH_1_GROUPS_CLAIM=groups
```

## Keycloak-Konfiguration

### 1. Client erstellen

1. In Keycloak Admin Console -> Clients -> Create
2. Client ID: `newpay-app`
3. Client Protocol: `openid-connect`
4. Access Type: `confidential`
5. Valid Redirect URIs: `http://localhost:8080/api/v1/auth/oauth/callback`
6. Speichern und Client Secret notieren

### 2. Gruppen erstellen

1. Groups -> New
2. Erstelle folgende Gruppen:
   - `newpay-admins`
   - `newpay-users`
   - `newpay-reviewers`

### 3. Client Mapper hinzufügen

1. Client -> newpay-app -> Mappers -> Create
2. **Name**: `groups`
3. **Mapper Type**: `Group Membership`
4. **Token Claim Name**: `groups`
5. **Full group path**: OFF (nur Gruppenname)
6. **Add to ID token**: ON
7. **Add to access token**: ON
8. **Add to userinfo**: ON
9. Speichern

### 4. Benutzer zu Gruppen hinzufügen

1. Users -> Select User -> Groups
2. Join Gruppe auswählen

## Test

1. Backend starten: `cd backend && go run cmd/api/main.go`
2. OAuth-Login testen: `http://localhost:8080/api/v1/auth/oauth/login?provider=keycloak`
3. Nach erfolgreichem Login in den Logs prüfen:
   ```
   Extracted OAuth groups provider=keycloak groups=[newpay-admins, newpay-users]
   Roles added from OAuth groups user_id=1 roles=[admin]
   ```

## Erwartetes Verhalten

- **Neuer Benutzer**: Erhält Rollen basierend auf seinen Gruppen
- **Bestehender Benutzer**: Rollen werden bei jedem Login synchronisiert
- **Gruppenänderung**: Beim nächsten Login werden Rollen angepasst
- **Audit-Log**: Alle Änderungen werden protokolliert
- **Admin-Schutz**: Letzter Admin kann nicht entfernt werden

## Beispiel-Szenarien

### Szenario 1: Benutzer wird Admin

1. Ausgangszustand: Benutzer hat Rolle "user"
2. Admin fügt Benutzer zur Gruppe "newpay-admins" in Keycloak hinzu
3. Benutzer meldet sich ab und neu an
4. System erkennt neue Gruppe
5. Rolle "admin" wird hinzugefügt
6. Audit-Log: "Roles added from OAuth groups (keycloak): [admin]"

### Szenario 2: Benutzer verliert Admin-Rechte

1. Ausgangszustand: Benutzer hat Rollen "admin" und "user"
2. Admin entfernt Benutzer aus Gruppe "newpay-admins"
3. Benutzer meldet sich ab und neu an
4. System erkennt fehlende Gruppe
5. Wenn weitere Admins existieren: Rolle "admin" wird entfernt
6. Wenn letzter Admin: Warnung im Log, Rolle bleibt erhalten

### Szenario 3: Mehrere Gruppen

```bash
OAUTH_1_GROUP_MAPPING=admins:admin,staff:user,contractors:user,reviewers:reviewer
```

- Benutzer in "admins" + "reviewers" → Rollen: admin, reviewer
- Benutzer in "staff" → Rolle: user
- Benutzer in "contractors" → Rolle: user
- Benutzer ohne Gruppen → Standard-Rolle (user für neue Benutzer)

## Troubleshooting

### Problem: Gruppen werden nicht erkannt

**Lösung**: Prüfen, ob Client Mapper korrekt konfiguriert ist

```bash
# Keycloak UserInfo-Endpoint manuell testen
curl -H "Authorization: Bearer <access_token>" \
  https://keycloak.example.com/realms/newpay/protocol/openid-connect/userinfo

# Sollte groups enthalten:
{
  "sub": "12345",
  "email": "user@example.com",
  "groups": ["newpay-admins", "newpay-users"]
}
```

### Problem: Rollen werden nicht synchronisiert

**Prüfe Logs**:
```bash
# Im Backend-Log suchen nach:
grep "Extracted OAuth groups" logs/app.log
grep "Roles added" logs/app.log
grep "Roles removed" logs/app.log
```

### Problem: Falscher Claim-Name

Manche Provider nutzen andere Claim-Namen:

```bash
# Azure AD verwendet oft "roles"
OAUTH_1_GROUPS_CLAIM=roles

# Authentik verwendet "groups"
OAUTH_1_GROUPS_CLAIM=groups

# GitLab verwendet manchmal "groups_direct"
OAUTH_1_GROUPS_CLAIM=groups_direct
```

## Mehrere Provider

```bash
# Provider 1: Keycloak (intern)
OAUTH_1_NAME=keycloak
OAUTH_1_GROUP_MAPPING=internal-admins:admin,internal-users:user
OAUTH_1_GROUPS_CLAIM=groups

# Provider 2: Azure AD (extern)
OAUTH_2_NAME=azure
OAUTH_2_GROUP_MAPPING=External-Contractors:user,External-Reviewers:reviewer
OAUTH_2_GROUPS_CLAIM=roles
```

## Siehe auch

- [Vollständige Dokumentation](OAUTH_GROUP_MAPPING.md)
- [OAuth-Konfiguration](OAUTH_CONFIGURATION.md)
