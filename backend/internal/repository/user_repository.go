package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/pwannenmacher/New-Pay/internal/models"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

// UserRepository handles user database operations
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(user *models.User) error {
	query := `
		INSERT INTO users (email, password_hash, first_name, last_name, oauth_provider, oauth_provider_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	now := time.Now()
	err := r.db.QueryRow(
		query,
		user.Email,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.OAuthProvider,
		user.OAuthProviderID,
		now,
		now,
	).Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	user.CreatedAt = now
	user.UpdatedAt = now
	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(id uint) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, email_verified, email_verified_at,
		       is_active, last_login_at, oauth_provider, oauth_provider_id, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.EmailVerified,
		&user.EmailVerifiedAt,
		&user.IsActive,
		&user.LastLoginAt,
		&user.OAuthProvider,
		&user.OAuthProviderID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, email_verified, email_verified_at,
		       is_active, last_login_at, oauth_provider, oauth_provider_id, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.EmailVerified,
		&user.EmailVerifiedAt,
		&user.IsActive,
		&user.LastLoginAt,
		&user.OAuthProvider,
		&user.OAuthProviderID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetByOAuth retrieves a user by OAuth provider and provider ID
func (r *UserRepository) GetByOAuth(provider, providerID string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, email_verified, email_verified_at,
		       is_active, last_login_at, oauth_provider, oauth_provider_id, created_at, updated_at
		FROM users
		WHERE oauth_provider = $1 AND oauth_provider_id = $2
	`

	user := &models.User{}
	err := r.db.QueryRow(query, provider, providerID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.EmailVerified,
		&user.EmailVerifiedAt,
		&user.IsActive,
		&user.LastLoginAt,
		&user.OAuthProvider,
		&user.OAuthProviderID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by OAuth: %w", err)
	}

	return user, nil
}

