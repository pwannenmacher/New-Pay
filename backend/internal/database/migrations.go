package database

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Migration represents a database migration
type Migration struct {
	Version  string
	Name     string
	Title    string // Human-readable title derived from filename
	UpSQL    string
	DownSQL  string
	Checksum string // SHA256 checksum of UpSQL content
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

	// Validate checksums of already applied migrations
	if err := m.validateMigrationChecksums(migrations); err != nil {
		return fmt.Errorf("migration validation failed: %w", err)
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
			slog.Info("Applied migration", "version", migration.Version, "title", migration.Title)
		}
	}

	return nil
}

// createMigrationsTable creates the migrations tracking table
func (m *MigrationExecutor) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			title VARCHAR(500),
			checksum VARCHAR(64),
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

			// Create human-readable title from filename
			title := strings.Join(parts[1:], " ")
			title = strings.TrimSuffix(title, ".up.sql")
			title = strings.TrimSuffix(title, ".down.sql")
			title = strings.ReplaceAll(title, "_", " ")

			migrationsMap[version] = &Migration{
				Version: version,
				Name:    name,
				Title:   title,
			}
		}

		if isUp {
			migrationsMap[version].UpSQL = string(content)
			// Calculate checksum for up migration
			migrationsMap[version].Checksum = calculateChecksum(string(content))
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
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			slog.Error("Failed to close rows", "error", err)
		}
	}(rows)

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

	// Rollback only if not committed
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			slog.Error("Failed to rollback transaction", "error", err)
		}
	}()

	// Execute migration SQL
	if _, err := tx.Exec(migration.UpSQL); err != nil {
		return fmt.Errorf("migration SQL failed: %w", err)
	}

	// Record migration with title and checksum
	query := `INSERT INTO schema_migrations (version, title, checksum) VALUES ($1, $2, $3)`
	if _, err := tx.Exec(query, migration.Version, migration.Title, migration.Checksum); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// validateMigrationChecksums verifies that applied migrations haven't been modified
func (m *MigrationExecutor) validateMigrationChecksums(migrations []Migration) error {
	// Get applied migrations with their checksums
	query := `SELECT version, title, checksum FROM schema_migrations WHERE checksum IS NOT NULL`
	rows, err := m.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	appliedChecksums := make(map[string]struct {
		title    string
		checksum string
	})

	for rows.Next() {
		var version, title, checksum string
		if err := rows.Scan(&version, &title, &checksum); err != nil {
			return err
		}
		appliedChecksums[version] = struct {
			title    string
			checksum string
		}{title: title, checksum: checksum}
	}

	// Check for mismatches
	var mismatches []string
	for _, migration := range migrations {
		if applied, exists := appliedChecksums[migration.Version]; exists {
			if applied.checksum != migration.Checksum {
				mismatch := fmt.Sprintf(
					"\n  Migration %s (%s):\n    Expected checksum: %s\n    Current checksum:  %s",
					migration.Version, migration.Title, applied.checksum, migration.Checksum,
				)
				mismatches = append(mismatches, mismatch)
			}
		}
	}

	if len(mismatches) > 0 {
		return fmt.Errorf(
			"CRITICAL: Applied migrations have been modified!%s\n\n"+
				"This indicates that migration files that were already applied to the database have been changed.\n"+
				"Modifying applied migrations can lead to inconsistent database states across environments.\n"+
				"Please restore the original migration files or create a new migration to apply the changes.",
			strings.Join(mismatches, ""),
		)
	}

	return nil
}

// calculateChecksum generates a SHA256 checksum for migration content
func calculateChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
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
