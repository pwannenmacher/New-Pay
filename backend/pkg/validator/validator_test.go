package validator

import (
	"testing"
)

func TestValidateStruct(t *testing.T) {
	type TestStruct struct {
		Email    string `validate:"required,email"`
		Password string `validate:"required,min=8"`
		Name     string `validate:"required"`
	}

	tests := []struct {
		name     string
		input    TestStruct
		expected bool
	}{
		{
			name: "valid struct",
			input: TestStruct{
				Email:    "test@example.com",
				Password: "password123",
				Name:     "John Doe",
			},
			expected: true,
		},
		{
			name: "missing required field",
			input: TestStruct{
				Email:    "test@example.com",
				Password: "password123",
				Name:     "",
			},
			expected: false,
		},
		{
			name: "invalid email",
			input: TestStruct{
				Email:    "invalid-email",
				Password: "password123",
				Name:     "John Doe",
			},
			expected: false,
		},
		{
			name: "password too short",
			input: TestStruct{
				Email:    "test@example.com",
				Password: "short",
				Name:     "John Doe",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(&tt.input)
			isValid := err == nil

			if isValid != tt.expected {
				t.Errorf("ValidateStruct() = %v, expected %v, error: %v", isValid, tt.expected, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		{"test@example.com", true},
		{"user.name@example.co.uk", true},
		{"invalid-email", false},
		{"@example.com", false},
		{"user@", false},
		{"", false},
		{"user@example", false},
	}

	for _, tt := range tests {
		err := ValidateEmail(tt.email)
		isValid := err == nil

		if isValid != tt.expected {
			t.Errorf("ValidateEmail(%q) = %v, expected %v", tt.email, isValid, tt.expected)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		password string
		expected bool
	}{
		{"password123", true},
		{"12345678", true},
		{"short", false},
		{"", false},
	}

	for _, tt := range tests {
		err := ValidatePassword(tt.password)
		isValid := err == nil

		if isValid != tt.expected {
			t.Errorf("ValidatePassword(%q) = %v, expected %v", tt.password, isValid, tt.expected)
		}
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		field    string
		value    string
		expected bool
	}{
		{"name", "John", true},
		{"name", "", false},
		{"name", "   ", false},
	}

	for _, tt := range tests {
		err := ValidateRequired(tt.field, tt.value)
		isValid := err == nil

		if isValid != tt.expected {
			t.Errorf("ValidateRequired(%q, %q) = %v, expected %v", tt.field, tt.value, isValid, tt.expected)
		}
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  test  ", "test"},
		{"test\x00string", "teststring"},
		{"normal", "normal"},
	}

	for _, tt := range tests {
		result := SanitizeString(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeString(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Test@Example.com", "test@example.com"},
		{"  USER@EXAMPLE.COM  ", "user@example.com"},
	}

	for _, tt := range tests {
		result := SanitizeEmail(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeEmail(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}
