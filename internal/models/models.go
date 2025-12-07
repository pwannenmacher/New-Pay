package models

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID              uint       `json:"id" db:"id"`
	Email           string     `json:"email" db:"email"`
	PasswordHash    string     `json:"-" db:"password_hash"`
	FirstName       string     `json:"first_name" db:"first_name"`
	LastName        string     `json:"last_name" db:"last_name"`
	EmailVerified   bool       `json:"email_verified" db:"email_verified"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty" db:"email_verified_at"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	OAuthProvider   *string    `json:"oauth_provider,omitempty" db:"oauth_provider"`
	OAuthProviderID *string    `json:"-" db:"oauth_provider_id"`
}

// Role represents a user role
type Role struct {
	ID          uint      `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Permission represents a permission in the system
type Permission struct {
	ID          uint      `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Resource    string    `json:"resource" db:"resource"`
	Action      string    `json:"action" db:"action"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// UserRole represents the many-to-many relationship between users and roles
type UserRole struct {
	UserID    uint      `json:"user_id" db:"user_id"`
	RoleID    uint      `json:"role_id" db:"role_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// RolePermission represents the many-to-many relationship between roles and permissions
type RolePermission struct {
	RoleID       uint      `json:"role_id" db:"role_id"`
	PermissionID uint      `json:"permission_id" db:"permission_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// EmailVerificationToken represents a token for email verification
type EmailVerificationToken struct {
	ID        uint       `json:"id" db:"id"`
	UserID    uint       `json:"user_id" db:"user_id"`
	Token     string     `json:"token" db:"token"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// PasswordResetToken represents a token for password reset
type PasswordResetToken struct {
	ID        uint       `json:"id" db:"id"`
	UserID    uint       `json:"user_id" db:"user_id"`
	Token     string     `json:"token" db:"token"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// Session represents a user session
type Session struct {
	ID             string    `json:"id" db:"id"`
	UserID         uint      `json:"user_id" db:"user_id"`
	Token          string    `json:"token" db:"token"`
	ExpiresAt      time.Time `json:"expires_at" db:"expires_at"`
	LastActivityAt time.Time `json:"last_activity_at" db:"last_activity_at"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	IPAddress      string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent      string    `json:"user_agent,omitempty" db:"user_agent"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        uint      `json:"id" db:"id"`
	UserID    *uint     `json:"user_id,omitempty" db:"user_id"`
	Action    string    `json:"action" db:"action"`
	Resource  string    `json:"resource" db:"resource"`
	Details   string    `json:"details,omitempty" db:"details"`
	IPAddress string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent string    `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// UserWithRoles extends User with roles information
type UserWithRoles struct {
	User
	Roles []Role `json:"roles"`
}
