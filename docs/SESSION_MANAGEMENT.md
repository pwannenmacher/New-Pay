# Session Management Guide

## Overview

The New Pay backend includes comprehensive session management with JWT Token Identifier (JTI) based tracking. This enables granular session control including per-device logout and secure session invalidation.

## Architecture

### Session Storage

Sessions are stored in the `sessions` table in PostgreSQL with the following schema:

```sql
CREATE TABLE sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id VARCHAR(255) NOT NULL,  -- Groups access + refresh tokens from same login
    jti VARCHAR(255) NOT NULL UNIQUE,  -- JWT Token Identifier
    token_type VARCHAR(20) DEFAULT 'refresh',
    expires_at TIMESTAMP NOT NULL,
    last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ip_address VARCHAR(45),
    user_agent TEXT
);
```

### Key Concepts

- **JTI (JWT ID)**: Unique identifier for each individual token (access or refresh)
- **Session ID**: Groups access and refresh tokens issued together during a single login
- **Token Type**: Distinguishes between `access` and `refresh` tokens
- **Multi-Device Support**: Each login creates a new session_id, enabling independent sessions across devices

### Session Repository

The `SessionRepository` provides the following methods:

- `Create(userID, sessionID, jti, tokenType, expiresAt, ipAddress, userAgent)` - Create a new session entry
- `GetByJTI(jti)` - Retrieve session by JWT Token Identifier
- `GetByUserID(userID)` - Get all active sessions for a user
- `DeleteBySessionID(sessionID)` - Delete all tokens from a specific login session (current device logout)
- `DeleteAllUserSessions(userID)` - Invalidate all user sessions (logout from all devices)
- `DeleteExpiredSessions()` - Cleanup expired sessions

## Usage

### Creating Sessions on Login

When a user logs in, two session entries are created (one for access token, one for refresh token):

```go
// Generate a unique session ID for this login
sessionID := GenerateSessionID() // e.g., base64-encoded random bytes

// Create access token with JTI
accessToken, accessJTI, err := authService.GenerateToken(user)

// Create refresh token with JTI
refreshToken, refreshJTI, err := authService.GenerateRefreshToken(user)

// Store both in database with same session_id
sessionRepo.Create(user.ID, sessionID, accessJTI, "access", accessExpiry, ip, userAgent)
sessionRepo.Create(user.ID, sessionID, refreshJTI, "refresh", refreshExpiry, ip, userAgent)
```

### Validating a Token

On each authenticated request, validate the JTI exists in the sessions table:

```go
// Extract JTI from JWT claims
jti := claims.ID

// Check if session exists
session, err := sessionRepo.GetByJTI(jti)
if err != nil {
    // Token has been invalidated
    return unauthorized
}

// Token is valid
```

### Logout (Current Device Only)

When a user logs out from one browser/device:

```go
// Extract JTI from refresh token cookie
jti, _ := authService.ExtractJTI(refreshToken)

// Get session to find session_id
session, _ := sessionRepo.GetByJTI(jti)

// Delete all tokens (access + refresh) from this login session
sessionRepo.DeleteBySessionID(session.SessionID)
```

**Result**: Only the current browser/device is logged out. Other devices remain logged in.

### Logout from All Devices

When a user wants to logout from all devices:

```go
sessionRepo.DeleteAllUserSessions(userID)
```

### Token Refresh

When refreshing tokens, create a NEW session_id for security:

```go
// Validate old refresh token
oldSession, err := sessionRepo.GetByJTI(oldJTI)

// Generate new session_id
newSessionID := GenerateSessionID()

// Create new tokens with new session_id
// Delete old tokens by old session_id
sessionRepo.DeleteBySessionID(oldSession.SessionID)
```

### Password Change

When a user changes their password, invalidate all existing sessions:

```go
// Change password
err := changePassword(userID, newPassword)

// Invalidate all sessions
err := sessionRepo.DeleteAllUserSessions(userID)

// User must login again from all devices
```

## Security Considerations

1. **Dual Validation**: Always check both JWT signature AND JTI existence in database
2. **Expired Sessions**: Run periodic cleanup of expired sessions
3. **Session ID Rotation**: Generate new session_id on token refresh for enhanced security
4. **IP Tracking**: Store IP addresses for security auditing
5. **User Agent**: Track devices for multi-device management
6. **Token Type Tracking**: Distinguish between access and refresh tokens for proper invalidation

## Multi-Device Behavior

| Action | Current Device | Other Devices |
|--------|---------------|---------------|
| Logout | ✅ Logged out | ✅ Stay logged in |
| Logout from all devices | ✅ Logged out | ✅ Logged out |
| Password change | ✅ Logged out | ✅ Logged out |
| Token refresh | ✅ New tokens | ✅ Stay logged in |

## Implementation Status

✅ Session repository with JTI-based tracking  
✅ Database schema with session_id column  
✅ Login creates linked access + refresh sessions  
✅ Logout invalidates only current device  
✅ Middleware validates JTI on every request  
✅ Token refresh creates new session_id  
✅ Multi-device support tested and working  
⏳ Automatic session cleanup job (to be implemented)  
⏳ Session listing endpoint for users (to be implemented)


## Future Enhancements

- Add session listing endpoint: `GET /api/v1/users/sessions`
- Add session termination endpoint: `DELETE /api/v1/users/sessions/{id}`
- Add "logout from all devices" endpoint: `POST /api/v1/auth/logout-all`
- Implement automatic cleanup job for expired sessions
- Add session activity tracking and suspicious activity detection
- Add device fingerprinting for better session management
