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
- **Frontend**: React 19 + TypeScript 5.9
- **UI Library**: Mantine 7
- **Database**: PostgreSQL 16+
- **Authentication**: JWT
- **Email**: SMTP

## Prerequisites

- Go 1.24 or higher
- Node.js 20 or higher
- PostgreSQL 16 or higher
- SMTP server credentials (for email functionality)

## Quick Start

### Option 1: Using Docker (Recommended)

The fastest way to get started is using Docker Compose, which runs both the database and API:

```bash
# Clone the repository
git clone https://github.com/pwannenmacher/New-Pay.git
cd New-Pay

# Start all services (PostgreSQL + API)
docker-compose up -d

# Check the logs
docker-compose logs -f api

# The API will be available at http://localhost:8080
```

See [Docker Documentation](docs/DOCKER.md) for more details.

### Option 2: Local Development

#### 1. Clone the repository

```bash
git clone https://github.com/pwannenmacher/New-Pay.git
cd New-Pay
```

#### 2. Set up environment variables

```bash
cp .env.example .env
```

Edit `.env` and configure your database and email settings.

#### 3. Start PostgreSQL

```bash
docker-compose up -d postgres
```

Or use your own PostgreSQL instance.

#### 4. Install Go dependencies

```bash
go mod download
```

#### 5. Run the application

```bash
go run cmd/api/*.go
```

The application will automatically run database migrations on startup.

#### 6. Access the API

- API: `http://localhost:8080`
- Health Check: `http://localhost:8080/health`
- **Swagger Documentation**: `http://localhost:8080/swagger/index.html`

#### 7. Start the Frontend (Optional)

In a new terminal:

```bash
cd frontend
npm install
npm run dev
```

- Frontend: `http://localhost:5173`

## Additional Documentation

- **[OAuth/SSO Configuration](docs/OAUTH_CONFIGURATION.md)** - How to configure OAuth providers (Google, Microsoft, Keycloak, etc.)
- **[Session Management](docs/SESSION_MANAGEMENT.md)** - User and admin session management
- **[API Documentation](docs/API.md)** - Detailed API endpoint documentation
- **[Development Guide](docs/DEVELOPMENT.md)** - Development workflow and best practices
- **[Docker Guide](docs/DOCKER.md)** - Docker deployment and configuration
- **[JWT Security](docs/JWT_SECURITY.md)** - JWT implementation and security
- **[JWT Key Management](docs/JWT_KEY_MANAGEMENT.md)** - Managing persistent JWT keys to prevent session invalidation
- **[Migrations](docs/MIGRATIONS.md)** - Database migration guide

## API Endpoints

For comprehensive API documentation with request/response examples, visit the Swagger UI at `http://localhost:8080/swagger/index.html` when the server is running.

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
go build -o bin/api cmd/api/*.go
```

### Run tests

```bash
go test ./...
```

### Run with coverage

```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Format code

```bash
go fmt ./...
```

### Vet code

```bash
go vet ./...
```

### Using Docker Compose for Development

```bash
# Start all services in detached mode
docker-compose up -d

# View logs
docker-compose logs -f api

# Stop all services
docker-compose down

# Rebuild and restart
docker-compose up -d --build
```


## Security Features

1. **Password Security**: Passwords are hashed using bcrypt with default cost
2. **JWT Tokens**: Secure token-based authentication with configurable expiration
3. **Session Management**: JWT sessions can be invalidated (logout from all devices, password change)
4. **Rate Limiting**: Prevents abuse with configurable request limits
5. **CORS**: Properly configured cross-origin resource sharing
6. **Security Headers**: X-Content-Type-Options, X-XSS-Protection, X-Frame-Options, etc.
7. **Input Validation**: All user inputs are validated and sanitized
8. **Audit Logging**: All security-related actions are logged
9. **Email Verification**: Optional email verification before account activation
10. **Password Recovery**: Secure password reset with time-limited tokens

## API Documentation

The API is fully documented using Swagger/OpenAPI. Once the server is running, you can access:

- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **Swagger JSON**: `http://localhost:8080/swagger/doc.json`

The Swagger documentation includes:
- All available endpoints
- Request/response schemas
- Authentication requirements
- Example requests and responses
- Try-it-out functionality

To regenerate Swagger documentation after changes:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/api/main.go -o docs
```

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
│   └── api/              # Application entry point
├── internal/
│   ├── auth/             # Authentication logic
│   ├── config/           # Configuration management
│   ├── database/         # Database connection & migrations
│   ├── email/            # Email service
│   ├── handlers/         # HTTP handlers
│   ├── middleware/       # HTTP middleware
│   ├── models/           # Data models
│   ├── repository/       # Data access layer
│   └── service/          # Business logic
├── pkg/
│   ├── logger/           # Logging utilities
│   └── validator/        # Input validation
├── frontend/             # React frontend application
│   ├── src/
│   │   ├── components/   # React components
│   │   ├── pages/        # Page components
│   │   ├── contexts/     # React contexts
│   │   ├── services/     # API services
│   │   └── types/        # TypeScript types
│   └── public/           # Static assets
├── migrations/           # Database migrations
├── docs/                 # Documentation & Swagger files
├── .env.example          # Environment variables template
├── docker-compose.yml    # Docker configuration
├── Dockerfile            # Multi-stage Docker build
└── README.md             # This file
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

- OAuth 2.0 callback handler implementation
- Business logic for salary estimates
- Review system implementation
- Real-time notifications
- Analytics and reporting
- Enhanced search and filtering

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
