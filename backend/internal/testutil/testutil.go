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
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq"
)

// TestMasterKey is a fixed 32-byte AES-256 master key used in tests.
// hex: 0101010101010101010101010101010101010101010101010101010101010101
var TestMasterKey = [32]byte{
	0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
	0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
	0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
	0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
}

// TestContainers holds references to test containers
type TestContainers struct {
	PostgresContainer *postgres.PostgresContainer
	DB                *sql.DB
	DBConnString      string
	JWTSecret         []byte
	// MasterKey is the 32-byte encryption master key to use in tests.
	MasterKey []byte
}

// SetupTestContainers initializes a PostgreSQL container for integration tests.
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

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	if err := runMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	mk := TestMasterKey
	return &TestContainers{
		PostgresContainer: postgresContainer,
		DB:                db,
		DBConnString:      connStr,
		JWTSecret:         []byte("test-secret-key-for-testing-only"),
		MasterKey:         mk[:],
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
}

// runMigrations executes SQL migrations
func runMigrations(db *sql.DB) error {
	migrationsDir := filepath.Join("..", "..", "migrations")
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

// GetMasterKeyBytes returns a copy of the test master key as a slice.
func GetMasterKeyBytes() []byte {
	mk := TestMasterKey
	return mk[:]
}
