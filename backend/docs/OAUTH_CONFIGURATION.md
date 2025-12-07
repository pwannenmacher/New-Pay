# OAuth/SSO Configuration

## Overview

The application supports configurable OAuth/SSO authentication through environment variables. Unlike hardcoded provider integrations, this flexible approach allows you to integrate with any OAuth 2.0 compatible provider.

## Configuration

OAuth/SSO is configured entirely through environment variables in the `.env` file:

```bash
# OAuth/SSO Configuration
OAUTH_ENABLED=true                    # Set to true to enable SSO
OAUTH_PROVIDER_NAME=Google            # Display name shown in UI
OAUTH_CLIENT_ID=your_client_id        # OAuth client ID from provider
OAUTH_CLIENT_SECRET=your_secret       # OAuth client secret from provider
OAUTH_REDIRECT_URL=http://localhost:8080/api/v1/auth/oauth/callback
OAUTH_AUTH_URL=https://provider.com/auth          # Provider's authorization endpoint
OAUTH_TOKEN_URL=https://provider.com/token        # Provider's token endpoint
OAUTH_USER_INFO_URL=https://provider.com/userinfo # Provider's user info endpoint
```

## Supported Providers

Any OAuth 2.0 compatible provider can be used. Here are some common examples:

### Google

```bash
OAUTH_ENABLED=true
OAUTH_PROVIDER_NAME=Google
OAUTH_CLIENT_ID=your_google_client_id
OAUTH_CLIENT_SECRET=your_google_secret
OAUTH_REDIRECT_URL=http://localhost:8080/api/v1/auth/oauth/callback
OAUTH_AUTH_URL=https://accounts.google.com/o/oauth2/v2/auth
OAUTH_TOKEN_URL=https://oauth2.googleapis.com/token
OAUTH_USER_INFO_URL=https://www.googleapis.com/oauth2/v2/userinfo
```

### Microsoft / Azure AD

```bash
OAUTH_ENABLED=true
OAUTH_PROVIDER_NAME=Microsoft
OAUTH_CLIENT_ID=your_azure_client_id
OAUTH_CLIENT_SECRET=your_azure_secret
OAUTH_REDIRECT_URL=http://localhost:8080/api/v1/auth/oauth/callback
OAUTH_AUTH_URL=https://login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize
OAUTH_TOKEN_URL=https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token
OAUTH_USER_INFO_URL=https://graph.microsoft.com/v1.0/me
```

### Keycloak

```bash
OAUTH_ENABLED=true
OAUTH_PROVIDER_NAME=Keycloak
OAUTH_CLIENT_ID=your_keycloak_client_id
OAUTH_CLIENT_SECRET=your_keycloak_secret
OAUTH_REDIRECT_URL=http://localhost:8080/api/v1/auth/oauth/callback
OAUTH_AUTH_URL=https://keycloak.example.com/auth/realms/{realm}/protocol/openid-connect/auth
OAUTH_TOKEN_URL=https://keycloak.example.com/auth/realms/{realm}/protocol/openid-connect/token
OAUTH_USER_INFO_URL=https://keycloak.example.com/auth/realms/{realm}/protocol/openid-connect/userinfo
```

### GitHub

```bash
OAUTH_ENABLED=true
OAUTH_PROVIDER_NAME=GitHub
OAUTH_CLIENT_ID=your_github_client_id
OAUTH_CLIENT_SECRET=your_github_secret
OAUTH_REDIRECT_URL=http://localhost:8080/api/v1/auth/oauth/callback
OAUTH_AUTH_URL=https://github.com/login/oauth/authorize
OAUTH_TOKEN_URL=https://github.com/login/oauth/access_token
OAUTH_USER_INFO_URL=https://api.github.com/user
```

## Frontend Behavior

The frontend automatically adapts based on the backend configuration:

- When `OAUTH_ENABLED=false`: No SSO/OAuth buttons are shown on login/register pages
- When `OAUTH_ENABLED=true`: A "Sign in with {OAUTH_PROVIDER_NAME}" button appears

The frontend fetches the OAuth configuration from the backend endpoint `/api/v1/config/oauth`, which returns:

```json
{
  "enabled": true,
  "provider_name": "Google"
}
```

## Configuration Endpoint

**GET** `/api/v1/config/oauth`

Returns the public OAuth configuration for the frontend.

**Response:**
```json
{
  "enabled": boolean,
  "provider_name": string
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/config/oauth
```

## Setup Instructions

1. Register your application with your OAuth provider to obtain:
   - Client ID
   - Client Secret
   - Authorized redirect URIs (add your callback URL)

2. Update your `.env` file with the provider-specific values

3. Restart the application:
   ```bash
   # Docker
   docker-compose up -d --build
   
   # Local development
   go build -o bin/api cmd/api/main.go cmd/api/helpers.go
   ./bin/api
   ```

4. The OAuth/SSO button will automatically appear on the login and registration pages

## Security Notes

- Always use HTTPS in production for OAuth redirect URLs
- Keep `OAUTH_CLIENT_SECRET` secure and never commit it to version control
- Use environment-specific `.env` files for different deployments
- Validate that the provider's URLs are correct and use HTTPS
- Configure proper allowed origins in `CORS_ALLOWED_ORIGINS`

## Disabling OAuth

To disable OAuth/SSO, simply set:

```bash
OAUTH_ENABLED=false
```

The frontend will automatically hide all OAuth-related UI elements.

## Troubleshooting

### SSO button not appearing
- Check that `OAUTH_ENABLED=true` in your `.env` file
- Verify the backend is running and accessible
- Check browser console for errors fetching `/api/v1/config/oauth`
- Ensure CORS is properly configured for your frontend domain

### OAuth authentication failing
- Verify all OAuth URLs are correct for your provider
- Check that Client ID and Secret are valid
- Ensure redirect URL matches exactly what's registered with the provider
- Check provider-specific documentation for required scopes and parameters

### Configuration not updating
- Restart the backend after changing `.env` file
- If using Docker, rebuild containers: `docker-compose up -d --build`
- Clear browser cache if frontend seems to cache old config
