# Audit Event Types

Dieses Dokument listet alle implementierten Audit-Log-Events auf.

## Authentifizierung

### Registrierung
- `user.register` - Erfolgreiche Benutzerregistrierung
- `user.register.error` - Registrierung fehlgeschlagen
- `user.register.token.error` - Token-Generierung nach Registrierung fehlgeschlagen
- `user.register.session.error` - Session-Erstellung nach Registrierung fehlgeschlagen

### Login
- `user.login` - Erfolgreicher Login
- `user.login.failed` - Login-Versuch fehlgeschlagen
- `user.login.session.error` - Session-Erstellung während Login fehlgeschlagen

### Logout
- `user.logout` - Erfolgreicher Logout
- `user.logout.error` - Fehler beim Invalidieren der Session

### OAuth
- `user.oauth.login` - Erfolgreicher OAuth-Login
- `user.oauth.register` - Neuer Benutzer via OAuth registriert
- `user.oauth.error` - OAuth-Fehler (User-Erstellung, Token-Generierung, Session-Fehler)

### Token Management
- `user.token.refresh.error` - Token-Refresh fehlgeschlagen

## Email-Versand

- `email.verification.sent` - Verifizierungs-Email versendet
- `email.welcome.sent` - Willkommens-Email versendet
- `email.password_reset.sent` - Passwort-Reset-Email versendet

## Email-Verifizierung

- `user.email.verified` - Email erfolgreich verifiziert
- `user.email.verify.error` - Email-Verifizierung fehlgeschlagen

## Passwort-Reset

- `user.password.reset.request` - Passwort-Reset angefordert
- `user.password.reset` - Passwort erfolgreich zurückgesetzt
- `user.password.reset.error` - Passwort-Reset fehlgeschlagen

## User-Profil

- `user.profile.update` - Profil aktualisiert
- `user.profile.update.error` - Profil-Aktualisierung fehlgeschlagen

## Admin: User Management

### Status
- `update_user_status` - Benutzer-Status geändert (aktiv/inaktiv)
- `user.status.update.error` - Status-Änderung fehlgeschlagen

### Rollen
- `user.role.assign` - Rolle zugewiesen
- `user.role.assign.error` - Rollen-Zuweisung fehlgeschlagen
- `user.role.remove` - Rolle entfernt
- `user.role.remove.error` - Rollen-Entfernung fehlgeschlagen

### CRUD-Operationen
- `update_user` - Benutzer aktualisiert (Email, Name)
- `update_user.error` - Benutzer-Aktualisierung fehlgeschlagen
- `set_user_password` - Passwort gesetzt
- `set_user_password.error` - Passwort-Änderung fehlgeschlagen
- `delete_user` - Benutzer gelöscht
- `delete_user.error` - Benutzer-Löschung fehlgeschlagen

## Session Management

### Benutzer-Aktionen
- `session.delete` - Benutzer hat eigene Session gelöscht
- `session.delete.error` - Session-Löschung fehlgeschlagen
- `session.delete_all_others` - Benutzer hat alle anderen Sessions gelöscht
- `session.get.error` - Fehler beim Abrufen von Sessions

### Admin-Aktionen
- `admin.session.delete` - Admin hat Session gelöscht
- `admin.session.delete.error` - Admin Session-Löschung fehlgeschlagen
- `admin.session.delete_all_user` - Admin hat alle Sessions eines Benutzers gelöscht
- `admin.session.delete_all_user.error` - Fehler beim Löschen aller User-Sessions

## Event-Details

Alle Audit-Logs enthalten folgende Informationen:
- `user_id` - ID des betroffenen Benutzers (wenn verfügbar, NULL bei gelöschten Benutzern)
- `user_email` - Email-Adresse des Benutzers (bleibt auch nach Löschung erhalten)
- `action` - Event-Typ (siehe oben)
- `resource` - Betroffene Ressource (users, sessions, emails)
- `details` - Detaillierte Beschreibung
- `ip_address` - IP-Adresse des Clients
- `user_agent` - Browser/Client User-Agent
- `created_at` - Zeitstempel

**Wichtig**: Die Email-Adresse wird beim Erstellen des Audit-Logs gespeichert und bleibt auch dann erhalten, wenn der zugehörige Benutzer später gelöscht wird. So ist eine vollständige Nachvollziehbarkeit auch bei gelöschten Accounts gewährleistet.

## Compliance

Diese Audit-Logs erfüllen die Anforderungen:
- ✅ Jeder (OAuth-)Login und Logout wird erfasst
- ✅ Jede Änderung an Usern wird erfasst
- ✅ Jede Änderung an Sessions wird erfasst
- ✅ Jeder Mailversand wird erfasst
- ✅ Jeder aufgetretene Fehler wird erfasst
