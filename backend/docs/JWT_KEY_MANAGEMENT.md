# JWT Key Management

## Problem

When using ECDSA (ES256) for JWT signing, a new key pair is generated on each backend restart if no persistent key is provided. This invalidates all existing user sessions, forcing all users to re-authenticate.

## Solution

The application supports loading a persistent ECDSA private key from the `JWT_SECRET` environment variable. The key must be in PEM format.

## Generating a Key Pair

Use the provided script to generate a new ECDSA P-256 key pair:

```bash
go run scripts/generate-jwt-keys.go
```

This will:
1. Generate a new ECDSA P-256 key pair
2. Save the private key to `jwt-private-key.pem`
3. Display the key in both file and single-line format for `.env`

## Using the Key

### Single-line in .env (For Docker and Production)

Copy the `JWT_SECRET=...` line from the script output and paste it into your `.env` file:

```env
JWT_SECRET=-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEI...\n-----END EC PRIVATE KEY-----\n
```

**Important:** The `\n` are literal escape sequences (backslash + n), not actual newlines. The application automatically converts these to real newlines when loading the key.

When you start the application, you should see in the logs:

```
✓ Loaded persistent ECDSA private key from JWT_SECRET
```

If you see this instead, the key is not being loaded correctly:

```
⚠ Generating new ECDSA key pair (sessions will be invalidated on restart)
```

## Security Considerations

1. **Never commit keys to git**: The `.gitignore` is configured to exclude `*.pem` files
2. **Protect the private key**: The key file should have restricted permissions (600)
3. **Rotate keys carefully**: Changing the key will invalidate all existing sessions
4. **Production deployment**: Use a secrets management system (AWS Secrets Manager, HashiCorp Vault, etc.)

## Key Rotation

If you need to rotate the JWT signing key:

1. Generate a new key pair
2. Update the `JWT_SECRET` environment variable
3. Restart the backend
4. **All users will be logged out** and need to re-authenticate

## Docker Deployment

For Docker, the key is loaded from the `.env` file during container startup. The key persists across container restarts as long as the `.env` file remains unchanged.

To update the key in Docker:

```bash
# 1. Generate new key
go run scripts/generate-jwt-keys.go

# 2. Update .env with the new JWT_SECRET value

# 3. Restart the container
docker-compose restart api
```

## Verification

To verify the key is being loaded correctly, check the application logs on startup:

```bash
docker logs newpay-api 2>&1 | grep ECDSA
```

You should see:
```
✓ Loaded persistent ECDSA private key from JWT_SECRET
```

You can also test session persistence by:

1. Log in to the application and obtain a JWT token (check browser DevTools → Application → Local Storage)
2. Note the token value
3. Restart the backend: `docker-compose restart api`
4. Try to access a protected endpoint with the same token
5. The token should still be valid (if not expired)

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
