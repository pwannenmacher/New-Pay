# Session Management

## Architecture

Sessions track JWT tokens in PostgreSQL using JTI (JWT Token Identifier).

```sql
CREATE TABLE sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id VARCHAR(255) NOT NULL,
    jti VARCHAR(255) NOT NULL UNIQUE,
    token_type VARCHAR(20) DEFAULT 'refresh',
    expires_at TIMESTAMP NOT NULL,
    last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ip_address VARCHAR(45),
    user_agent TEXT
);
```

## Concepts

- **JTI**: Unique identifier for each token
- **Session ID**: Groups access and refresh tokens from same login
- **Token Type**: `access` or `refresh`

## API

### User Endpoints

- `GET /api/v1/users/sessions` - List active sessions
- `DELETE /api/v1/users/sessions/delete` - Logout current device
- `DELETE /api/v1/users/sessions/delete-all` - Logout all devices

### Admin Endpoints

- `GET /api/v1/admin/sessions/user?user_id=ID` - List user sessions
- `DELETE /api/v1/admin/sessions/delete` - Delete specific session
- `DELETE /api/v1/admin/sessions/delete-all?user_id=ID` - Delete all user sessions

## Invalidation

Sessions are invalidated on:

- User logout
- Password change
- Admin action
- Token expiration

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
