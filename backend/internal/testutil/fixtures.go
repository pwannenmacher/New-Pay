package testutil

import (
	"database/sql"
	"fmt"
	"new-pay/internal/models"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Fixtures holds test data
type Fixtures struct {
	DB           *sql.DB
	AdminUser    *models.User
	ReviewerUser *models.User
	RegularUser  *models.User
	Catalog      *models.CriteriaCatalog
	Categories   []models.Category
	Paths        []models.Path
	Levels       []models.Level
}

// SetupFixtures creates test data
func SetupFixtures(t *testing.T, db *sql.DB) *Fixtures {
	t.Helper()

	fixtures := &Fixtures{
		DB: db,
	}

	// Create roles
	adminRole := createRole(t, db, "admin")
	reviewerRole := createRole(t, db, "reviewer")
	userRole := createRole(t, db, "user")

	// Create users
	fixtures.AdminUser = createUser(t, db, "admin@test.com", "Admin", "User", []int{int(adminRole.ID), int(reviewerRole.ID), int(userRole.ID)})
	fixtures.ReviewerUser = createUser(t, db, "reviewer@test.com", "Reviewer", "User", []int{int(reviewerRole.ID), int(userRole.ID)})
	fixtures.RegularUser = createUser(t, db, "user@test.com", "Regular", "User", []int{int(userRole.ID)})

	// Create catalog with categories, paths, and levels
	fixtures.Catalog = createCatalog(t, db)
	fixtures.Categories = createCategories(t, db, fixtures.Catalog.ID)
	fixtures.Paths = createPaths(t, db, fixtures.Catalog.ID)
	fixtures.Levels = createLevels(t, db, fixtures.Catalog.ID)

	return fixtures
}

// CleanupFixtures removes all test data
func (f *Fixtures) Cleanup(t *testing.T) {
	t.Helper()

	// Cleanup is handled by container termination
	// Data is not persisted between tests
}

// createRole creates a role in the database or returns existing
func createRole(t *testing.T, db *sql.DB, name string) *models.Role {
	t.Helper()

	var role models.Role

	// Try to get existing role first
	err := db.QueryRow(
		"SELECT id, name, created_at, updated_at FROM roles WHERE name = $1",
		name,
	).Scan(&role.ID, &role.Name, &role.CreatedAt, &role.UpdatedAt)

	if err == nil {
		// Role already exists
		return &role
	}

	// Create new role
	err = db.QueryRow(
		"INSERT INTO roles (name) VALUES ($1) RETURNING id, name, created_at, updated_at",
		name,
	).Scan(&role.ID, &role.Name, &role.CreatedAt, &role.UpdatedAt)

	if err != nil {
		t.Fatalf("Failed to create role %s: %v", name, err)
	}

	return &role
}

// createUser creates a user with specified roles
func createUser(t *testing.T, db *sql.DB, email, firstName, lastName string, roleIDs []int) *models.User {
	t.Helper()

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Create user
	var user models.User
	err = db.QueryRow(`
		INSERT INTO users (email, password_hash, first_name, last_name, email_verified)
		VALUES ($1, $2, $3, $4, true)
		RETURNING id, email, first_name, last_name, email_verified, created_at, updated_at
	`, email, string(hashedPassword), firstName, lastName).Scan(
		&user.ID, &user.Email, &user.FirstName, &user.LastName,
		&user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		t.Fatalf("Failed to create user %s: %v", email, err)
	}

	// Assign roles
	for _, roleID := range roleIDs {
		_, err := db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)", user.ID, roleID)
		if err != nil {
			t.Fatalf("Failed to assign role %d to user %s: %v", roleID, email, err)
		}
	}

	return &user
}

// createCatalog creates a test catalog
func createCatalog(t *testing.T, db *sql.DB) *models.CriteriaCatalog {
	t.Helper()

	validFrom := time.Now().Add(-24 * time.Hour)
	validUntil := time.Now().Add(365 * 24 * time.Hour)

	var catalog models.CriteriaCatalog
	err := db.QueryRow(`
		INSERT INTO criteria_catalogs (name, description, phase, valid_from, valid_until)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, description, phase, valid_from, valid_until, created_at, updated_at
	`, "Test Catalog", "Test catalog for integration tests", "active", validFrom, validUntil).Scan(
		&catalog.ID, &catalog.Name, &catalog.Description, &catalog.Phase,
		&catalog.ValidFrom, &catalog.ValidUntil, &catalog.CreatedAt, &catalog.UpdatedAt,
	)

	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	return &catalog
}

