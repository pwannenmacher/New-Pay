# New Pay

New Pay is a modern platform for salary estimates and peer reviews with a robust backend API built in Go.

## Features

### Authentication & Security
- ✅ JWT-based authentication
- ✅ Password hashing with bcrypt
- ✅ Email verification after registration
- ✅ Password recovery functionality
- ✅ Session management with automatic logout
- ✅ Rate limiting to prevent abuse
- ✅ CORS configuration
- ✅ Security headers
- ✅ Input validation and sanitization

### User Management
- ✅ User registration and login
- ✅ Profile management
- ✅ Role-based access control (RBAC)
- ✅ User roles: admin, reviewer, user
- ✅ Granular permissions system
- ✅ Role assignment and management

### Audit & Logging
- ✅ Comprehensive audit logging
- ✅ Security event tracking
- ✅ IP address and user agent logging

### Email Integration
- ✅ SMTP email delivery
- ✅ Email verification emails
- ✅ Password reset emails
- ✅ Welcome emails

## Tech Stack

- **Backend**: Go 1.24+
- **Database**: PostgreSQL 16+
- **Authentication**: JWT
- **Email**: SMTP

## Prerequisites

- Go 1.24 or higher
- PostgreSQL 16 or higher
- SMTP server credentials (for email functionality)

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/pwannenmacher/New-Pay.git
cd New-Pay
```

### 2. Set up environment variables

```bash
cp .env.example .env
```

Edit `.env` and configure your database and email settings.

### 3. Start PostgreSQL (using Docker)

```bash
docker-compose up -d
```

Or use your own PostgreSQL instance.

### 4. Run database migrations

```bash
make migrate-up
```

Or manually:

```bash
psql -U newpay -d newpay_db -f migrations/001_init_schema.up.sql
```

### 5. Install dependencies

```bash
make deps
```

### 6. Run the application

```bash
make run
```

The server will start on `http://localhost:8080`

## API Endpoints

### Public Endpoints

#### Authentication

- `POST /api/v1/auth/register` - Register a new user
  ```json
  {
    "email": "user@example.com",
    "password": "securepassword",
    "first_name": "John",
    "last_name": "Doe"
  }
  ```

- `POST /api/v1/auth/login` - Login
  ```json
  {
    "email": "user@example.com",
    "password": "securepassword"
  }
  ```

- `GET /api/v1/auth/verify-email?token=TOKEN` - Verify email
- `POST /api/v1/auth/password-reset/request` - Request password reset
  ```json
  {
    "email": "user@example.com"
  }
  ```

- `POST /api/v1/auth/password-reset/confirm` - Reset password
  ```json
  {
    "token": "reset_token",
    "new_password": "newsecurepassword"
  }
  ```

- `POST /api/v1/auth/refresh` - Refresh access token
  ```json
  {
    "refresh_token": "your_refresh_token"
  }
  ```

### Protected Endpoints (Requires Authentication)

#### User Management

- `GET /api/v1/users/profile` - Get current user profile
  - Headers: `Authorization: Bearer <token>`

- `POST /api/v1/users/profile/update` - Update profile
  - Headers: `Authorization: Bearer <token>`
  ```json
  {
    "first_name": "John",
    "last_name": "Doe"
  }
  ```

### Admin Endpoints (Requires Admin Role)

- `GET /api/v1/admin/users/get?id=USER_ID` - Get user by ID
- `POST /api/v1/admin/users/assign-role` - Assign role to user
  ```json
  {
    "user_id": 1,
    "role_id": 2
  }
  ```
- `POST /api/v1/admin/users/remove-role` - Remove role from user

### Health Check

- `GET /health` - Health check endpoint

## Environment Variables

See `.env.example` for all available configuration options.

Key variables:

- `DB_HOST` - Database host
- `DB_PORT` - Database port
- `DB_USER` - Database user
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name
- `JWT_SECRET` - Secret key for JWT tokens
- `SMTP_HOST` - SMTP server host
- `SMTP_PORT` - SMTP server port
- `SMTP_USERNAME` - SMTP username
- `SMTP_PASSWORD` - SMTP password

## Database Schema

The application uses the following main tables:

- `users` - User accounts
- `roles` - User roles (admin, reviewer, user)
- `permissions` - System permissions
- `user_roles` - User-role assignments
- `role_permissions` - Role-permission assignments
- `email_verification_tokens` - Email verification tokens
- `password_reset_tokens` - Password reset tokens
- `sessions` - User sessions
- `audit_logs` - Security audit logs

## Development

### Build

```bash
make build
```

### Run tests

```bash
make test
```

### Run with coverage

```bash
make test-coverage
```

### Format code

```bash
make fmt
```

### Lint

```bash
make lint
```

### Clean build artifacts

```bash
make clean
```

## Security Features

1. **Password Security**: Passwords are hashed using bcrypt with default cost
2. **JWT Tokens**: Secure token-based authentication with configurable expiration
3. **Rate Limiting**: Prevents abuse with configurable request limits
4. **CORS**: Properly configured cross-origin resource sharing
5. **Security Headers**: X-Content-Type-Options, X-XSS-Protection, X-Frame-Options, etc.
6. **Input Validation**: All user inputs are validated and sanitized
7. **Audit Logging**: All security-related actions are logged
8. **Email Verification**: Optional email verification before account activation
9. **Password Recovery**: Secure password reset with time-limited tokens

## Default Roles and Permissions

The system comes with three default roles:

1. **Admin** - Full system access
2. **Reviewer** - Can create and manage reviews
3. **User** - Basic user access

Permissions are grouped by resource and action (e.g., `users.read`, `reviews.create`)

## Project Structure

```
New-Pay/
├── cmd/
│   └── api/           # Application entry point
├── internal/
│   ├── auth/          # Authentication logic
│   ├── config/        # Configuration management
│   ├── database/      # Database connection
│   ├── email/         # Email service
│   ├── handlers/      # HTTP handlers
│   ├── middleware/    # HTTP middleware
│   ├── models/        # Data models
│   ├── repository/    # Data access layer
│   └── service/       # Business logic
├── pkg/
│   ├── logger/        # Logging utilities
│   └── validator/     # Input validation
├── migrations/        # Database migrations
├── docs/              # Documentation
├── .env.example       # Environment variables template
├── docker-compose.yml # Docker configuration
├── Makefile           # Build commands
└── README.md          # This file
```

## OAuth 2.0 Integration

The backend is prepared for OAuth 2.0 integration with Google and Facebook. Configuration is available through environment variables:

- `GOOGLE_CLIENT_ID`
- `GOOGLE_CLIENT_SECRET`
- `GOOGLE_REDIRECT_URL`
- `FACEBOOK_CLIENT_ID`
- `FACEBOOK_CLIENT_SECRET`
- `FACEBOOK_REDIRECT_URL`

OAuth handlers will be implemented in future updates.

## Future Development

- OAuth 2.0 implementation for Google and Facebook
- Swagger/OpenAPI documentation
- Additional business logic for salary estimates
- Review system implementation
- Frontend integration
- Enhanced search and filtering
- Real-time notifications
- Analytics and reporting

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License.

## Support

For support, email support@newpay.com or open an issue in the repository.
