package repository

import (
	"database/sql"
	"fmt"
	"time"

	"new-pay/internal/models"
)

// RoleRepository handles role database operations
type RoleRepository struct {
	db *sql.DB
}

// NewRoleRepository creates a new role repository
func NewRoleRepository(db *sql.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

// Create creates a new role
func (r *RoleRepository) Create(role *models.Role) error {
	query := `
		INSERT INTO roles (name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	now := time.Now()
	err := r.db.QueryRow(query, role.Name, role.Description, now, now).Scan(&role.ID)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	role.CreatedAt = now
	role.UpdatedAt = now
	return nil
}

// GetByID retrieves a role by ID
func (r *RoleRepository) GetByID(id uint) (*models.Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		WHERE id = $1
	`

	role := &models.Role{}
	err := r.db.QueryRow(query, id).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.CreatedAt,
		&role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return role, nil
}

// GetByName retrieves a role by name
func (r *RoleRepository) GetByName(name string) (*models.Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		WHERE name = $1
	`

	role := &models.Role{}
	err := r.db.QueryRow(query, name).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.CreatedAt,
		&role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role by name: %w", err)
	}

	return role, nil
}

// GetAll retrieves all roles
func (r *RoleRepository) GetAll() ([]models.Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		ORDER BY name
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
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

// Update updates a role
func (r *RoleRepository) Update(role *models.Role) error {
	query := `
		UPDATE roles
		SET name = $1, description = $2, updated_at = $3
		WHERE id = $4
	`

	role.UpdatedAt = time.Now()
	_, err := r.db.Exec(query, role.Name, role.Description, role.UpdatedAt, role.ID)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	return nil
}

// Delete deletes a role
func (r *RoleRepository) Delete(id uint) error {
	query := `DELETE FROM roles WHERE id = $1`
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}
	return nil
}

// GetRolePermissions retrieves all permissions for a role
func (r *RoleRepository) GetRolePermissions(roleID uint) ([]models.Permission, error) {
	query := `
		SELECT p.id, p.name, p.resource, p.action, p.description, p.created_at, p.updated_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
	`

	rows, err := r.db.Query(query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		var perm models.Permission
		if err := rows.Scan(&perm.ID, &perm.Name, &perm.Resource, &perm.Action, &perm.Description, &perm.CreatedAt, &perm.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, perm)
	}

	return permissions, nil
}

// AssignPermission assigns a permission to a role
func (r *RoleRepository) AssignPermission(roleID, permissionID uint) error {
	query := `
		INSERT INTO role_permissions (role_id, permission_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`

	_, err := r.db.Exec(query, roleID, permissionID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to assign permission: %w", err)
	}

	return nil
}

// RemovePermission removes a permission from a role
func (r *RoleRepository) RemovePermission(roleID, permissionID uint) error {
	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`
	_, err := r.db.Exec(query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to remove permission: %w", err)
	}
	return nil
}

// GetUsersByRole retrieves all users with a specific role
func (r *RoleRepository) GetUsersByRole(roleName string) ([]models.User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.first_name, u.last_name, 
		       u.email_verified, u.email_verified_at, u.is_active, u.last_login_at,
		       u.created_at, u.updated_at, u.oauth_provider, u.oauth_provider_id
		FROM users u
		INNER JOIN user_roles ur ON u.id = ur.user_id
		INNER JOIN roles r ON ur.role_id = r.id
		WHERE r.name = $1
	`

	rows, err := r.db.Query(query, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by role: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
			&user.EmailVerified, &user.EmailVerifiedAt, &user.IsActive, &user.LastLoginAt,
			&user.CreatedAt, &user.UpdatedAt, &user.OAuthProvider, &user.OAuthProviderID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}
