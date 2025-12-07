package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/pwannenmacher/New-Pay/internal/config"
)

// Database wraps the SQL database connection
type Database struct {
	DB *sql.DB
}

// New creates a new database connection
func New(cfg *config.DatabaseConfig) (*Database, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Name,
		cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{DB: db}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.DB.Close()
}

// Ping checks if the database connection is alive
func (d *Database) Ping() error {
	return d.DB.Ping()
}

// RunMigrations runs database migrations from the migrations directory
func (d *Database) RunMigrations(migrationsPath string) error {
	// Read and execute migration file
	// For now, this is a placeholder - you would typically use a migration library
	// like golang-migrate/migrate
	return nil
}

// HealthCheck performs a health check on the database
func (d *Database) HealthCheck() error {
	ctx, cancel := getContext(5 * time.Second)
	defer cancel()

	if err := d.DB.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}