// Update updates a user
func (r *UserRepository) Update(user *models.User) error {
	query := `
		UPDATE users
		SET email = $1, first_name = $2, last_name = $3, email_verified = $4,
		    email_verified_at = $5, is_active = $6, last_login_at = $7, updated_at = $8
		WHERE id = $9
	`

	user.UpdatedAt = time.Now()
	_, err := r.db.Exec(
		query,
		user.Email,
		user.FirstName,
		user.LastName,
		user.EmailVerified,
		user.EmailVerifiedAt,
		user.IsActive,
		user.LastLoginAt,
		user.UpdatedAt,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(userID uint, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.Exec(query, passwordHash, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// VerifyEmail marks a user's email as verified
func (r *UserRepository) VerifyEmail(userID uint) error {
	query := `
		UPDATE users
		SET email_verified = true, email_verified_at = $1, updated_at = $2
		WHERE id = $3
	`

	now := time.Now()
	_, err := r.db.Exec(query, now, now, userID)
	if err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	return nil
}

// UnverifyEmail marks a user's email as unverified
func (r *UserRepository) UnverifyEmail(userID uint) error {
	query := `
		UPDATE users
		SET email_verified = false, email_verified_at = NULL, updated_at = $1
		WHERE id = $2
	`

	_, err := r.db.Exec(query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to unverify email: %w", err)
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(userID uint) error {
	query := `
		UPDATE users
		SET last_login_at = $1, updated_at = $2
		WHERE id = $3
	`

	now := time.Now()
	_, err := r.db.Exec(query, now, now, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// Delete deletes a user
func (r *UserRepository) Delete(id uint) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// GetUserRoles retrieves all roles for a user
func (r *UserRepository) GetUserRoles(userID uint) ([]models.Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.created_at, r.updated_at
		FROM roles r
		INNER JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// AssignRole assigns a role to a user
func (r *UserRepository) AssignRole(userID, roleID uint) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`

	_, err := r.db.Exec(query, userID, roleID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	return nil
}

// RemoveRole removes a role from a user
func (r *UserRepository) RemoveRole(userID, roleID uint) error {
	query := `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`
	_, err := r.db.Exec(query, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}
	return nil
}

// UpdateActiveStatus updates the is_active status of a user
func (r *UserRepository) UpdateActiveStatus(userID uint, isActive bool) error {
	query := `
		UPDATE users
		SET is_active = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.Exec(query, isActive, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update active status: %w", err)
	}

	return nil
}

// UserFilters holds filter parameters for user queries
type UserFilters struct {
	Search        string
	RoleIDs       []int
	IsActive      *bool
	EmailVerified *bool
	SortBy        string
	SortOrder     string
}

// GetAll retrieves all users with pagination
func (r *UserRepository) GetAll(limit, offset int) ([]models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, email_verified, email_verified_at,
		       is_active, last_login_at, oauth_provider, oauth_provider_id, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.FirstName,
			&user.LastName,
			&user.EmailVerified,
			&user.EmailVerifiedAt,
			&user.IsActive,
			&user.LastLoginAt,
			&user.OAuthProvider,
			&user.OAuthProviderID,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// GetAllWithFilters retrieves users with filtering, sorting, and pagination
func (r *UserRepository) GetAllWithFilters(filters UserFilters, limit, offset int) ([]models.User, error) {
	query := `
		SELECT DISTINCT u.id, u.email, u.password_hash, u.first_name, u.last_name, u.email_verified, 
		       u.email_verified_at, u.is_active, u.last_login_at, u.oauth_provider, u.oauth_provider_id, 
		       u.created_at, u.updated_at
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	// Search filter (name or email)
	if filters.Search != "" {
		query += fmt.Sprintf(` AND (u.email ILIKE $%d OR u.first_name ILIKE $%d OR u.last_name ILIKE $%d)`, argPos, argPos, argPos)
		args = append(args, "%"+filters.Search+"%")
		argPos++
	}

	// Role filter (user must have ALL selected roles)
	if len(filters.RoleIDs) > 0 {
		query += fmt.Sprintf(` AND ur.role_id = ANY($%d)`, argPos)
		args = append(args, pq.Array(filters.RoleIDs))
		argPos++
	}

	// Active status filter
	if filters.IsActive != nil {
		query += fmt.Sprintf(` AND u.is_active = $%d`, argPos)
		args = append(args, *filters.IsActive)
		argPos++
	}

	// Email verified filter
	if filters.EmailVerified != nil {
		query += fmt.Sprintf(` AND u.email_verified = $%d`, argPos)
		args = append(args, *filters.EmailVerified)
		argPos++
	}

	// Sorting
	sortColumn := "u.created_at"
	sortOrder := "DESC"

	if filters.SortBy != "" {
		switch filters.SortBy {
		case "id":
			sortColumn = "u.id"
		case "email":
			sortColumn = "u.email"
		case "name":
			sortColumn = "u.first_name"
		case "created_at":
			sortColumn = "u.created_at"
		case "last_login_at":
			sortColumn = "u.last_login_at"
		}
	}

	if filters.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	// Group by user and filter by role count if roles are specified
	if len(filters.RoleIDs) > 0 {
		query += ` GROUP BY u.id, u.email, u.password_hash, u.first_name, u.last_name, u.email_verified, u.email_verified_at, u.is_active, u.last_login_at, u.oauth_provider, u.oauth_provider_id, u.created_at, u.updated_at`
		query += fmt.Sprintf(` HAVING COUNT(DISTINCT ur.role_id) = %d`, len(filters.RoleIDs))
	}

	query += fmt.Sprintf(` ORDER BY %s %s LIMIT $%d OFFSET $%d`, sortColumn, sortOrder, argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.FirstName,
			&user.LastName,
			&user.EmailVerified,
			&user.EmailVerifiedAt,
			&user.IsActive,
			&user.LastLoginAt,
			&user.OAuthProvider,
			&user.OAuthProviderID,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// CountWithFilters returns the total count of users matching the filters
func (r *UserRepository) CountWithFilters(filters UserFilters) (int, error) {
	query := `
		SELECT COUNT(DISTINCT u.id)
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	if filters.Search != "" {
		query += fmt.Sprintf(` AND (u.email ILIKE $%d OR u.first_name ILIKE $%d OR u.last_name ILIKE $%d)`, argPos, argPos, argPos)
		args = append(args, "%"+filters.Search+"%")
		argPos++
	}

	if len(filters.RoleIDs) > 0 {
		query += fmt.Sprintf(` AND ur.role_id = ANY($%d)`, argPos)
		args = append(args, pq.Array(filters.RoleIDs))
		argPos++
	}

	if filters.IsActive != nil {
		query += fmt.Sprintf(` AND u.is_active = $%d`, argPos)
		args = append(args, *filters.IsActive)
		argPos++
	}

	if filters.EmailVerified != nil {
		query += fmt.Sprintf(` AND u.email_verified = $%d`, argPos)
		args = append(args, *filters.EmailVerified)
		argPos++
	}

	// Group by user and filter by role count if roles are specified
	if len(filters.RoleIDs) > 0 {
		query += ` GROUP BY u.id`
		query += fmt.Sprintf(` HAVING COUNT(DISTINCT ur.role_id) = %d`, len(filters.RoleIDs))
	}

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// CountAll returns the total number of users in the system
func (r *UserRepository) CountAll() (int, error) {
	query := `SELECT COUNT(*) FROM users`

	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count all users: %w", err)
	}

	return count, nil
}

// CountActiveAdmins returns the number of active users with the Admin role
func (r *UserRepository) CountActiveAdmins() (int, error) {
	query := `
		SELECT COUNT(DISTINCT u.id)
		FROM users u
		JOIN user_roles ur ON u.id = ur.user_id
		JOIN roles r ON ur.role_id = r.id
		WHERE u.is_active = true AND r.name = 'Admin'
	`

	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active admins: %w", err)
	}

	return count, nil
}

// IsLastActiveAdmin checks if a user is the last active admin in the system
func (r *UserRepository) IsLastActiveAdmin(userID uint) (bool, error) {
	count, err := r.CountActiveAdmins()
	if err != nil {
		return false, err
	}

	// If only 1 active admin, check if it's this user
	if count == 1 {
		query := `
			SELECT EXISTS(
				SELECT 1
				FROM users u
				JOIN user_roles ur ON u.id = ur.user_id
				JOIN roles r ON ur.role_id = r.id
				WHERE u.id = $1 AND u.is_active = true AND r.name = 'Admin'
			)
		`

		var isAdmin bool
		err := r.db.QueryRow(query, userID).Scan(&isAdmin)
		if err != nil {
			return false, fmt.Errorf("failed to check if user is admin: %w", err)
		}

		return isAdmin, nil
	}

	return false, nil
}
