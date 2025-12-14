# OAuth Group to Role Mapping

Diese Dokumentation beschreibt, wie OAuth-Gruppen auf interne Anwendungsrollen gemappt werden können.

## Übersicht

Das System unterstützt automatisches Mapping von OAuth-Provider-Gruppen auf interne Rollen. Bei jedem Login werden die Gruppenzugehörigkeiten des Benutzers mit den konfigurierten Mappings abgeglichen und die Rollen entsprechend synchronisiert.

## Features

- **Provider-spezifisches Mapping**: Jeder OAuth-Provider kann eigene Gruppen-zu-Rollen-Zuordnungen haben
- **Automatische Synchronisierung**: Rollen werden bei jedem Login aktualisiert
- **Audit-Logging**: Alle Rollenänderungen werden im Audit-Log erfasst
- **Admin-Schutz**: Das System stellt sicher, dass immer mindestens ein Administrator vorhanden bleibt

## Konfiguration

Die Konfiguration erfolgt über Umgebungsvariablen pro OAuth-Provider:

### Umgebungsvariablen

Für jeden OAuth-Provider (nummeriert von 1 bis 50):

```bash
# Standard OAuth-Konfiguration
OAUTH_1_NAME=keycloak
OAUTH_1_ENABLED=true
OAUTH_1_CLIENT_ID=your-client-id
OAUTH_1_CLIENT_SECRET=your-client-secret
OAUTH_1_AUTH_URL=https://keycloak.example.com/auth/realms/master/protocol/openid-connect/auth
OAUTH_1_TOKEN_URL=https://keycloak.example.com/auth/realms/master/protocol/openid-connect/token
OAUTH_1_USER_INFO_URL=https://keycloak.example.com/auth/realms/master/protocol/openid-connect/userinfo

# Gruppen-Mapping (NEU)
OAUTH_1_GROUP_MAPPING=admins:admin,developers:user,reviewers:reviewer
OAUTH_1_GROUPS_CLAIM=groups
```

### Group Mapping Format

Das `GROUP_MAPPING` Format ist:

```
oauth-gruppe-1:rolle-1,oauth-gruppe-2:rolle-2,oauth-gruppe-3:rolle-3
```

**Beispiele:**

```bash
# Einfaches Mapping
OAUTH_1_GROUP_MAPPING=admins:admin,users:user

# Mehrere Gruppen auf dieselbe Rolle
OAUTH_1_GROUP_MAPPING=admins:admin,superusers:admin,staff:user,contractors:user

# Komplexes Mapping
OAUTH_1_GROUP_MAPPING=system-admins:admin,hr-team:user,reviewers:reviewer
```

### Groups Claim

Mit `GROUPS_CLAIM` kann der Name des Claims konfiguriert werden, der die Gruppenliste enthält:

```bash
# Standard (default: "groups")
OAUTH_1_GROUPS_CLAIM=groups

# Alternatives Beispiel für Azure AD
OAUTH_2_GROUPS_CLAIM=roles

# Beispiel für benutzerdefiniertes Claim
OAUTH_3_GROUPS_CLAIM=organization_groups
```

## Beispiel-Konfigurationen

### Keycloak

```bash
OAUTH_1_NAME=keycloak
OAUTH_1_ENABLED=true
OAUTH_1_CLIENT_ID=newpay-app
OAUTH_1_CLIENT_SECRET=your-secret-here
OAUTH_1_AUTH_URL=https://keycloak.example.com/realms/newpay/protocol/openid-connect/auth
OAUTH_1_TOKEN_URL=https://keycloak.example.com/realms/newpay/protocol/openid-connect/token
OAUTH_1_USER_INFO_URL=https://keycloak.example.com/realms/newpay/protocol/openid-connect/userinfo
OAUTH_1_GROUP_MAPPING=/admins:admin,/users:user,/reviewers:reviewer
OAUTH_1_GROUPS_CLAIM=groups
```

### Azure AD / Entra ID

