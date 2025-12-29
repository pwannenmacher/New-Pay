# New Pay - Backend

New Pay is a platform for salary estimation and peer review, designed to enable fair and transparent salary processes. The backend is written in Go and uses PostgreSQL for data storage and HashiCorp Vault for encrypting sensitive data.

## Features

*   **User Management**: Authentication via JWT (Access & Refresh Tokens), Role-Based Access Control (RBAC).
*   **Criteria Catalogs**: Definition of competence matrices with categories, paths, and levels.
*   **Self-Assessment**: Users can assess themselves based on the catalogs.
*   **Peer Review**: Colleagues assess the user's competencies.
*   **Consolidation**: Merging assessments with the ability to discuss and override discrepancies.
*   **Discussion & Conclusion**: Process steps for discussing results and final determination.
*   **Security**: End-to-end encryption of sensitive comments and justifications using HashiCorp Vault (Transit Engine).

## Tech Stack

*   **Language**: Go (Golang) 1.25
*   **Database**: PostgreSQL
*   **Security**: HashiCorp Vault (Transit Secrets Engine)
*   **Authentication**: JWT (JSON Web Tokens) with ECDSA Signature
*   **Documentation**: Swagger / OpenAPI

## Prerequisites

*   Go 1.25 or higher
*   Docker (for database and Vault containers)
*   PostgreSQL 14+
*   HashiCorp Vault

## Installation & Setup

1.  **Clone Repository**

2.  **Install Dependencies**
    ```bash
    go mod download
    ```

3.  **Configuration**
    The application is configured via environment variables. Ensure a database and Vault server are available.

4.  **Start Application**
    ```bash
    go run main.go
    ```
    Database migrations are executed automatically on startup.

## Tests

Integration tests use `testcontainers-go` to automatically start temporary Docker containers for PostgreSQL and Vault.

```bash
# Run all tests
go test ./...
```

For more information on tests, see `TEST_QUICKSTART.md`.

## API Documentation

API documentation is generated via Swagger.
After starting the application, the Swagger UI is available at the following path (default):
`http://localhost:8080/docs/index.html`

## Security

Special attention is paid to data security:
*   **SecureStore**: Sensitive texts (e.g., justifications in reviews) are stored encrypted.
*   **Vault Integration**: Key management and cryptographic operations are offloaded to HashiCorp Vault.
*   **Audit Logs**: Important actions are logged.
