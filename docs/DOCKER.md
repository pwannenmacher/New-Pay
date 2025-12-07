# Docker Build and Run Instructions

## Building the Docker Image

The backend includes a multi-stage Dockerfile that creates a lean production image.

### Build the image:

```bash
docker build -t newpay-backend:latest .
```

The Dockerfile uses:
- **Build stage**: `golang:1.24-alpine` - Downloads dependencies and compiles the binary
- **Final stage**: `alpine:3.18` - Minimal runtime image with only the binary and migrations

Image size: Approximately 20-30MB

### Build arguments (optional):

```bash
# Build for specific platform
docker build --platform linux/amd64 -t newpay-backend:latest .
```

## Running with Docker

### Option 1: Using docker-compose (Recommended)

The easiest way to run the entire stack (database + backend):

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f api

# Stop all services
docker-compose down
```

**Important**: The `docker-compose.yml` reads all configuration from your `.env` file. Make sure to:

1. Copy `.env.example` to `.env`
2. Update the credentials and secrets in `.env`
3. Generate a persistent JWT key: `go run scripts/generate-jwt-keys.go`
4. All database credentials, OAuth secrets, and SMTP settings are loaded from `.env`

The docker-compose configuration:
- PostgreSQL credentials come from `DB_USER`, `DB_PASSWORD`, `DB_NAME` in `.env`
- API configuration is loaded from `.env` with Docker-specific overrides:
  - `DB_HOST` is overridden to `postgres` (container name)
  - `SMTP_HOST` is overridden to `mailpit` (container name)
- No hardcoded credentials in `docker-compose.yml`

### Option 2: Running the container manually

First, ensure PostgreSQL is running, then:

```bash
docker run -d \
  --name newpay-api \
  -p 8080:8080 \
  --env-file .env \
  newpay-backend:latest
```

### Option 3: With environment variables

```bash
docker run -d \
  --name newpay-api \
  -p 8080:8080 \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_USER=newpay \
  -e DB_PASSWORD=newpay_password \
  -e DB_NAME=newpay_db \
  -e JWT_SECRET=your_secret_here \
  newpay-backend:latest
```

## Multi-Stage Build Benefits

1. **Small Image Size**: Only the compiled binary and necessary runtime files
2. **Security**: Runs as non-root user
3. **Fast Builds**: Dependencies are cached in separate layer
4. **No Source Code**: Final image contains only the binary
5. **Health Checks**: Built-in health monitoring

## Image Layers

```
Build Stage (golang:1.24-alpine):
├── Install git, ca-certificates
├── Copy go.mod, go.sum
├── Download dependencies (cached)
├── Copy source code
└── Build static binary

Final Stage (alpine:3.18):
├── Install ca-certificates, tzdata
├── Create non-root user
├── Copy binary from build stage
├── Copy migrations
└── Set user and entrypoint
```

## Health Check

The image includes a built-in health check:
- **Endpoint**: `GET /health`
- **Interval**: 30 seconds
- **Timeout**: 3 seconds
- **Retries**: 3

Check container health:
```bash
docker ps
docker inspect --format='{{.State.Health.Status}}' newpay-api
```

## Best Practices

1. **Use .env file for local development**
   ```bash
   cp .env.example .env
   # Edit .env with your values
   docker run --env-file .env newpay-backend:latest
   ```

2. **Use secrets in production**
   ```bash
   # Docker Swarm
   docker service create \
     --secret jwt_secret \
     --env JWT_SECRET_FILE=/run/secrets/jwt_secret \
     newpay-backend:latest
   
   # Kubernetes
   kubectl create secret generic newpay-secrets \
     --from-literal=JWT_SECRET=your_secret_here
   ```

3. **Run database migrations**
   ```bash
   # One-time setup
   docker-compose exec postgres psql -U newpay -d newpay_db \
     -f /docker-entrypoint-initdb.d/001_init_schema.up.sql
   ```

## Troubleshooting

### Container won't start

```bash
# Check logs
docker logs newpay-api

# Common issues:
# 1. Database not accessible
# 2. Missing JWT_SECRET
# 3. Invalid environment variables
```

### Build fails

```bash
# Clean build (no cache)
docker build --no-cache -t newpay-backend:latest .

# Check Go version
docker run golang:1.24-alpine go version
```

### Connection issues

```bash
# Check if container is running
docker ps -a

# Test health endpoint
curl http://localhost:8080/health

# Check network connectivity
docker network ls
docker network inspect bridge
```

## Production Deployment

For production, consider:

1. **Use specific version tags**
   ```bash
   docker build -t newpay-backend:1.0.0 .
   ```

2. **Scan for vulnerabilities**
   ```bash
   docker scan newpay-backend:latest
   ```

3. **Use multi-platform builds**
   ```bash
   docker buildx build --platform linux/amd64,linux/arm64 \
     -t newpay-backend:latest .
   ```

4. **Push to registry**
   ```bash
   docker tag newpay-backend:latest registry.example.com/newpay-backend:latest
   docker push registry.example.com/newpay-backend:latest
   ```

## Docker Compose Configuration

See `docker-compose.yml` for the full stack configuration including:
- PostgreSQL database
- Automatic migrations
- Network configuration
- Volume persistence