// createCategories creates test categories
func createCategories(t *testing.T, db *sql.DB, catalogID uint) []models.Category {
	t.Helper()

	categories := []models.Category{}
	categoryNames := []string{"Category A", "Category B", "Category C"}
	weights := []float64{0.4, 0.3, 0.3}

	for i, name := range categoryNames {
		var category models.Category
		err := db.QueryRow(`
			INSERT INTO categories (catalog_id, name, description, weight, sort_order)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, catalog_id, name, description, weight, sort_order, created_at, updated_at
		`, catalogID, name, fmt.Sprintf("Description for %s", name), weights[i], i+1).Scan(
			&category.ID, &category.CatalogID, &category.Name, &category.Description,
			&category.Weight, &category.SortOrder, &category.CreatedAt, &category.UpdatedAt,
		)

		if err != nil {
			t.Fatalf("Failed to create category %s: %v", name, err)
		}

		categories = append(categories, category)
	}

	return categories
}

// createPaths creates test development paths
func createPaths(t *testing.T, db *sql.DB, catalogID uint) []models.Path {
	t.Helper()

	// First, get categories for this catalog
	rows, err := db.Query("SELECT id FROM categories WHERE catalog_id = $1 ORDER BY sort_order LIMIT 1", catalogID)
	if err != nil {
		t.Fatalf("Failed to get categories: %v", err)
	}
	defer rows.Close()

	var categoryID uint
	if rows.Next() {
		if err := rows.Scan(&categoryID); err != nil {
			t.Fatalf("Failed to scan category ID: %v", err)
		}
	}

	paths := []models.Path{}
	pathNames := []string{"Technical Path", "Management Path"}

	for i, name := range pathNames {
		var path models.Path
		err := db.QueryRow(`
			INSERT INTO paths (category_id, name, description, sort_order)
			VALUES ($1, $2, $3, $4)
			RETURNING id, category_id, name, description, sort_order, created_at, updated_at
		`, categoryID, name, fmt.Sprintf("Description for %s", name), i+1).Scan(
			&path.ID, &path.CategoryID, &path.Name, &path.Description,
			&path.SortOrder, &path.CreatedAt, &path.UpdatedAt,
		)

		if err != nil {
			t.Fatalf("Failed to create path %s: %v", name, err)
		}

		paths = append(paths, path)
	}

	return paths
}

// createLevels creates test levels
func createLevels(t *testing.T, db *sql.DB, catalogID uint) []models.Level {
	t.Helper()

	levels := []models.Level{}
	levelData := []struct {
		name   string
		number int
	}{
		{"Beginner", 1},
		{"Intermediate", 2},
		{"Advanced", 3},
		{"Expert", 4},
	}

	for _, data := range levelData {
		var level models.Level
		err := db.QueryRow(`
			INSERT INTO levels (catalog_id, name, level_number, description)
			VALUES ($1, $2, $3, $4)
			RETURNING id, catalog_id, name, level_number, description, created_at, updated_at
		`, catalogID, data.name, data.number, fmt.Sprintf("Description for %s", data.name)).Scan(
			&level.ID, &level.CatalogID, &level.Name, &level.LevelNumber,
			&level.Description, &level.CreatedAt, &level.UpdatedAt,
		)

		if err != nil {
			t.Fatalf("Failed to create level %s: %v", data.name, err)
		}

		levels = append(levels, level)
	}

	return levels
}

// CreateSelfAssessment creates a self-assessment for testing
func (f *Fixtures) CreateSelfAssessment(t *testing.T, userID uint, status string) *models.SelfAssessment {
	t.Helper()

	var assessment models.SelfAssessment
	err := f.DB.QueryRow(`
		INSERT INTO self_assessments (catalog_id, user_id, status)
		VALUES ($1, $2, $3)
		RETURNING id, catalog_id, user_id, status, created_at, updated_at
	`, f.Catalog.ID, userID, status).Scan(
		&assessment.ID, &assessment.CatalogID, &assessment.UserID,
		&assessment.Status, &assessment.CreatedAt, &assessment.UpdatedAt,
	)

	if err != nil {
		t.Fatalf("Failed to create self-assessment: %v", err)
	}

	return &assessment
}

// CreateAssessmentResponse creates a response for testing
func (f *Fixtures) CreateAssessmentResponse(t *testing.T, assessmentID, categoryID, pathID, levelID uint) *models.AssessmentResponse {
	t.Helper()

	var response models.AssessmentResponse
	err := f.DB.QueryRow(`
		INSERT INTO assessment_responses (assessment_id, category_id, path_id, level_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, assessment_id, category_id, path_id, level_id, created_at, updated_at
	`, assessmentID, categoryID, pathID, levelID).Scan(
		&response.ID, &response.AssessmentID, &response.CategoryID,
		&response.PathID, &response.LevelID, &response.CreatedAt, &response.UpdatedAt,
	)

	if err != nil {
		t.Fatalf("Failed to create assessment response: %v", err)
	}

	return &response
}
