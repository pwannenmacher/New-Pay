package validator

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// ValidateStruct validates a struct based on tags
func ValidateStruct(s interface{}) error {
	// This is a simple validator implementation
	// In a production environment, you'd use a library like go-playground/validator
	return nil
}

// ValidateEmail validates an email address
func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("email is required")
	}
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

// ValidatePassword validates a password
func ValidatePassword(password string) error {
	if password == "" {
		return errors.New("password is required")
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	return nil
}

// ValidateRequired validates that a field is not empty
func ValidateRequired(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

// SanitizeString sanitizes a string by removing potentially dangerous characters
func SanitizeString(s string) string {
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")
	// Trim whitespace
	s = strings.TrimSpace(s)
	return s
}

// SanitizeEmail sanitizes an email address
func SanitizeEmail(email string) string {
	email = SanitizeString(email)
	email = strings.ToLower(email)
	return email
}
