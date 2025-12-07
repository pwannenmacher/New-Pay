# Multi-Provider OAuth Configuration

This document describes how to configure multiple OAuth providers in New Pay.

## Overview

New Pay supports configuring multiple OAuth providers simultaneously. Users can login or register using any enabled provider. The system automatically manages OAuth connections and links them to user accounts.

## Configuration

OAuth providers are configured through environment variables using a numbered prefix system.

### Global Settings

```env
# Shared redirect URL for all providers (must match OAuth app configuration)
OAUTH_REDIRECT_URL=http://localhost:8080/api/v1/auth/oauth/callback
```

### Provider Configuration

Each provider is configured using the pattern `OAUTH_N_*` where `N` is a number from 1 onwards (scans up to provider 50).

Required fields for each provider:
- `OAUTH_N_NAME` - Display name for the provider (e.g., "GitLab", "Google")
- `OAUTH_N_ENABLED` - Enable/disable the provider (true/false)
- `OAUTH_N_CLIENT_ID` - OAuth client ID from the provider
- `OAUTH_N_CLIENT_SECRET` - OAuth client secret from the provider
- `OAUTH_N_AUTH_URL` - Authorization endpoint URL
- `OAUTH_N_TOKEN_URL` - Token exchange endpoint URL
- `OAUTH_N_USER_INFO_URL` - User info endpoint URL

### Example Configuration

```env
# Provider 1 - GitLab
OAUTH_1_NAME=GitLab
OAUTH_1_ENABLED=true
OAUTH_1_CLIENT_ID=your_gitlab_client_id
OAUTH_1_CLIENT_SECRET=your_gitlab_client_secret
OAUTH_1_AUTH_URL=https://gitlab.com/oauth/authorize
OAUTH_1_TOKEN_URL=https://gitlab.com/oauth/token
OAUTH_1_USER_INFO_URL=https://gitlab.com/api/v4/user

# Provider 2 - Google
OAUTH_2_NAME=Google
OAUTH_2_ENABLED=true
OAUTH_2_CLIENT_ID=your_google_client_id
OAUTH_2_CLIENT_SECRET=your_google_client_secret
OAUTH_2_AUTH_URL=https://accounts.google.com/o/oauth2/v2/auth
OAUTH_2_TOKEN_URL=https://oauth2.googleapis.com/token
OAUTH_2_USER_INFO_URL=https://www.googleapis.com/oauth2/v2/userinfo

# Provider 3 - Authentik
OAUTH_3_NAME=Authentik
OAUTH_3_ENABLED=true
OAUTH_3_CLIENT_ID=your_authentik_client_id
OAUTH_3_CLIENT_SECRET=your_authentik_client_secret
OAUTH_3_AUTH_URL=https://your-authentik-domain.com/application/o/authorize/
OAUTH_3_TOKEN_URL=https://your-authentik-domain.com/application/o/token/
OAUTH_3_USER_INFO_URL=https://your-authentik-domain.com/application/o/userinfo/
```

## Supported Providers

### GitLab

```env
OAUTH_N_NAME=GitLab
OAUTH_N_AUTH_URL=https://gitlab.com/oauth/authorize
OAUTH_N_TOKEN_URL=https://gitlab.com/oauth/token
OAUTH_N_USER_INFO_URL=https://gitlab.com/api/v4/user
```

**Setup:**
1. Go to GitLab → User Settings → Applications
2. Create new application
3. Set redirect URI to: `http://localhost:8080/api/v1/auth/oauth/callback`
4. Select scopes: `read_user`, `openid`, `profile`, `email`

### Google

```env
OAUTH_N_NAME=Google
OAUTH_N_AUTH_URL=https://accounts.google.com/o/oauth2/v2/auth
OAUTH_N_TOKEN_URL=https://oauth2.googleapis.com/token
OAUTH_N_USER_INFO_URL=https://www.googleapis.com/oauth2/v2/userinfo
```

**Setup:**
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Enable Google+ API
4. Create OAuth 2.0 credentials
5. Add authorized redirect URI: `http://localhost:8080/api/v1/auth/oauth/callback`

