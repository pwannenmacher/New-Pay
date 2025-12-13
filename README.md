# New Pay

Platform for salary estimates and peer reviews.

## Stack

- Backend: Go 1.25
- Frontend: React 19, TypeScript 5.9, Mantine 8
- Database: PostgreSQL 17
- Auth: JWT (ES256)
- Email: SMTP

## Features

### Authentication

- JWT authentication with persistent ECDSA keys
- Password hashing (bcrypt)
- Email verification
- Password recovery
- Session management
- Rate limiting
- OAuth 2.0 (Google, Microsoft, Keycloak, custom providers)
- **OAuth Group to Role Mapping**: Automatic role synchronization based on OAuth provider groups

### User Management

- Registration and login
- Profile management
- Role-based access control (admin, reviewer, user)
- **Independent role system** without hierarchy
- Permissions system
- **Automatic role updates** from OAuth groups at each login
- **Default role assignment** for OAuth users without group mappings

### Security

- CORS configuration
- Security headers
- Input validation
- Audit logging

## Quick Start

### Docker

```bash
docker compose up -d
```

API: <http://localhost:8080>
Frontend: <http://localhost:3001>
Swagger: <http://localhost:8080/swagger/index.html>

### Local Development

```bash
cp .env.example .env
# Edit .env with your settings

docker compose up -d postgres
cd backend && go run cmd/api/*.go
```

Frontend:

```bash
cd frontend && npm install && npm run dev
```

## Configuration

See `.env.example` for all options.

Required:

- `DB_*` - Database connection
- `JWT_SECRET` - JWT signing key (auto-generated if missing)
- `SMTP_*` - Email server (optional)

## Documentation

- `docs/ROLE_BASED_ACCESS.md` - **Role hierarchy and access control**
- `docs/OAUTH_GROUP_MAPPING.md` - **OAuth group to role mapping**
- `docs/CATALOG_SYSTEM.md` - Catalog system and role-based visibility
- `docs/ENCRYPTION.md` - Encryption system architecture
- `docs/OAUTH_CONFIGURATION.md` - OAuth setup
- `docs/SESSION_MANAGEMENT.md` - Session management
- `docs/JWT_KEY_MANAGEMENT.md` - JWT key handling
- `docs/DOCKER.md` - Docker deployment

## Project Structure

```plain
new-pay-gh/
├── backend/
│   ├── cmd/api/         # Entry point
│   ├── internal/        # Application code
│   ├── migrations/      # Database migrations
│   └── docs/            # Swagger/API docs
├── frontend/
│   └── src/             # React application
└── docs/                # Documentation
```

## Default Credentials

First registered user becomes admin automatically.

## License

MIT
