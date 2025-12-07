# Session Management Guide

## Overview

The New Pay backend now includes comprehensive session management that allows for individual JWT token invalidation. This is crucial for security features like "logout from all devices" and invalidating sessions when users change their passwords.

## Architecture

### Session Storage

Sessions are stored in the `sessions` table in PostgreSQL with the following schema:

```sql
CREATE TABLE sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    last_activity_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ip_address VARCHAR(45),
    user_agent TEXT
);
```

### Session Repository

The `SessionRepository` provides the following methods:

- `Create(session)` - Create a new session
- `GetByToken(token)` - Retrieve active session by token
- `GetByUserID(userID)` - Get all active sessions for a user
- `UpdateLastActivity(sessionID)` - Update session activity timestamp
- `Delete(sessionID)` - Delete specific session
- `DeleteByToken(token)` - Delete session by token (logout)
- `DeleteAllUserSessions(userID)` - Invalidate all user sessions (logout from all devices)
- `DeleteExpiredSessions()` - Cleanup expired sessions

## Usage

### Creating a Session

When a user logs in, create a session:

```go
session := &models.Session{
    ID:               generateUUID(),
    UserID:           user.ID,
    Token:            jwtToken,
    ExpiresAt:        time.Now().Add(24 * time.Hour),
    LastActivityAt:   time.Now(),
    CreatedAt:        time.Now(),
    IPAddress:        r.RemoteAddr,
    UserAgent:        r.UserAgent(),
}

err := sessionRepo.Create(session)
```

### Validating a Session

On each authenticated request, validate the session exists and is active:

```go
session, err := sessionRepo.GetByToken(token)
if err != nil {
    // Token is invalid or expired
    return unauthorized
}

// Update last activity
sessionRepo.UpdateLastActivity(session.ID)
```

### Logout (Single Device)

When a user logs out:

```go
err := sessionRepo.DeleteByToken(token)
```

### Logout from All Devices

When a user wants to logout from all devices:

```go
err := sessionRepo.DeleteAllUserSessions(userID)
```

### Password Change

When a user changes their password, invalidate all existing sessions:

```go
// Change password
err := changePassword(userID, newPassword)

// Invalidate all sessions
err := sessionRepo.DeleteAllUserSessions(userID)

// Create new session for current device
session := createNewSession(userID, newToken)
err := sessionRepo.Create(session)
```

## Security Considerations

1. **Token Validation**: Always check both JWT validity AND session existence
2. **Expired Sessions**: Run periodic cleanup of expired sessions
3. **IP Tracking**: Store IP addresses for security auditing
4. **User Agent**: Track devices for multi-device management
5. **Last Activity**: Update on each request for inactivity timeout

## Implementation Status

✅ Session repository created
✅ Database schema includes sessions table
✅ Basic CRUD operations implemented
⏳ Integration with authentication handlers (to be implemented)
⏳ Automatic session cleanup job (to be implemented)
⏳ Session listing endpoint for users (to be implemented)

## Future Enhancements

- Add session listing endpoint: `GET /api/v1/users/sessions`
- Add session termination endpoint: `DELETE /api/v1/users/sessions/{id}`
- Add "logout from all devices" endpoint: `POST /api/v1/auth/logout-all`
- Implement automatic cleanup job for expired sessions
- Add session activity tracking and suspicious activity detection
- Add device fingerprinting for better session management
