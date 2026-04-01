# New Pay - Backend

New Pay is a platform for salary estimation and peer review, designed to enable fair and transparent salary processes. The backend is written in Go and uses PostgreSQL for data storage and local AES-256-GCM encryption for sensitive data.

## Features

*   **User Management**: Authentication via JWT (Access & Refresh Tokens), Role-Based Access Control (RBAC).
*   **Criteria Catalogs**: Definition of competence matrices with categories, paths, and levels.
*   **Self-Assessment**: Users can assess themselves based on the catalogs.
*   **Peer Review**: Colleagues assess the user's competencies.
*   **Consolidation**: Merging assessments with the ability to discuss and override discrepancies.
*   **Discussion & Conclusion**: Process steps for discussing results and final determination.
*   **Security**: End-to-end encryption of sensitive comments and justifications using AES-256-GCM with a local master key.

## Tech Stack

*   **Language**: Go (Golang) 1.25
*   **Database**: PostgreSQL
*   **Security**: AES-256-GCM encryption with Ed25519-signed hash-chain audit trail
*   **Authentication**: JWT (JSON Web Tokens) with ECDSA Signature
*   **Documentation**: Swagger / OpenAPI

## Prerequisites

*   Go 1.25 or higher
*   Docker (for database container)
*   PostgreSQL 14+

## Installation & Setup

1.  **Clone Repository**

2.  **Install Dependencies**
    ```bash
    go mod download
    ```

3.  **Configuration**
    The application is configured via environment variables. Copy `.env.template` to `.env` and fill in the required values, including a generated `ENCRYPTION_MASTER_KEY`:
    ```bash
    openssl rand -hex 32   # → paste result as ENCRYPTION_MASTER_KEY
    ```

4.  **Start Application**
    ```bash
    go run main.go
    ```
    Database migrations are executed automatically on startup.

## Tests

Integration tests use `testcontainers-go` to automatically start a temporary Docker container for PostgreSQL.

```bash
# Run all tests
go test ./internal/handlers/... -v
```

For more information on tests, see `TEST_QUICKSTART.md`.

## API Documentation

API documentation is generated via Swagger.
After starting the application, the Swagger UI is available at the following path (default):
`http://localhost:8080/docs/index.html`

## Security

Special attention is paid to data security:
*   **SecureStore**: Sensitive texts (e.g., justifications in reviews) are encrypted with AES-256-GCM and stored in `encrypted_records`.
*   **Key Hierarchy**: A master key from `ENCRYPTION_MASTER_KEY` protects per-user Ed25519 keys and per-process symmetric keys in the database.
*   **Audit Logs**: Important actions are logged with a tamper-evident hash chain.

## Upgrading dependencies

```bash
cd backend

# Only patch and minor updates
go get -u=patch
# Or for all updates:
go get -u ./...

go mod tidy
```

## License

MIT
