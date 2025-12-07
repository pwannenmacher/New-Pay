package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Migration represents a database migration
type Migration struct {
	Version string
	Name    string
	UpSQL   string
	DownSQL string
}

// MigrationExecutor handles database migrations
type MigrationExecutor struct {
	db *sql.DB
}

// NewMigrationExecutor creates a new migration executor
func NewMigrationExecutor(db *sql.DB) *MigrationExecutor {
	return &MigrationExecutor{db: db}
}

// RunMigrations executes all pending migrations from the migrations directory
func (m *MigrationExecutor) RunMigrations(migrationsPath string) error {
	// Create migrations tracking table if it doesn't exist
	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Read migration files
	migrations, err := m.readMigrationFiles(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Execute pending migrations
	for _, migration := range migrations {
		if !contains(applied, migration.Version) {
			if err := m.executeMigration(migration); err != nil {
				return fmt.Errorf("failed to execute migration %s: %w", migration.Version, err)
			}
			fmt.Printf("Applied migration: %s - %s\n", migration.Version, migration.Name)
		}
	}

	return nil
}

// createMigrationsTable creates the migrations tracking table
func (m *MigrationExecutor) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := m.db.Exec(query)
	return err
}

// readMigrationFiles reads all migration files from the directory
func (m *MigrationExecutor) readMigrationFiles(migrationsPath string) ([]Migration, error) {
	files, err := os.ReadDir(migrationsPath)
	if err != nil {
		return nil, err
	}

	migrationsMap := make(map[string]*Migration)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		if !strings.HasSuffix(filename, ".sql") {
			continue
		}

		// Extract version and type (up/down)
		parts := strings.Split(filename, "_")
		if len(parts) < 2 {
			continue
		}

		version := parts[0]
		isUp := strings.HasSuffix(filename, ".up.sql")
		isDown := strings.HasSuffix(filename, ".down.sql")

		if !isUp && !isDown {
			continue
		}

		// Read file content
		content, err := os.ReadFile(filepath.Join(migrationsPath, filename))
		if err != nil {
			return nil, err
		}

		// Get or create migration
		if migrationsMap[version] == nil {
			name := strings.TrimSuffix(strings.TrimSuffix(filename, ".up.sql"), ".down.sql")
			name = strings.Join(parts[1:], "_")
			name = strings.TrimSuffix(name, ".up")
			name = strings.TrimSuffix(name, ".down")

			migrationsMap[version] = &Migration{
				Version: version,
				Name:    name,
			}
		}

		if isUp {
			migrationsMap[version].UpSQL = string(content)
		} else {
			migrationsMap[version].DownSQL = string(content)
		}
	}

	// Convert map to sorted slice
	var migrations []Migration
	for _, migration := range migrationsMap {
		if migration.UpSQL != "" {
			migrations = append(migrations, *migration)
		}
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// getAppliedMigrations returns list of applied migration versions
func (m *MigrationExecutor) getAppliedMigrations() ([]string, error) {
	query := `SELECT version FROM schema_migrations ORDER BY version`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []string
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}

	return versions, nil
}

// executeMigration executes a single migration
func (m *MigrationExecutor) executeMigration(migration Migration) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.Exec(migration.UpSQL); err != nil {
		return fmt.Errorf("migration SQL failed: %w", err)
	}

	// Record migration
	query := `INSERT INTO schema_migrations (version) VALUES ($1)`
	if _, err := tx.Exec(query, migration.Version); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
