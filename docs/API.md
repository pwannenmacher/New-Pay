# New Pay API Documentation

## Overview

The New Pay API provides comprehensive backend services for a salary estimation and peer review platform. The API is built with security, scalability, and developer experience in mind.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

Most endpoints require authentication using JWT (JSON Web Tokens). Include the token in the Authorization header:

```
Authorization: Bearer <your_jwt_token>
```

## Response Format

All responses are in JSON format.

### Success Response

```json
{
  "data": { ... },
  "message": "Success message"
}
```

### Error Response

```json
{
  "error": "Error message"
}
```

## HTTP Status Codes

- `200 OK` - Request succeeded
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request parameters
- `401 Unauthorized` - Authentication required or invalid
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

## Endpoints

### Authentication

#### Register

Create a new user account.

```http
POST /auth/register
```

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securepassword",
  "first_name": "John",
  "last_name": "Doe"
}
```

**Response (201):**
```json
{
  "message": "Registration successful. Please check your email to verify your account.",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe"
  }
}
```

#### Login

Authenticate a user and receive JWT tokens.

```http
POST /auth/login
```

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securepassword"
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGci...",
  "refresh_token": "eyJhbGci...",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe"
  }
}
```

#### Verify Email

Verify user's email address using the token sent via email.

```http
GET /auth/verify-email?token={token}
```

**Response (200):**
```json
{
  "message": "Email verified successfully"
}
```

#### Request Password Reset

Request a password reset email.

```http
POST /auth/password-reset/request
```

**Request Body:**
```json
{
  "email": "user@example.com"
}
```

**Response (200):**
```json
{
  "message": "If the email exists, a password reset link has been sent"
}
```

#### Reset Password

Reset password using the token from the email.

```http
POST /auth/password-reset/confirm
```

**Request Body:**
```json
{
  "token": "reset_token_from_email",
  "new_password": "newsecurepassword"
}
```

**Response (200):**
```json
{
  "message": "Password reset successfully"
}
```

#### Refresh Token

Get a new access token using a refresh token.

```http
POST /auth/refresh
```

**Request Body:**
```json
{
  "refresh_token": "eyJhbGci..."
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGci..."
}
```

### User Management

#### Get Profile

Get the authenticated user's profile.

```http
GET /users/profile
```

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200):**
```json
{
  "id": 1,
  "email": "user@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "email_verified": true,
  "email_verified_at": "2024-01-15T10:30:00Z",
  "is_active": true,
  "last_login_at": "2024-01-20T15:45:00Z",
  "created_at": "2024-01-15T10:00:00Z",
  "roles": [
    {
      "id": 3,
      "name": "user",
      "description": "Regular user with basic access"
    }
  ]
}
```

#### Update Profile

Update the authenticated user's profile.

```http
POST /users/profile/update
```

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
  "first_name": "John",
  "last_name": "Smith"
}
```

**Response (200):**
```json
{
  "message": "Profile updated successfully",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Smith"
  }
}
```

### Admin Endpoints

These endpoints require admin role.

#### Get User by ID

Get user information by ID.

```http
GET /admin/users/get?id={user_id}
```

**Headers:**
```
Authorization: Bearer <admin_access_token>
```

**Response (200):**
```json
{
  "id": 2,
  "email": "otheruser@example.com",
  "first_name": "Jane",
  "last_name": "Doe",
  "email_verified": true,
  "is_active": true,
  "roles": [...]
}
```

#### Assign Role to User

Assign a role to a user.

```http
POST /admin/users/assign-role
```

**Headers:**
```
Authorization: Bearer <admin_access_token>
```

**Request Body:**
```json
{
  "user_id": 2,
  "role_id": 2
}
```

**Response (200):**
```json
{
  "message": "Role assigned successfully"
}
```

#### Remove Role from User

Remove a role from a user.

```http
POST /admin/users/remove-role
```

**Headers:**
```
Authorization: Bearer <admin_access_token>
```

**Request Body:**
```json
{
  "user_id": 2,
  "role_id": 2
}
```

**Response (200):**
```json
{
  "message": "Role removed successfully"
}
```

### Health Check

#### Health

Check API and database health.

```http
GET /health
```

**Response (200):**
```json
{
  "status": "healthy",
  "version": "1.0.0"
}
```

## Rate Limiting

The API implements rate limiting to prevent abuse:

- Default: 100 requests per minute per IP
- Configurable via environment variables

When rate limit is exceeded, you'll receive a `429 Too Many Requests` response:

```json
{
  "error": "Rate limit exceeded. Please try again later."
}
```

## Roles and Permissions

### Default Roles

1. **admin** - Full system access
   - All permissions

2. **reviewer** - Review management
   - users.read
   - reviews.create
   - reviews.read
   - reviews.update

3. **user** - Basic access
   - users.read
   - reviews.read

### Available Permissions

- `users.create` - Create new users
- `users.read` - Read user information
- `users.update` - Update user information
- `users.delete` - Delete users
- `roles.create` - Create new roles
- `roles.read` - Read role information
- `roles.update` - Update roles
- `roles.delete` - Delete roles
- `permissions.read` - Read permissions
- `permissions.assign` - Assign permissions to roles
- `audit.read` - Read audit logs
- `reviews.create` - Create reviews
- `reviews.read` - Read reviews
- `reviews.update` - Update reviews
- `reviews.delete` - Delete reviews

## Security

### Password Requirements

- Minimum 8 characters
- Recommended: Mix of uppercase, lowercase, numbers, and special characters

### Token Expiration

- Access Token: 24 hours (default)
- Refresh Token: 7 days (default)
- Email Verification Token: 24 hours
- Password Reset Token: 1 hour

### Security Headers

All responses include security headers:

- `X-Content-Type-Options: nosniff`
- `X-XSS-Protection: 1; mode=block`
- `X-Frame-Options: DENY`
- `Strict-Transport-Security: max-age=31536000; includeSubDomains`
- `Referrer-Policy: strict-origin-when-cross-origin`

## CORS

CORS is configured to allow requests from specified origins. Configure via environment variables:

```
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
```

## Error Codes

### Authentication Errors

- `INVALID_CREDENTIALS` - Email or password is incorrect
- `EMAIL_NOT_VERIFIED` - Email verification required
- `USER_INACTIVE` - User account is inactive
- `INVALID_TOKEN` - JWT token is invalid or expired

### Validation Errors

- `INVALID_EMAIL` - Email format is invalid
- `PASSWORD_TOO_SHORT` - Password must be at least 8 characters
- `REQUIRED_FIELD` - Required field is missing

### Authorization Errors

- `INSUFFICIENT_PERMISSIONS` - User lacks required permissions
- `ROLE_REQUIRED` - Specific role is required

## Examples

### Using cURL

#### Register a user

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "securepassword",
    "first_name": "John",
    "last_name": "Doe"
  }'
```

#### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "securepassword"
  }'
```

#### Get profile (with authentication)

```bash
curl -X GET http://localhost:8080/api/v1/users/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## Development

For local development, see the main README.md for setup instructions.

## Support

For API issues or questions, please open an issue on GitHub or contact support@newpay.com.
