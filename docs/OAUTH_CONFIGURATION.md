# OAuth Configuration

## Environment Variables

```bash
OAUTH_PROVIDERS=[{"name":"Google","enabled":true,"client_id":"...","client_secret":"..."}]
OAUTH_REDIRECT_URL=http://localhost:8080/api/v1/auth/oauth/callback
OAUTH_FRONTEND_CALLBACK_URL=http://localhost:3001/oauth/callback

# Registration Settings
ENABLE_REGISTRATION=false
ENABLE_OAUTH_REGISTRATION=false
```

## Registration Control

### ENABLE_REGISTRATION

Controls whether users can register via the traditional email/password registration endpoint.

- `true`: Email/password registration is enabled
- `false`: Email/password registration is disabled
- **Exception**: First user can always register when database is empty (for initial setup)

### ENABLE_OAUTH_REGISTRATION

Controls whether new users can register through OAuth/SSO providers.

- `true`: New users can register via OAuth providers
- `false`: Only existing users can log in via OAuth
- **Exception**: First user can always register via OAuth when database is empty (for initial setup)
- **Existing users**: Can always log in via OAuth, regardless of this setting

### Use Cases

**Open Registration** (Public Application):
```bash
ENABLE_REGISTRATION=true
ENABLE_OAUTH_REGISTRATION=true
```

**Closed System** (Invite-Only):
```bash
ENABLE_REGISTRATION=false
ENABLE_OAUTH_REGISTRATION=false
# Admins must create users manually
```

**OAuth-Only System**:
```bash
ENABLE_REGISTRATION=false
ENABLE_OAUTH_REGISTRATION=true
# Users can only register via SSO
```

**Email/Password Only**:
```bash
ENABLE_REGISTRATION=true
ENABLE_OAUTH_REGISTRATION=false
# OAuth can be used for login by existing users only
```

## Provider Examples

### Google

```bash
{
  "name": "Google",
  "enabled": true,
  "client_id": "your_client_id",
  "client_secret": "your_secret",
  "auth_url": "https://accounts.google.com/o/oauth2/v2/auth",
  "token_url": "https://oauth2.googleapis.com/token",
  "user_info_url": "https://www.googleapis.com/oauth2/v2/userinfo"
}
```

### Microsoft

```bash
{
  "name": "Microsoft",
  "enabled": true,
  "client_id": "your_client_id",
  "client_secret": "your_secret",
  "auth_url": "https://login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize",
  "token_url": "https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token",
  "user_info_url": "https://graph.microsoft.com/v1.0/me"
}
```

### Keycloak

```bash
{
  "name": "Keycloak",
  "enabled": true,
  "client_id": "your_client_id",
  "client_secret": "your_secret",
  "auth_url": "https://keycloak.example.com/auth/realms/{realm}/protocol/openid-connect/auth",
  "token_url": "https://keycloak.example.com/auth/realms/{realm}/protocol/openid-connect/token",
  "user_info_url": "https://keycloak.example.com/auth/realms/{realm}/protocol/openid-connect/userinfo"
}
```

## Setup

1. Register application with OAuth provider
2. Add provider to `OAUTH_PROVIDERS` array in `.env`
3. Set redirect URLs in provider settings
4. Configure registration settings (`ENABLE_REGISTRATION`, `ENABLE_OAUTH_REGISTRATION`)
5. Restart application

### Initial Setup

When setting up a new instance:

1. Start with `ENABLE_OAUTH_REGISTRATION=true` (or `false` if you prefer)
2. Register the first admin user via OAuth or email/password
3. The first user automatically receives admin role
4. After first user is created, adjust registration settings as needed
5. First user can create additional users manually if registration is disabled

## Security

- Use HTTPS in production
- Never commit secrets to version control
- Configure CORS properly
