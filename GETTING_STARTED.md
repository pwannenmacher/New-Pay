# New Pay - Getting Started Guide

This guide will help you set up and run the complete New Pay application with both backend and frontend.

## System Requirements

- Go 1.24+
- Node.js 20+
- PostgreSQL 16+
- npm 10+

## Quick Start

### 1. Start the Backend

#### Option A: Using Docker (Recommended)

```bash
# Start PostgreSQL and API
docker-compose up -d

# Check logs
docker-compose logs -f api
```

The API will be available at `http://localhost:8080`

#### Option B: Local Development

```bash
# Set up environment
cp .env.example .env
# Edit .env with your database credentials

# Start PostgreSQL (if not using Docker)
docker-compose up -d postgres

# Run the backend
go run cmd/api/*.go
```

### 2. Start the Frontend

```bash
# Navigate to frontend directory
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev
```

The frontend will be available at `http://localhost:5173`

## Default Admin Setup

After the application starts, you'll need to create an admin user:

1. Register a new user through the frontend or API
2. Connect to PostgreSQL and manually assign the admin role:

```sql
-- Find your user ID
SELECT id, email FROM users WHERE email = 'your@email.com';

-- Find the admin role ID
SELECT id, name FROM roles WHERE name = 'admin';

-- Assign admin role to user
INSERT INTO user_roles (user_id, role_id, created_at)
VALUES (YOUR_USER_ID, ADMIN_ROLE_ID, NOW());
```

## Using the Application

### For Regular Users

1. **Register**: Go to `http://localhost:5173/register`
   - Fill in your email, password, first name, and last name
   - You'll receive a verification email (if SMTP is configured)

2. **Login**: Go to `http://localhost:5173/login`
   - Enter your email and password
   - Or use OAuth (Google/Facebook) if configured

3. **Profile**: Access your profile at `http://localhost:5173/profile`
   - View your account information
   - Edit your name

### For Admin Users

After logging in as an admin, you'll see additional menu items:

1. **Admin Dashboard**: `http://localhost:5173/admin`
   - Overview of admin features

2. **User Management**: `http://localhost:5173/admin/users`
   - View all users
   - Assign/remove roles
   - View user details

3. **Audit Logs**: `http://localhost:5173/admin/audit-logs`
   - View all security-related actions
   - Track user activities
   - Monitor system events

## API Documentation

The backend API is fully documented with Swagger:

- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **API Base URL**: `http://localhost:8080/api/v1`

### Key Endpoints

#### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/refresh` - Refresh access token
- `GET /api/v1/auth/verify-email` - Verify email
- `POST /api/v1/auth/password-reset/request` - Request password reset
- `POST /api/v1/auth/password-reset/confirm` - Confirm password reset

#### User Profile
- `GET /api/v1/users/profile` - Get current user profile
- `POST /api/v1/users/profile/update` - Update profile

#### Admin (Admin Role Required)
- `GET /api/v1/admin/users/list` - List all users
- `GET /api/v1/admin/users/get?id=USER_ID` - Get specific user
- `POST /api/v1/admin/users/assign-role` - Assign role to user
- `POST /api/v1/admin/users/remove-role` - Remove role from user
- `GET /api/v1/admin/roles/list` - List all roles
- `GET /api/v1/admin/audit-logs/list` - List audit logs

## Features

### Authentication & Security
✅ JWT-based authentication with automatic refresh
✅ Email verification
✅ Password reset flow
✅ OAuth 2.0 support (Google, Facebook)
✅ Rate limiting
✅ CORS protection
✅ Security headers
✅ Audit logging

### User Management
✅ User registration and login
✅ Profile viewing and editing
✅ Role-based access control (RBAC)
✅ User roles: admin, reviewer, user
✅ Granular permissions system

### Admin Dashboard
✅ User management interface
✅ Role assignment
✅ Audit log viewer
✅ System monitoring

### Frontend
✅ Responsive design (mobile & desktop)
✅ Dark/light mode support
✅ Modern UI with Mantine components
✅ Form validation
✅ Real-time error handling
✅ Loading states

## Configuration

### Backend (.env)

```env
# Server
SERVER_HOST=localhost
SERVER_PORT=8080

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=newpay
DB_PASSWORD=your_password
DB_NAME=newpay_db

# JWT
JWT_SECRET=your_super_secret_key
JWT_EXPIRATION=24h
JWT_REFRESH_EXPIRATION=168h

# SMTP (for emails)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your_email@gmail.com
SMTP_PASSWORD=your_password

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173

# OAuth (optional)
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
FACEBOOK_CLIENT_ID=your_facebook_client_id
FACEBOOK_CLIENT_SECRET=your_facebook_client_secret
```

### Frontend (.env)

```env
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

## Development

### Backend Development

```bash
# Run tests
go test ./...

# Build
go build -o bin/api cmd/api/*.go

# Format code
go fmt ./...

# Vet code
go vet ./...

# Update Swagger docs
swag init -g cmd/api/main.go -o docs
```

### Frontend Development

```bash
cd frontend

# Development server with hot reload
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Lint code
npm run lint
```

## Troubleshooting

### Backend Issues

**Database Connection Failed**
- Check PostgreSQL is running: `docker-compose ps`
- Verify database credentials in `.env`
- Check database exists: `psql -U newpay -d newpay_db`

**Port Already in Use**
- Check if port 8080 is free: `lsof -i :8080`
- Change `SERVER_PORT` in `.env`

### Frontend Issues

**API Connection Failed**
- Verify backend is running on port 8080
- Check `VITE_API_BASE_URL` in `frontend/.env`
- Check CORS settings in backend `.env`

**Build Errors**
- Clear cache: `rm -rf node_modules/.vite`
- Reinstall dependencies: `rm -rf node_modules && npm install`

## Next Steps

### Planned Features

1. **OAuth Callback Handlers** (Backend)
   - Implement Google OAuth callback
   - Implement Facebook OAuth callback

2. **Business Logic**
   - Salary estimation algorithms
   - Peer review system
   - Rating and feedback

3. **Advanced Features**
   - Real-time notifications
   - Analytics dashboard
   - Search and filtering
   - Export functionality

4. **Improvements**
   - Dynamic pagination (backend returns total counts)
   - Advanced user search
   - Bulk user operations
   - Role and permission management UI

## Support

For issues or questions:
- Check the Swagger documentation: `http://localhost:8080/swagger/index.html`
- Review the code comments
- Check the logs: `docker-compose logs -f api`

## License

MIT License
