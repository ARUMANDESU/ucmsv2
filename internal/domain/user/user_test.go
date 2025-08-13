package user

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidatePasswordManual(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected bool
	}{
		// Valid passwords
		{
			name:     "valid password with all requirements",
			password: "Password1!",
			expected: true,
		},
		{
			name:     "valid password with @ symbol",
			password: "MyPass123@",
			expected: true,
		},
		{
			name:     "valid password with $ symbol",
			password: "SecureP4$$",
			expected: true,
		},
		{
			name:     "valid password with % symbol",
			password: "Strong9%Pass",
			expected: true,
		},
		{
			name:     "valid password with * symbol",
			password: "Test123*Word",
			expected: true,
		},
		{
			name:     "valid password with ? symbol",
			password: "Question8?Mark",
			expected: true,
		},
		{
			name:     "valid password with & symbol",
			password: "Ampersand7&",
			expected: true,
		},
		{
			name:     "valid long password",
			password: "ThisIsAVeryLongPassword123!",
			expected: true,
		},
		{
			name:     "valid password with multiple special chars",
			password: "Multi9@!Special",
			expected: true,
		},

		// Invalid passwords - too short
		{
			name:     "too short - 7 characters",
			password: "Pass1!",
			expected: false,
		},
		{
			name:     "too short - empty string",
			password: "",
			expected: false,
		},
		{
			name:     "too short - 1 character",
			password: "P",
			expected: false,
		},

		// Invalid passwords - missing lowercase
		{
			name:     "missing lowercase letter",
			password: "PASSWORD1!",
			expected: false,
		},
		{
			name:     "only uppercase, digits, and special",
			password: "TESTPASS123@",
			expected: false,
		},

		// Invalid passwords - missing uppercase
		{
			name:     "missing uppercase letter",
			password: "password1!",
			expected: false,
		},
		{
			name:     "only lowercase, digits, and special",
			password: "testpass123@",
			expected: false,
		},

		// Invalid passwords - missing digit
		{
			name:     "missing digit",
			password: "Password!",
			expected: false,
		},
		{
			name:     "only letters and special chars",
			password: "TestPassword@",
			expected: false,
		},

		// Invalid passwords - missing special character
		{
			name:     "missing special character",
			password: "Password123",
			expected: false,
		},
		{
			name:     "only letters and digits",
			password: "TestPassword123",
			expected: false,
		},

		// Invalid passwords - invalid characters
		{
			name:     "contains space",
			password: "Pass word1!",
			expected: false,
		},
		{
			name:     "contains hyphen",
			password: "Pass-word1!",
			expected: false,
		},
		{
			name:     "contains underscore",
			password: "Pass_word1!",
			expected: false,
		},
		{
			name:     "contains period",
			password: "Pass.word1!",
			expected: false,
		},
		{
			name:     "contains comma",
			password: "Pass,word1!",
			expected: false,
		},
		{
			name:     "contains plus",
			password: "Password1+",
			expected: false,
		},
		{
			name:     "contains hash",
			password: "Password1#",
			expected: false,
		},
		{
			name:     "contains unicode",
			password: "Pássword1!",
			expected: false,
		},

		// Edge cases
		{
			name:     "exactly 8 characters - valid",
			password: "Pass123!",
			expected: true,
		},
		{
			name:     "exactly 8 characters - missing requirement",
			password: "Pass123a",
			expected: false,
		},
		{
			name:     "minimum valid with each special char",
			password: "aB3@efgh",
			expected: true,
		},
		{
			name:     "all special characters allowed",
			password: "aB3@$!%*?&ef",
			expected: true,
		},
		{
			name:     "multiple of same character type",
			password: "AAAaaa111@@@",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePasswordManual(tt.password)
			if result != tt.expected {
				t.Errorf("validatePasswordManual(%q) = %v; expected %v", tt.password, result, tt.expected)
			}
		})
	}
}

// Benchmark test to measure performance
func BenchmarkValidatePasswordManual(b *testing.B) {
	password := "BenchmarkTest123!"

	for b.Loop() {
		ValidatePasswordManual(password)
	}
}

// Test with various password lengths
func TestPasswordLengths(t *testing.T) {
	basePassword := "aB1@"

	tests := []struct {
		length   int
		expected bool
	}{
		{4, false},  // too short
		{7, false},  // too short
		{8, true},   // minimum valid
		{12, true},  // normal length
		{50, true},  // long password
		{100, true}, // very long password
	}

	for _, tt := range tests {
		// Create password of specific length by repeating pattern
		password := strings.Repeat(basePassword, (tt.length/4)+1)[:tt.length]

		// Ensure it has all required character types for lengths >= 8
		if tt.length >= 8 {
			password = "aB1@" + strings.Repeat("x", tt.length-4)
		}

		t.Run(fmt.Sprintf("length_%d", tt.length), func(t *testing.T) {
			result := ValidatePasswordManual(password)
			if result != tt.expected {
				t.Errorf("Password length %d: got %v, expected %v (password: %q)", tt.length, result, tt.expected, password)
			}
		})
	}
}

// Test special characters individually
func TestSpecialCharacters(t *testing.T) {
	allowedSpecial := "@$!%*?&"
	basePassword := "Password1"

	for _, char := range allowedSpecial {
		t.Run(fmt.Sprintf("special_char_%c", char), func(t *testing.T) {
			password := basePassword + string(char)
			result := ValidatePasswordManual(password)
			if !result {
				t.Errorf("Password with special char '%c' should be valid: %q", char, password)
			}
		})
	}

	// Test disallowed special characters
	disallowedSpecial := "+-=_[]{}|\\:;\"'<>,./~`"
	for _, char := range disallowedSpecial {
		t.Run(fmt.Sprintf("invalid_special_%c", char), func(t *testing.T) {
			password := basePassword + string(char)
			result := ValidatePasswordManual(password)
			if result {
				t.Errorf("Password with disallowed special char '%c' should be invalid: %q", char, password)
			}
		})
	}
}
