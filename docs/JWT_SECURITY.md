# JWT Security Implementation

## Overview

The JWT authentication system has been enhanced with modern security best practices:

### 1. ECDSA (ES256) Algorithm

**Changed from:** HS256 (HMAC with SHA-256)  
**Changed to:** ES256 (Elliptic Curve Digital Signature Algorithm with P-256 and SHA-256)

**Benefits:**
- **Asymmetric encryption**: Uses public/private key pairs instead of shared secrets
- **Better security**: More resistant to brute-force attacks
- **Smaller signatures**: More efficient than RSA with equivalent security
- **Industry standard**: Recommended by NIST and used by major platforms

**Implementation:**
- Private key used for signing tokens
- Public key used for verification
- Auto-generates keys on startup (for development)
- Supports loading PEM-encoded private keys from configuration

### 2. HTTP-Only Cookie for Refresh Tokens

**Changed from:** Refresh token in response body  
**Changed to:** HTTP-Only cookie with specific path

**Cookie Configuration:**
```
Name: refresh_token
Path: /api/v1/auth/refresh
HttpOnly: true
Secure: true (in production with HTTPS)
SameSite: Strict
Max-Age: 604800 (7 days)
```

**Benefits:**
- **XSS Protection**: Cookie not accessible via JavaScript
- **CSRF Protection**: SameSite=Strict prevents cross-site requests
- **Path Restriction**: Cookie only sent to refresh endpoint
- **Automatic Management**: Browser handles storage and transmission

### 3. Token Rotation

**Feature:** Each refresh generates a new refresh token

**Benefits:**
- **Compromised token mitigation**: Old tokens become invalid after use
- **Improved security**: Reduces window of opportunity for token theft
- **Audit trail**: Easier to track token usage patterns

## API Changes

### Login Response

**Before:**
```json
{
  "access_token": "...",
  "refresh_token": "...",
  "user": {...}
}
```

**After:**
```json
{
  "access_token": "...",
  "user": {...}
}
```
+ `Set-Cookie` header with refresh_token

### Refresh Token Endpoint

**Before:**
```bash
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "..."
}
```

**After:**
```bash
POST /api/v1/auth/refresh
Cookie: refresh_token=...

# Response includes new refresh_token cookie
```

### New Logout Endpoint

```bash
POST /api/v1/auth/logout

# Clears refresh_token cookie
```

## Security Considerations

### Production Deployment

1. **Use proper private keys**: Generate and store ECDSA keys securely
2. **Enable HTTPS**: Required for Secure cookie flag
3. **Key rotation**: Implement periodic key rotation strategy
4. **Monitoring**: Track token refresh patterns for anomalies

### Key Management

For production, generate a proper ECDSA key:

```bash
# Generate private key
openssl ecparam -name prime256v1 -genkey -noout -out private-key.pem

# Extract public key
openssl ec -in private-key.pem -pubout -out public-key.pem
```

Store the private key in your secrets management system and load it via the `JWT_SECRET` environment variable.

## Testing

### Test Login with Cookie
```bash
curl -v -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}'
```

### Test Refresh with Cookie
```bash
curl -v -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Cookie: refresh_token=YOUR_TOKEN"
```

### Test Logout
```bash
curl -v -X POST http://localhost:8080/api/v1/auth/logout
```

## Migration Notes

**Breaking Changes:**
- Clients must handle cookies for refresh tokens
- Refresh endpoint no longer accepts JSON body
- New logout endpoint available

**Frontend Integration:**
```javascript
// Login
const response = await fetch('/api/v1/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email, password }),
  credentials: 'include' // Important: include cookies
});

// Refresh
const refreshResponse = await fetch('/api/v1/auth/refresh', {
  method: 'POST',
  credentials: 'include' // Sends cookie automatically
});

// Logout
await fetch('/api/v1/auth/logout', {
  method: 'POST',
  credentials: 'include'
});
```
