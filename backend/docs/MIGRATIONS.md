# Database Migration Guide

## Overview

The New Pay backend includes an automatic database migration system that executes SQL migrations on startup. This eliminates the need for manual migration execution and ensures the database schema is always up-to-date.

## How It Works

### Automatic Execution

When the backend starts:

1. Connects to PostgreSQL
2. Creates `schema_migrations` tracking table (if not exists)
3. Reads all `.sql` files from `./migrations` directory
4. Compares with applied migrations in tracking table
5. Executes pending migrations in version order
6. Records successful migrations in tracking table

### Migration Tracking

Applied migrations are tracked in the `schema_migrations` table:

```sql
CREATE TABLE schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Migration File Format

### Naming Convention

Migration files must follow this pattern:

```
{version}_{name}.{direction}.sql
```

Examples:
- `001_init_schema.up.sql` - Initial schema (up migration)
- `001_init_schema.down.sql` - Rollback for initial schema
- `002_add_sessions_table.up.sql` - Add sessions table
- `002_add_sessions_table.down.sql` - Remove sessions table

### Directory Structure

```
migrations/
├── 001_init_schema.up.sql
├── 001_init_schema.down.sql
├── 002_add_feature.up.sql
└── 002_add_feature.down.sql
```

## Creating New Migrations

### Step 1: Create Migration Files

Create both up and down migration files with the next version number:

```bash
cd migrations

# Create up migration
cat > 003_add_reviews_table.up.sql << 'EOF'
CREATE TABLE reviews (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    company_name VARCHAR(255) NOT NULL,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_reviews_user_id ON reviews(user_id);
CREATE INDEX idx_reviews_company ON reviews(company_name);
EOF

# Create down migration
cat > 003_add_reviews_table.down.sql << 'EOF'
DROP TABLE IF EXISTS reviews;
EOF
```

### Step 2: Test Locally

```bash
# Start the application
go run cmd/api/*.go

# The migration will run automatically
# Check logs for "Applied migration: 003"
```

### Step 3: Verify

```bash
# Connect to database
docker-compose exec postgres psql -U newpay -d newpay_db

# Check applied migrations
SELECT * FROM schema_migrations ORDER BY version;

# Verify table exists
\dt reviews
```

## Migration Best Practices

### 1. Always Create Both Directions

Create both `.up.sql` and `.down.sql` files for every migration.

### 2. Make Migrations Idempotent

Use `IF NOT EXISTS` and `IF EXISTS` clauses:

```sql
-- Good
CREATE TABLE IF NOT EXISTS users (...);

-- Bad
CREATE TABLE users (...);
```

### 3. Use Transactions

Each migration runs in a transaction automatically. If it fails, changes are rolled back.

### 4. Handle Data Carefully

When modifying existing tables with data:

```sql
-- Add column with default
ALTER TABLE users ADD COLUMN middle_name VARCHAR(100) DEFAULT '';

-- Then remove default if not needed
ALTER TABLE users ALTER COLUMN middle_name DROP DEFAULT;
```

### 5. Test Migrations

Always test migrations on a copy of production data before applying to production.

### 6. Keep Migrations Small

Create focused migrations that do one thing well.

## Troubleshooting

### Migration Failed

If a migration fails:

1. Check application logs for error details
2. Fix the SQL in the migration file
3. Remove the failed version from `schema_migrations`:
   ```sql
   DELETE FROM schema_migrations WHERE version = '003';
   ```
4. Restart the application to retry

### Skip a Migration

To mark a migration as applied without running it:

```sql
INSERT INTO schema_migrations (version) VALUES ('003');
```

### Re-run a Migration

To re-run a migration:

```sql
-- Remove from tracking
DELETE FROM schema_migrations WHERE version = '003';

-- Manually run down migration if needed
\i migrations/003_add_feature.down.sql

-- Restart application to re-apply
```

## Docker Compose Integration

### Previous Approach (Removed)

Previously, migrations were mounted as a volume to PostgreSQL:

```yaml
volumes:
  - ./migrations:/docker-entrypoint-initdb.d  # OLD - Don't use
```

This only worked on first container creation and couldn't handle updates.

### Current Approach

Migrations are now executed by the backend application:

```yaml
api:
  # No migration volume needed
  # Migrations are bundled with the application
  # Executed automatically on startup
```

Benefits:
- ✅ Runs on every startup
- ✅ Handles migration updates
- ✅ Version tracking
- ✅ Transaction safety
- ✅ Better error handling

## Production Deployment

### Strategy 1: Blue-Green Deployment

1. Deploy new version with migrations
2. New version applies migrations automatically
3. Switch traffic to new version
4. Keep old version running briefly for rollback

### Strategy 2: Separate Migration Step

1. Run migrations in separate job:
   ```bash
   # Run just migrations
   go run cmd/migrate/main.go up
   ```
2. Deploy application
3. Application skips already-applied migrations

### Strategy 3: Rolling Update

1. Ensure migrations are backward compatible
2. Deploy new application version
3. Each instance applies migrations (idempotent)
4. First instance applies, others skip

## Monitoring

Monitor migration execution:

```bash
# Application logs
docker-compose logs -f api | grep migration

# Database tracking
docker-compose exec postgres psql -U newpay -d newpay_db \
  -c "SELECT * FROM schema_migrations ORDER BY applied_at DESC LIMIT 10;"
```

## Current Migrations

- `001_init_schema` - Initial database schema
  - Users, roles, permissions
  - User-role and role-permission mappings
  - Email verification and password reset tokens
  - Sessions table
  - Audit logs
  - Default roles and permissions

## Migration Workflow

```
Developer creates migration files
          ↓
Commit to repository
          ↓
Deploy application
          ↓
Application starts
          ↓
Migration executor runs
          ↓
Pending migrations applied
          ↓
Application ready
```
