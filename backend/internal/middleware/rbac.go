package middleware

import (
	"database/sql"
	"net/http"

	"new-pay/internal/repository"
)

// RBACMiddleware handles role-based access control
type RBACMiddleware struct {
	userRepo *repository.UserRepository
	roleRepo *repository.RoleRepository
}

// NewRBACMiddleware creates a new RBAC middleware
func NewRBACMiddleware(db *sql.DB) *RBACMiddleware {
	return &RBACMiddleware{
		userRepo: repository.NewUserRepository(db),
		roleRepo: repository.NewRoleRepository(db),
	}
}

// RequireRole checks if the user has the required role
func (m *RBACMiddleware) RequireRole(roleName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserID(r)
			if !ok {
				respondWithError(w, http.StatusUnauthorized, "User not authenticated")
				return
			}

			// Get user roles
			roles, err := m.userRepo.GetUserRoles(userID)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to get user roles")
				return
			}

			// Check if user has the required role
			hasRole := false
			for _, role := range roles {
				if role.Name == roleName {
					hasRole = true
					break
				}
			}

			if !hasRole {
				respondWithError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission checks if the user has the required permission
func (m *RBACMiddleware) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserID(r)
			if !ok {
				respondWithError(w, http.StatusUnauthorized, "User not authenticated")
				return
			}

			// Get user roles
			roles, err := m.userRepo.GetUserRoles(userID)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to get user roles")
				return
			}

			// Check permissions for each role
			hasPermission := false
			for _, role := range roles {
				permissions, err := m.roleRepo.GetRolePermissions(role.ID)
				if err != nil {
					continue
				}

				for _, perm := range permissions {
					if perm.Resource == resource && perm.Action == action {
						hasPermission = true
						break
					}
				}

				if hasPermission {
					break
				}
			}

			if !hasPermission {
				respondWithError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole checks if the user has any of the required roles
func (m *RBACMiddleware) RequireAnyRole(roleNames ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserID(r)
			if !ok {
				respondWithError(w, http.StatusUnauthorized, "User not authenticated")
				return
			}

			roles, err := m.userRepo.GetUserRoles(userID)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to get user roles")
				return
			}

			hasRole := false
			for _, role := range roles {
				for _, requiredRole := range roleNames {
					if role.Name == requiredRole {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				respondWithError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
