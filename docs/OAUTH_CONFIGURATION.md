# OAuth Configuration

## Environment Variables

```bash
OAUTH_PROVIDERS=[{"name":"Google","enabled":true,"client_id":"...","client_secret":"..."}]
OAUTH_REDIRECT_URL=http://localhost:8080/api/v1/auth/oauth/callback
OAUTH_FRONTEND_CALLBACK_URL=http://localhost:3001/oauth/callback
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
4. Restart application

## Security

- Use HTTPS in production
- Never commit secrets to version control
- Configure CORS properly
