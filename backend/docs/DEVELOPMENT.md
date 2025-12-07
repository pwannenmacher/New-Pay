# Development Setup Guide

This guide will help you set up the New Pay backend for local development.

## Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.24+**: [Download Go](https://go.dev/dl/)
- **PostgreSQL 16+**: [Download PostgreSQL](https://www.postgresql.org/download/)
- **Docker & Docker Compose** (optional): [Download Docker](https://www.docker.com/products/docker-desktop/)
- **Git**: [Download Git](https://git-scm.com/downloads)

## Step-by-Step Setup

### 1. Clone the Repository

```bash
git clone https://github.com/pwannenmacher/New-Pay.git
cd New-Pay
```

### 2. Install Go Dependencies

```bash
go mod download
```

### 3. Set Up PostgreSQL

#### Option A: Using Docker (Recommended)

The easiest way to run PostgreSQL is using Docker Compose:

```bash
docker-compose up -d
```

This will start PostgreSQL on port 5432 with the following credentials:
- **User**: newpay
- **Password**: newpay_password
- **Database**: newpay_db

#### Option B: Using Local PostgreSQL

If you have PostgreSQL installed locally:

1. Create a database:
```bash
createdb newpay_db
```

2. Create a user:
```bash
psql -c "CREATE USER newpay WITH PASSWORD 'newpay_password';"
psql -c "GRANT ALL PRIVILEGES ON DATABASE newpay_db TO newpay;"
```

### 4. Configure Environment Variables

Copy the example environment file:

```bash
cp .env.example .env
```

Edit `.env` and configure your settings. For local development, you can use the defaults, but make sure to update:

```env
# Database (if not using Docker defaults)
DB_HOST=localhost
DB_PORT=5432
DB_USER=newpay
DB_PASSWORD=newpay_password
DB_NAME=newpay_db

# JWT Secret (IMPORTANT: Change this!)
JWT_SECRET=your-super-secret-jwt-key-change-this

# Email (optional for development, but required for email features)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your_email@gmail.com
SMTP_PASSWORD=your_app_specific_password
SMTP_FROM=noreply@newpay.com
```

### 5. Run Database Migrations

Apply the database schema:

```bash
# If using Docker
docker-compose exec postgres psql -U newpay -d newpay_db -f /docker-entrypoint-initdb.d/001_init_schema.up.sql

# If using local PostgreSQL
psql -U newpay -d newpay_db -f migrations/001_init_schema.up.sql
```

Or use the Makefile:

```bash
DB_USER=newpay DB_NAME=newpay_db make migrate-up
```

### 6. Build and Run the Application

Build the application:

```bash
make build
```

Run the application:

```bash
make run
```

Or run directly:

```bash
go run cmd/api/*.go
```

The server will start on `http://localhost:8080`

### 7. Verify the Setup

Check if the server is running:

```bash
curl http://localhost:8080/health
```

You should see:
```json
{"status":"healthy","version":"1.0.0"}
```

## Development Workflow

### Running Tests

Run all tests:

```bash
make test
```

Run tests with coverage:

```bash
make test-coverage
```

This will generate a `coverage.html` file you can open in your browser.

### Code Formatting

Format your code:

```bash
make fmt
```

### Linting

Run the linter:

```bash
make lint
```

Note: You may need to install golangci-lint first:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Code Vetting

Run go vet:

```bash
make vet
```

### Clean Build Artifacts

```bash
make clean
```

## Testing the API

### 1. Register a User

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "first_name": "Test",
    "last_name": "User"
  }'
```

### 2. Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

Save the `access_token` from the response.

### 3. Get Profile

```bash
curl -X GET http://localhost:8080/api/v1/users/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## Creating an Admin User

To create an admin user for testing:

1. Register a normal user
2. Connect to the database:
```bash
# Using Docker
docker-compose exec postgres psql -U newpay -d newpay_db

# Using local PostgreSQL
psql -U newpay -d newpay_db
```

3. Assign the admin role:
```sql
INSERT INTO user_roles (user_id, role_id, created_at)
SELECT u.id, r.id, NOW()
FROM users u, roles r
WHERE u.email = 'test@example.com' AND r.name = 'admin';
```

## Email Configuration

For email verification and password reset to work, you need to configure SMTP.

### Using Gmail

1. Enable 2-factor authentication on your Google account
2. Generate an App Password: https://myaccount.google.com/apppasswords
3. Update `.env`:

```env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your_email@gmail.com
SMTP_PASSWORD=your_app_specific_password
SMTP_FROM=noreply@newpay.com
```

### Using Mailgun, SendGrid, or other services

Update the SMTP settings according to your email service provider's documentation.

### Development: Disable Email

For development without email, you can:
1. Check the console logs - email content will be printed (if implemented)
2. Directly mark emails as verified in the database:

```sql
UPDATE users SET email_verified = true WHERE email = 'test@example.com';
```

## Database Management

### View Database Tables

```bash
# Using Docker
docker-compose exec postgres psql -U newpay -d newpay_db -c "\dt"

# Using local PostgreSQL
psql -U newpay -d newpay_db -c "\dt"
```

### View Users

```sql
SELECT id, email, first_name, last_name, email_verified, is_active, created_at 
FROM users;
```

### View User Roles

```sql
SELECT u.email, r.name as role
FROM users u
JOIN user_roles ur ON u.id = ur.user_id
JOIN roles r ON ur.role_id = r.id;
```

### Reset Database

```bash
# Run down migration
DB_USER=newpay DB_NAME=newpay_db make migrate-down

# Run up migration
DB_USER=newpay DB_NAME=newpay_db make migrate-up
```

## Troubleshooting

### Database Connection Issues

**Error**: `failed to connect to database`

**Solution**:
1. Ensure PostgreSQL is running:
   ```bash
   docker-compose ps  # If using Docker
   # or
   pg_isready -U newpay
   ```

2. Check your database credentials in `.env`

3. Verify the database exists:
   ```bash
   psql -U newpay -l
   ```

### Port Already in Use

**Error**: `address already in use`

**Solution**:
1. Check what's using port 8080:
   ```bash
   lsof -i :8080  # macOS/Linux
   ```

2. Either stop that process or change the port in `.env`:
   ```env
   SERVER_PORT=8081
   ```

### JWT Token Invalid

**Error**: `invalid or expired token`

**Solution**:
1. Make sure `JWT_SECRET` is set in `.env`
2. Token may have expired - login again to get a new token
3. Ensure you're including the token in the Authorization header:
   ```
   Authorization: Bearer YOUR_TOKEN_HERE
   ```

## IDE Setup

### Visual Studio Code

Recommended extensions:
- Go (by Go Team at Google)
- REST Client (for testing API endpoints)
- PostgreSQL (for database management)

Create `.vscode/settings.json`:

```json
{
  "go.testFlags": ["-v"],
  "go.lintTool": "golangci-lint",
  "go.formatTool": "gofmt"
}
```

### GoLand

GoLand comes with excellent Go support out of the box. Just open the project directory.

## Next Steps

1. Read the [API Documentation](API.md)
2. Explore the codebase structure
3. Run the tests to understand the functionality
4. Start building your features!

## Getting Help

- Check the [main README](../README.md)
- Review the [API documentation](API.md)
- Open an issue on GitHub
- Contact the development team

Happy coding! 🚀