```bash
OAUTH_2_NAME=azure
OAUTH_2_ENABLED=true
OAUTH_2_CLIENT_ID=your-azure-client-id
OAUTH_2_CLIENT_SECRET=your-azure-secret
OAUTH_2_AUTH_URL=https://login.microsoftonline.com/YOUR_TENANT_ID/oauth2/v2.0/authorize
OAUTH_2_TOKEN_URL=https://login.microsoftonline.com/YOUR_TENANT_ID/oauth2/v2.0/token
OAUTH_2_USER_INFO_URL=https://graph.microsoft.com/oidc/userinfo
OAUTH_2_GROUP_MAPPING=NewPay-Admins:admin,NewPay-Users:user
OAUTH_2_GROUPS_CLAIM=groups
```

### Google Workspace

```bash
OAUTH_3_NAME=google
OAUTH_3_ENABLED=true
OAUTH_3_CLIENT_ID=your-google-client-id
OAUTH_3_CLIENT_SECRET=your-google-secret
OAUTH_3_AUTH_URL=https://accounts.google.com/o/oauth2/v2/auth
OAUTH_3_TOKEN_URL=https://oauth2.googleapis.com/token
OAUTH_3_USER_INFO_URL=https://openidconnect.googleapis.com/v1/userinfo
# Google Groups erfordern zusätzliche API-Calls - möglicherweise nicht direkt im userinfo verfügbar
OAUTH_3_GROUP_MAPPING=admin@example.com:admin
OAUTH_3_GROUPS_CLAIM=hd
```

## Verfügbare Rollen

Das System verfügt standardmäßig über folgende Rollen:

- `admin` - Administratoren mit vollen Rechten
- `user` - Standard-Benutzer
- `reviewer` - Reviewer-Rolle für Selbsteinschätzungen

Weitere Rollen können in der Datenbank definiert werden.

## Funktionsweise

### 1. Login-Prozess

1. Benutzer authentifiziert sich über OAuth-Provider
2. System erhält Access Token
3. UserInfo-Endpoint wird aufgerufen
4. Gruppen werden aus dem konfigurierten Claim extrahiert
5. Gruppen werden mit dem Mapping verglichen
6. Rollen werden synchronisiert

### 2. Rollen-Synchronisierung

Die Synchronisierung erfolgt folgendermaßen:

1. **Ermittlung Ziel-Rollen**: Welche Rollen sollte der Benutzer basierend auf seinen Gruppen haben?
2. **Vergleich mit aktuellen Rollen**: Was hat der Benutzer bereits?
3. **Rollen hinzufügen**: Fehlende Rollen werden zugewiesen
4. **Rollen entfernen**: Nicht mehr zutreffende Rollen werden entfernt (mit Admin-Schutz)
5. **Audit-Logging**: Alle Änderungen werden protokolliert

**Wichtig**: Es werden nur Rollen synchronisiert, die im Mapping konfiguriert sind. Manuell zugewiesene Rollen, die nicht im Mapping vorkommen, bleiben unberührt.

### 3. Admin-Schutz

Das System verhindert, dass der letzte Administrator aus dem System entfernt wird:

- Vor dem Entfernen der Admin-Rolle wird geprüft, ob noch andere Admins existieren
- Wenn nur noch ein Admin vorhanden ist, wird die Rolle nicht entfernt
- Eine Warnung wird im Log ausgegeben
- Der Vorgang wird im Audit-Log dokumentiert

## Audit-Logging

Alle rollenrelevanten Aktionen werden im Audit-Log erfasst:

### Ereignis-Typen

- `user.roles.added` - Rollen wurden hinzugefügt
- `user.roles.removed` - Rollen wurden entfernt
- `user.roles.sync.error` - Fehler bei der Synchronisierung
- `user.oauth.login` - OAuth-Login erfolgreich

### Beispiel Audit-Log-Einträge

```
Action: user.roles.added
Resource: users
Details: Roles added from OAuth groups (keycloak): [admin, reviewer]

Action: user.roles.removed
Resource: users
Details: Roles removed based on OAuth groups (keycloak): [user]
```

## Sicherheitsüberlegungen

### 1. Gruppen-Claim Validierung

