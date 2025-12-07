# JWT Key Management

## Problem

ECDSA (ES256) keys regenerate on restart if not persisted, invalidating all sessions.

## Generating Keys

```bash
go run backend/scripts/generate-jwt-keys.go
```

Output:

- `jwt-private-key.pem` (file)
- Single-line format for `.env`

## Configuration

Add to `.env`:

```env
JWT_SECRET=-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEI...\n-----END EC PRIVATE KEY-----\n
```

Note: `\n` are literal escape sequences (backslash + n).

On startup, check logs:

```log
✓ Loaded persistent ECDSA private key from JWT_SECRET
```

If key not loaded:

```log
⚠ Generating new ECDSA key pair (sessions will be invalidated on restart)
```

## Security

- Never commit `.pem` files
- File permissions: 600
- Use secrets manager in production
- Key rotation invalidates all sessions

## Docker

Key persists across container restarts via `.env` file.

Update key:

```bash
go run backend/scripts/generate-jwt-keys.go
# Update .env
docker compose restart api
```

**Quick test:**

```bash
# Login and save token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@newpay.com","password":"admin123"}' | jq -r '.access_token')

# Use token to access protected endpoint
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/users/profile

# Restart backend
docker-compose restart api

# Wait for startup
sleep 5

# Try same token again - should still work
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/users/profile
```

## Troubleshooting

### Sessions still invalidated after restart

- Verify the `JWT_SECRET` in `.env` contains the complete PEM-encoded key with `\n` escape sequences
- Check that the `.env` file is being loaded by the application
- Ensure the key format is correct (starts with `-----BEGIN EC PRIVATE KEY-----`)

### Authentication errors after adding key

- The key format may be incorrect
- Make sure there are no extra spaces or newlines in the `.env` value
- Try regenerating the key with the script

### "Failed to parse EC private key" error

- The PEM block format is invalid
- Regenerate the key using the provided script
- Ensure you copied the entire key including the header and footer lines
