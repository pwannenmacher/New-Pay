package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/vault"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq"
)

// TestContainers holds references to test containers
type TestContainers struct {
	PostgresContainer *postgres.PostgresContainer
	VaultContainer    *vault.VaultContainer
	DB                *sql.DB
	DBConnString      string
	VaultToken        string
	VaultAddr         string
	JWTSecret         []byte
}

// SetupTestContainers initializes PostgreSQL and Vault containers
func SetupTestContainers(t *testing.T) *TestContainers {
	t.Helper()
	ctx := context.Background()

	// Setup PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:18",
		postgres.WithDatabase("newpay_test"),
		postgres.WithUsername("newpay_test"),
		postgres.WithPassword("newpay_test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Setup Vault container
	vaultContainer, err := vault.Run(ctx,
		"hashicorp/vault:1.15",
		vault.WithToken("test-token"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Vault server started!").
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start Vault container: %v", err)
	}

	// Get Vault address
	vaultAddr, err := vaultContainer.HttpHostAddress(ctx)
	if err != nil {
		t.Fatalf("Failed to get Vault address: %v", err)
	}

	return &TestContainers{
		PostgresContainer: postgresContainer,
		VaultContainer:    vaultContainer,
		DB:                db,
		DBConnString:      connStr,
		VaultToken:        "test-token",
		VaultAddr:         fmt.Sprintf("http://%s", vaultAddr),
		JWTSecret:         []byte("test-secret-key-for-testing-only"),
	}
}

// Cleanup terminates all test containers
func (tc *TestContainers) Cleanup(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	if tc.DB != nil {
		tc.DB.Close()
	}

	if tc.PostgresContainer != nil {
		if err := tc.PostgresContainer.Terminate(ctx); err != nil {
			t.Errorf("Failed to terminate PostgreSQL container: %v", err)
		}
	}

	if tc.VaultContainer != nil {
		if err := tc.VaultContainer.Terminate(ctx); err != nil {
			t.Errorf("Failed to terminate Vault container: %v", err)
		}
	}
}

// runMigrations executes SQL migrations
func runMigrations(db *sql.DB) error {
	// Get migrations directory relative to the project root
	migrationsDir := filepath.Join("..", "..", "migrations")

	// Check if running from test directory
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		migrationsDir = filepath.Join("..", "..", "..", "migrations")
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("failed to find migration files: %w", err)
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
	}

	return nil
}