### Microsoft/Azure AD

```env
OAUTH_N_NAME=Microsoft
OAUTH_N_AUTH_URL=https://login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize
OAUTH_N_TOKEN_URL=https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token
OAUTH_N_USER_INFO_URL=https://graph.microsoft.com/v1.0/me
```

**Note:** Replace `{tenant}` with your Azure AD tenant ID or `common` for multi-tenant.

**Setup:**
1. Go to [Azure Portal](https://portal.azure.com/)
2. Navigate to Azure Active Directory → App registrations
3. Create new registration
4. Add redirect URI: `http://localhost:8080/api/v1/auth/oauth/callback`
5. Generate client secret in Certificates & secrets

### Authentik

```env
OAUTH_N_NAME=Authentik
OAUTH_N_AUTH_URL=https://your-authentik-domain.com/application/o/authorize/
OAUTH_N_TOKEN_URL=https://your-authentik-domain.com/application/o/token/
OAUTH_N_USER_INFO_URL=https://your-authentik-domain.com/application/o/userinfo/
```

**Setup:**
1. Go to Authentik admin panel
2. Create new OAuth2/OIDC provider
3. Create application linked to provider
4. Set redirect URI: `http://localhost:8080/api/v1/auth/oauth/callback`
5. Note the client ID and secret

### GitHub

```env
OAUTH_N_NAME=GitHub
OAUTH_N_AUTH_URL=https://github.com/login/oauth/authorize
OAUTH_N_TOKEN_URL=https://github.com/login/oauth/access_token
OAUTH_N_USER_INFO_URL=https://api.github.com/user
```

**Setup:**
1. Go to GitHub → Settings → Developer settings → OAuth Apps
2. Create new OAuth App
3. Set Authorization callback URL: `http://localhost:8080/api/v1/auth/oauth/callback`

## How It Works

### User Registration Flow

1. New user clicks "Sign up with [Provider]"
2. User is redirected to provider's authorization page
3. After authorization, provider redirects back with authorization code
4. Backend exchanges code for access token
5. Backend fetches user info (email, name) from provider
6. New user account is created
7. OAuth connection is stored linking the user to the provider
8. User is redirected to frontend with JWT token

### User Login Flow

1. Existing user clicks "Sign in with [Provider]"
2. User is redirected to provider's authorization page
3. After authorization, provider redirects back with authorization code
4. Backend exchanges code for access token
5. Backend fetches user info from provider
6. System looks up user by:
   - Provider + Provider User ID (if OAuth connection exists)
   - Email (if user exists but no OAuth connection)
   - Creates new user if not found
7. If user found by email, creates new OAuth connection
8. User is redirected to frontend with JWT token

### Multiple Providers per User

Users can link multiple OAuth providers to their account:
- Each provider connection is stored separately in `oauth_connections` table
- Users can see all connected providers in their profile
- Users can login with any linked provider
- Database constraint prevents duplicate connections (same user + same provider)

## Security Features

- **State Parameter:** CSRF protection using random state parameter
- **HttpOnly Cookies:** State and provider stored in secure cookies
- **Token Validation:** JWT tokens with ES256 signing
- **Connection Uniqueness:** Database constraints prevent duplicate connections
- **Provider Isolation:** Each provider's credentials are isolated

## Database Schema

OAuth connections are stored in the `oauth_connections` table:

```sql
CREATE TABLE oauth_connections (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, provider),
    UNIQUE (provider, provider_id)
);
```

Constraints:
- `UNIQUE (user_id, provider)`: One connection per provider per user
- `UNIQUE (provider, provider_id)`: One New Pay account per provider account

## API Endpoints

### Get OAuth Configuration

```http
GET /api/v1/config/oauth
```

Returns list of enabled OAuth providers for frontend to display login buttons.

**Response:**
```json
{
  "enabled": true,
  "providers": [
    { "name": "GitLab" },
    { "name": "Google" },
    { "name": "Authentik" }
  ]
}
```

### Initiate OAuth Login

```http
GET /api/v1/auth/oauth/login?provider=GitLab
```

**Parameters:**
- `provider` (required): Name of the OAuth provider (must match configured OAUTH_N_NAME)

Redirects user to OAuth provider's authorization page.

### OAuth Callback

```http
GET /api/v1/auth/oauth/callback?code=...&state=...
```

**Parameters:**
- `code`: Authorization code from OAuth provider
- `state`: CSRF protection state parameter

Handles the OAuth callback, creates/logs in user, and redirects to frontend.

## Frontend Integration

The frontend automatically displays all enabled OAuth providers:

```typescript
import { useOAuthConfig } from '../../hooks/useOAuthConfig';

const { config: oauthConfig } = useOAuthConfig();

// Show OAuth buttons
{oauthConfig?.enabled && oauthConfig.providers.length > 0 && (
  <Stack gap="xs">
    {oauthConfig.providers.map((provider) => (
      <Button
        key={provider.name}
        onClick={() => handleOAuthLogin(provider.name)}
      >
        Sign in with {provider.name}
      </Button>
    ))}
  </Stack>
)}
```

## Troubleshooting

### Provider not showing in login page

1. Verify `OAUTH_N_ENABLED=true`
2. Check all required fields are set (NAME, CLIENT_ID, CLIENT_SECRET, URLs)
3. Ensure provider number is within `OAUTH_MAX_PROVIDERS` range
4. Check backend logs for configuration errors

### OAuth callback fails

1. Verify redirect URL matches in both:
   - Environment variable `OAUTH_REDIRECT_URL`
   - OAuth app configuration at provider
2. Check state cookie is set and valid
3. Verify client secret is correct
4. Check backend logs for detailed error messages

### User info not found

Different providers use different field names for user info. The backend checks:
- Email: `email` (required)
- Name: `name`, `preferred_username`, `given_name`, `family_name`
- ID: `sub`, `id`

If your provider uses different field names, you may need to customize the user info extraction logic.

### Multiple accounts with same email

The system handles this gracefully:
- First login with Provider A creates user with email
- Login with Provider B (same email) links to existing user
- Both OAuth connections are stored
- User can login with either provider

## Production Deployment

For production deployment:

1. **Use HTTPS:**
   ```env
   OAUTH_REDIRECT_URL=https://yourdomain.com/api/v1/auth/oauth/callback
   ```

2. **Update OAuth Apps:**
   - Add production redirect URL to all OAuth provider configurations
   - Keep development URL for testing

3. **Secure Cookies:**
   - Cookies automatically use `Secure` flag when TLS is detected
   - Set `SameSite=Strict` for session cookies

4. **Environment Variables:**
   - Use secrets management (e.g., Kubernetes secrets, AWS Secrets Manager)
   - Never commit `.env` file with real credentials

5. **Rate Limiting:**
   - OAuth endpoints are protected by rate limiting
   - Adjust `RATE_LIMIT_REQUESTS` if needed

## Migration from Single Provider

If you previously used single-provider configuration (OAUTH_ENABLED, OAUTH_PROVIDER_NAME):

1. Remove old variables:
   ```
   # Remove these
   OAUTH_ENABLED=...
   OAUTH_PROVIDER_NAME=...
   OAUTH_CLIENT_ID=...
   OAUTH_CLIENT_SECRET=...
   OAUTH_AUTH_URL=...
   OAUTH_TOKEN_URL=...
   OAUTH_USER_INFO_URL=...
   ```

2. Add new multi-provider configuration:
   ```env
   OAUTH_REDIRECT_URL=http://localhost:8080/api/v1/auth/oauth/callback
   
   OAUTH_1_NAME=YourProvider
   OAUTH_1_ENABLED=true
   OAUTH_1_CLIENT_ID=your_client_id
   OAUTH_1_CLIENT_SECRET=your_client_secret
   OAUTH_1_AUTH_URL=https://...
   OAUTH_1_TOKEN_URL=https://...
   OAUTH_1_USER_INFO_URL=https://...
   ```

3. Restart the application

Existing OAuth connections will continue to work as the `provider` field matches the `OAUTH_N_NAME`.
