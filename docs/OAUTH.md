# OAuth 2.0 Integration Guide

This guide explains how OAuth 2.0 will be integrated into the New Pay platform for Google and Facebook authentication.

## Overview

The backend is configured to support OAuth 2.0 authentication with:
- Google OAuth
- Facebook OAuth

## Current Status

✅ **Completed:**
- Database schema supports OAuth users
- Configuration management for OAuth credentials
- User model includes OAuth provider fields

⏳ **Pending Implementation:**
- OAuth authentication handlers
- Callback endpoints
- Token exchange logic
- Provider-specific user data mapping

## Configuration

OAuth settings are managed through environment variables in `.env`:

### Google OAuth

```env
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/google/callback
```

### Facebook OAuth

```env
FACEBOOK_CLIENT_ID=your_facebook_app_id
FACEBOOK_CLIENT_SECRET=your_facebook_app_secret
FACEBOOK_REDIRECT_URL=http://localhost:8080/api/v1/auth/facebook/callback
```

## Getting OAuth Credentials

### Google OAuth Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Google+ API
4. Go to "Credentials" → "Create Credentials" → "OAuth 2.0 Client ID"
5. Configure the OAuth consent screen
6. Add authorized redirect URIs:
   - `http://localhost:8080/api/v1/auth/google/callback` (development)
   - `https://yourdomain.com/api/v1/auth/google/callback` (production)
7. Copy the Client ID and Client Secret

### Facebook OAuth Setup

1. Go to [Facebook Developers](https://developers.facebook.com/)
2. Create a new app or select an existing one
3. Add "Facebook Login" product
4. Configure OAuth redirect URIs:
   - `http://localhost:8080/api/v1/auth/facebook/callback` (development)
   - `https://yourdomain.com/api/v1/auth/facebook/callback` (production)
5. Copy the App ID and App Secret

## Implementation Plan

### Phase 1: Google OAuth

**Endpoints to implement:**

1. `GET /api/v1/auth/google` - Initiate Google OAuth flow
   - Redirect to Google's OAuth consent screen
   
2. `GET /api/v1/auth/google/callback` - Handle Google OAuth callback
   - Exchange authorization code for tokens
   - Fetch user profile from Google
   - Create or update user in database
   - Return JWT tokens

**User Flow:**

```
User → Click "Sign in with Google"
     → GET /api/v1/auth/google
     → Redirect to Google
     → User authorizes
     → Google redirects to callback
     → Backend creates/updates user
     → Returns JWT tokens
```

### Phase 2: Facebook OAuth

Similar implementation to Google OAuth with Facebook-specific endpoints.

## Database Schema

The `users` table already supports OAuth:

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),  -- NULL for OAuth users
    oauth_provider VARCHAR(50),  -- 'google', 'facebook', etc.
    oauth_provider_id VARCHAR(255),  -- Provider's user ID
    -- ... other fields
);
```

## Security Considerations

1. **State Parameter**: Use CSRF protection with state parameter
2. **Token Storage**: Store OAuth tokens securely if needed for API calls
3. **Scope Limitation**: Request only necessary permissions
4. **Email Verification**: Mark OAuth emails as verified by default
5. **Account Linking**: Prevent duplicate accounts with same email

## Testing OAuth Integration

### Development Testing

1. Update `.env` with OAuth credentials
2. Start the application
3. Use Postman or browser to test OAuth flow
4. Check callback handling and token generation

### Production Testing

1. Configure production redirect URLs
2. Test with real OAuth providers
3. Verify error handling
4. Test account linking scenarios

## Error Handling

Common OAuth errors to handle:

- `access_denied` - User denied access
- `invalid_grant` - Authorization code is invalid
- `invalid_client` - Invalid client credentials
- Network errors during token exchange
- Missing or invalid user data from provider

## Future Enhancements

- [ ] LinkedIn OAuth integration
- [ ] GitHub OAuth integration
- [ ] Microsoft OAuth integration
- [ ] Apple Sign In
- [ ] Account linking (merge OAuth and email/password accounts)
- [ ] OAuth token refresh for API access
- [ ] Revoke OAuth access endpoint

## Example Implementation (Pseudocode)

```go
// Google OAuth handler (to be implemented)
func (h *AuthHandler) GoogleOAuth(w http.ResponseWriter, r *http.Request) {
    // Generate state token for CSRF protection
    state := generateStateToken()
    
    // Build Google OAuth URL
    url := buildGoogleAuthURL(state)
    
    // Redirect user to Google
    http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Google OAuth callback (to be implemented)
func (h *AuthHandler) GoogleOAuthCallback(w http.ResponseWriter, r *http.Request) {
    // Verify state token
    // Exchange code for tokens
    // Fetch user profile from Google
    // Create or update user in database
    // Generate JWT tokens
    // Return response
}
```

## Resources

- [Google OAuth 2.0 Documentation](https://developers.google.com/identity/protocols/oauth2)
- [Facebook Login Documentation](https://developers.facebook.com/docs/facebook-login)
- [OAuth 2.0 RFC](https://tools.ietf.org/html/rfc6749)

## Support

For OAuth integration questions, please refer to the respective provider's documentation or contact the development team.