- Stelle sicher, dass der OAuth-Provider vertrauenswürdig ist
- Prüfe, dass der Groups-Claim nicht vom Client manipuliert werden kann
- Verwende HTTPS für alle OAuth-Endpunkte

### 2. Rollenberechtigungen

- Definiere Gruppen-Mappings restriktiv
- Vergib Admin-Rechte nur an vertrauenswürdige Gruppen
- Dokumentiere, welche OAuth-Gruppen welche Berechtigungen erhalten

### 3. Fallback-Strategie

Wenn kein Gruppen-Mapping konfiguriert ist:

- Neue Benutzer erhalten die Standard-Rolle "user"
- Der erste Benutzer im System erhält automatisch "admin"
- Bestehende Rollen bleiben unverändert

## Troubleshooting

### Rollen werden nicht synchronisiert

1. **Prüfe die Logs**: Suche nach "Extracted OAuth groups" im Log
2. **Verifiziere Groups Claim**: Ist der richtige Claim-Name konfiguriert?
3. **Teste UserInfo Response**: Rufe den UserInfo-Endpoint manuell auf
4. **Prüfe Mapping-Syntax**: Format `gruppe:rolle,gruppe:rolle`

### Admin-Rolle kann nicht entfernt werden

Dies ist beabsichtigt, wenn es der letzte Admin ist. Lösung:

1. Weise einem anderen Benutzer die Admin-Rolle zu
2. Dann kann die Rolle vom ersten Benutzer entfernt werden

### Gruppen werden nicht vom Provider gesendet

Provider-spezifische Konfiguration erforderlich:

- **Keycloak**: Client Mappers für Groups konfigurieren
- **Azure AD**: Group Claims im App-Manifest aktivieren
- **Google**: Directory API aktivieren (komplexer)

## Beispiel Docker Compose

```yaml
services:
  backend:
    environment:
      # OAuth Provider 1: Keycloak
      - OAUTH_1_NAME=keycloak
      - OAUTH_1_ENABLED=true
      - OAUTH_1_CLIENT_ID=newpay
      - OAUTH_1_CLIENT_SECRET=${KEYCLOAK_SECRET}
      - OAUTH_1_AUTH_URL=https://keycloak.example.com/realms/newpay/protocol/openid-connect/auth
      - OAUTH_1_TOKEN_URL=https://keycloak.example.com/realms/newpay/protocol/openid-connect/token
      - OAUTH_1_USER_INFO_URL=https://keycloak.example.com/realms/newpay/protocol/openid-connect/userinfo
      - OAUTH_1_GROUP_MAPPING=newpay-admins:admin,newpay-users:user,newpay-reviewers:reviewer
      - OAUTH_1_GROUPS_CLAIM=groups
      
      # OAuth Provider 2: Azure AD
      - OAUTH_2_NAME=azure
      - OAUTH_2_ENABLED=true
      - OAUTH_2_CLIENT_ID=${AZURE_CLIENT_ID}
      - OAUTH_2_CLIENT_SECRET=${AZURE_CLIENT_SECRET}
      - OAUTH_2_AUTH_URL=https://login.microsoftonline.com/${AZURE_TENANT_ID}/oauth2/v2.0/authorize
      - OAUTH_2_TOKEN_URL=https://login.microsoftonline.com/${AZURE_TENANT_ID}/oauth2/v2.0/token
      - OAUTH_2_USER_INFO_URL=https://graph.microsoft.com/oidc/userinfo
      - OAUTH_2_GROUP_MAPPING=App-Admins:admin,App-Users:user
      - OAUTH_2_GROUPS_CLAIM=groups
```

## Migration bestehender Benutzer

Wenn OAuth-Gruppen-Mapping für bestehende Benutzer aktiviert wird:

1. Bestehende Rollen bleiben zunächst erhalten
2. Beim nächsten OAuth-Login werden Rollen synchronisiert
3. Nur Rollen aus dem Mapping werden angepasst
4. Manuell zugewiesene Rollen außerhalb des Mappings bleiben erhalten

**Empfehlung**: Teste das Mapping zuerst mit einem Test-Account, bevor es produktiv aktiviert wird.
