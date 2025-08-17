package errorx

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidatePasswordManual(t *testing.T) {
	tests := []struct {
		name          string
		password      string
		expectedError bool
	}{
		// Valid passwords
		{
			name:          "valid password with all requirements",
			password:      "Password1!",
			expectedError: true,
		},
		{
			name:          "valid password with @ symbol",
			password:      "MyPass123@",
			expectedError: true,
		},
		{
			name:          "valid password with $ symbol",
			password:      "SecureP4$$",
			expectedError: true,
		},
		{
			name:          "valid password with % symbol",
			password:      "Strong9%Pass",
			expectedError: true,
		},
		{
			name:          "valid password with * symbol",
			password:      "Test123*Word",
			expectedError: true,
		},
		{
			name:          "valid password with ? symbol",
			password:      "Question8?Mark",
			expectedError: true,
		},
		{
			name:          "valid password with & symbol",
			password:      "Ampersand7&",
			expectedError: true,
		},
		{
			name:          "valid long password",
			password:      "ThisIsAVeryLongPassword123!",
			expectedError: true,
		},
		{
			name:          "valid password with multiple special chars",
			password:      "Multi9@!Special",
			expectedError: true,
		},

		// Invalid passwords - too short
		{
			name:          "too short - 7 characters",
			password:      "Pass1!",
			expectedError: false,
		},
		{
			name:          "too short - empty string",
			password:      "",
			expectedError: false,
		},
		{
			name:          "too short - 1 character",
			password:      "P",
			expectedError: false,
		},

		// Invalid passwords - missing lowercase
		{
			name:          "missing lowercase letter",
			password:      "PASSWORD1!",
			expectedError: false,
		},
		{
			name:          "only uppercase, digits, and special",
			password:      "TESTPASS123@",
			expectedError: false,
		},

		// Invalid passwords - missing uppercase
		{
			name:          "missing uppercase letter",
			password:      "password1!",
			expectedError: false,
		},
		{
			name:          "only lowercase, digits, and special",
			password:      "testpass123@",
			expectedError: false,
		},

		// Invalid passwords - missing digit
		{
			name:          "missing digit",
			password:      "Password!",
			expectedError: false,
		},
		{
			name:          "only letters and special chars",
			password:      "TestPassword@",
			expectedError: false,
		},

		// Invalid passwords - missing special character
		{
			name:          "missing special character",
			password:      "Password123",
			expectedError: false,
		},
		{
			name:          "only letters and digits",
			password:      "TestPassword123",
			expectedError: false,
		},

		// Invalid passwords - invalid characters
		{
			name:          "contains space",
			password:      "Pass word1!",
			expectedError: false,
		},
		{
			name:          "contains hyphen",
			password:      "Pass-word1!",
			expectedError: false,
		},
		{
			name:          "contains underscore",
			password:      "Pass_word1!",
			expectedError: false,
		},
		{
			name:          "contains period",
			password:      "Pass.word1!",
			expectedError: false,
		},
		{
			name:          "contains comma",
			password:      "Pass,word1!",
			expectedError: false,
		},
		{
			name:          "contains plus",
			password:      "Password1+",
			expectedError: false,
		},
		{
			name:          "contains hash",
			password:      "Password1#",
			expectedError: false,
		},
		{
			name:          "contains unicode",
			password:      "Pássword1!",
			expectedError: false,
		},

		// Edge cases
		{
			name:          "exactly 8 characters - valid",
			password:      "Pass123!",
			expectedError: true,
		},
		{
			name:          "exactly 8 characters - missing requirement",
			password:      "Pass123a",
			expectedError: false,
		},
		{
			name:          "minimum valid with each special char",
			password:      "aB3@efgh",
			expectedError: true,
		},
		{
			name:          "all special characters allowed",
			password:      "aB3@$!%*?&ef",
			expectedError: true,
		},
		{
			name:          "multiple of same character type",
			password:      "AAAaaa111@@@",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordManual(tt.password)
			if (err == nil) != tt.expectedError {
				t.Errorf("ValidatePasswordManual(%q) = %v, expected error: %v", tt.password, err == nil, tt.expectedError)
			} else if err != nil && tt.expectedError {
				t.Errorf("ValidatePasswordManual(%q) returned unexpected error: %v", tt.password, err)
			} else if err == nil && !tt.expectedError {
				t.Logf("Password is valid: %q", tt.password)
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
		length        int
		expectedError bool
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
			err := ValidatePasswordManual(password)
			if (err == nil) != tt.expectedError {
				t.Errorf("ValidatePasswordManual(%q) = %v, expected %v", password, err == nil, tt.expectedError)
			} else if err != nil && tt.expectedError {
				t.Errorf("ValidatePasswordManual(%q) returned unexpected error: %v", password, err)
			} else if err == nil && !tt.expectedError {
				t.Logf("Password of length %d is valid: %q", tt.length, password)
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
			err := ValidatePasswordManual(password)
			if err != nil {
				t.Errorf("Password with allowed special char '%c' should be valid: %q, got error: %v", char, password, err)
			} else {
				t.Logf("Password with allowed special char '%c' is valid: %q", char, password)
			}
		})
	}

	// Test disallowed special characters
	disallowedSpecial := "+-=_[]{}|\\:;\"'<>,./~`"
	for _, char := range disallowedSpecial {
		t.Run(fmt.Sprintf("invalid_special_%c", char), func(t *testing.T) {
			password := basePassword + string(char)
			err := ValidatePasswordManual(password)
			if err == nil {
				t.Errorf("Password with disallowed special char '%c' should be invalid: %q", char, password)
			} else {
				t.Logf("Password with disallowed special char '%c' is correctly invalid: %q, error: %v", char, password, err)
			}
		})
	}
}
