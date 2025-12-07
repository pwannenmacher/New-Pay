package validator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// ValidateStruct validates a struct based on validate tags
// This is a basic implementation. For production use, consider using go-playground/validator
func ValidateStruct(s interface{}) error {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return errors.New("not a struct")
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		tag := field.Tag.Get("validate")

		if tag == "" {
			continue
		}

		// Parse validate tag
		rules := strings.Split(tag, ",")
		for _, rule := range rules {
			if err := validateField(field.Name, value, rule); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateField validates a single field based on a rule
func validateField(fieldName string, value reflect.Value, rule string) error {
	switch rule {
	case "required":
		if isZero(value) {
			return fmt.Errorf("%s is required", fieldName)
		}
	case "email":
		if value.Kind() == reflect.String {
			if err := ValidateEmail(value.String()); err != nil {
				return fmt.Errorf("%s must be a valid email", fieldName)
			}
		}
	default:
		// Check for min=X format
		if strings.HasPrefix(rule, "min=") {
			minStr := strings.TrimPrefix(rule, "min=")
			var minVal int
			fmt.Sscanf(minStr, "%d", &minVal)
			if value.Kind() == reflect.String && len(value.String()) < minVal {
				return fmt.Errorf("%s must be at least %d characters", fieldName, minVal)
			}
		}
	}
	return nil
}

// isZero checks if a value is zero/empty
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
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
