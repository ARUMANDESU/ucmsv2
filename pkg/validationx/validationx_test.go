package validationx

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestValidatePasswordManual(t *testing.T) {
	tests := []struct {
		name          string
		password      string
		notError bool
	}{
		// Valid passwords
		{
			name:          "valid password with all requirements",
			password:      "Password1!",
			notError: true,
		},
		{
			name:          "valid password with @ symbol",
			password:      "MyPass123@",
			notError: true,
		},
		{
			name:          "valid password with $ symbol",
			password:      "SecureP4$$",
			notError: true,
		},
		{
			name:          "valid password with % symbol",
			password:      "Strong9%Pass",
			notError: true,
		},
		{
			name:          "valid password with * symbol",
			password:      "Test123*Word",
			notError: true,
		},
		{
			name:          "valid password with ? symbol",
			password:      "Question8?Mark",
			notError: true,
		},
		{
			name:          "valid password with & symbol",
			password:      "Ampersand7&",
			notError: true,
		},
		{
			name:          "valid long password",
			password:      "ThisIsAVeryLongPassword123!",
			notError: true,
		},
		{
			name:          "valid password with multiple special chars",
			password:      "Multi9@!Special",
			notError: true,
		},

		// Invalid passwords - too short
		{
			name:          "too short - 7 characters",
			password:      "Pass1!",
			notError: false,
		},
		{
			name:          "too short - empty string",
			password:      "",
			notError: false,
		},
		{
			name:          "too short - 1 character",
			password:      "P",
			notError: false,
		},

		// Invalid passwords - missing lowercase
		{
			name:          "missing lowercase letter",
			password:      "PASSWORD1!",
			notError: false,
		},
		{
			name:          "only uppercase, digits, and special",
			password:      "TESTPASS123@",
			notError: false,
		},

		// Invalid passwords - missing uppercase
		{
			name:          "missing uppercase letter",
			password:      "password1!",
			notError: false,
		},
		{
			name:          "only lowercase, digits, and special",
			password:      "testpass123@",
			notError: false,
		},

		// Invalid passwords - missing digit
		{
			name:          "missing digit",
			password:      "Password!",
			notError: false,
		},
		{
			name:          "only letters and special chars",
			password:      "TestPassword@",
			notError: false,
		},

		// Invalid passwords - missing special character
		{
			name:          "missing special character",
			password:      "Password123",
			notError: false,
		},
		{
			name:          "only letters and digits",
			password:      "TestPassword123",
			notError: false,
		},

		// Invalid passwords - invalid characters
		{
			name:          "contains space",
			password:      "Pass word1!",
			notError: false,
		},
		{
			name:          "contains unicode",
			password:      "Pássword1!",
			notError: false,
		},
        // Valid passwords with special characters
		{
			name:          "contains hyphen",
			password:      "Pass-word1!",
			notError: true,
		},
		{
			name:          "contains underscore",
			password:      "Pass_word1!",
			notError: true,
		},
		{
			name:          "contains period",
			password:      "Pass.word1!",
			notError: true,
		},
		{
			name:          "contains comma",
			password:      "Pass,word1!",
			notError: true,
		},
		{
			name:          "contains plus",
			password:      "Password1+",
			notError: true,
		},
		{
			name:          "contains hash",
			password:      "Password1#",
			notError: true,
		},

		// Edge cases
		{
			name:          "exactly 8 characters - valid",
			password:      "Pass123!",
			notError: true,
		},
		{
			name:          "exactly 8 characters - missing requirement",
			password:      "Pass123a",
			notError: false,
		},
		{
			name:          "minimum valid with each special char",
			password:      "aB3@efgh",
			notError: true,
		},
		{
			name:          "all special characters allowed",
			password:      "aB3@$!%*?&ef",
			notError: true,
		},
		{
			name:          "multiple of same character type",
			password:      "AAAaaa111@@@",
			notError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PasswordFormatRule{}.Validate(tt.password)
			if (err == nil) != tt.notError {
				t.Errorf("ValidatePasswordManual(%q) = %v, expected error: %v", tt.password, err == nil, tt.notError)
			} else if err != nil && tt.notError {
				t.Errorf("ValidatePasswordManual(%q) returned unexpected error: %v", tt.password, err)
			} else if err == nil && !tt.notError {
				t.Logf("Password is valid: %q", tt.password)
			}
		})
	}
}

// Benchmark test to measure performance
func BenchmarkValidatePasswordManual(b *testing.B) {
	password := "BenchmarkTest123!"

	for b.Loop() {
		PasswordFormatRule{}.Validate(password)
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
			err := PasswordFormatRule{}.Validate(password)
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

func TestIsUsername(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name     string
        username string
        valid    bool
    }{
        {"valid username", "user_name123", true},
        {"empty", "", true}, // Let Required handle emptiness
        {"only letters", "username", true},
        {"only letters and digits", "user123", true},
        {"capital letters", "ARUMANDESU", true},
        {"contains period", "user.name", true},
        {"contains underscore", "user_name", true},
        {"ends with underscore", "username_", false},
        {"contains hyphen", "user-name", false},
        {"contains plus", "user+name", false},
        {"contains hash", "user#name", false},
        {"too short", "us", false},
        {"too long", strings.Repeat("a", 31), false},
        {"invalid char", "user$name", false},
        {"invalid space", "user name", false},
        {"starts with underscore", "_username", false},
        {"starts with digit", "1username", false},
        {"only digits", "123456", false},
        {"only special chars", "___", false},
        {"mixed invalid chars", "user@name!", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            err := IsUsername.Validate(tt.username)
            if (err == nil) != tt.valid {
                t.Errorf("IsUsername(%q) = %v, expected valid: %v", tt.username, err == nil, tt.valid)
            }
        })
    }
}

func TestIsPersonName(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name     string
        personName string
        valid    bool
    }{
        {"valid name", "John Doe", true},
        {"empty", "", true}, // Let Required handle emptiness
        {"single name", "Alice", true},
        {"name with hyphen", "Mary-Jane", true},
        {"name with apostrophe", "O'Connor", true},
        {"name with period", "Dr. Smith", true},
        {"name with multiple spaces", "  John   Doe  ", true},
        {"name with accented chars", "José Ángel", true},
        {"name with unicode chars", "李小龙", true},
        {"name with comma", "Smith, John", false},
        {"name with invalid char #1", "John_Doe", false},
        {"name with invalid char #2", "Jane@Doe", false},
        {"name with invalid char #3", "Alice!", false},
        {"name with digits", "John123", false},
        {"name with special chars", "Mary#Jane$", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            err := IsPersonName.Validate(tt.personName)
            if (err == nil) != tt.valid {
                t.Errorf("IsPersonName(%q) = %v, expected valid: %v", tt.personName, err == nil, tt.valid)
            }
        })
    }
}

// Test special characters individually
func TestSpecialCharacters(t *testing.T) {
	allowedSpecial := "@$!%*?&+-=_[]{}|\\:;\"'<>,./~`"
	basePassword := "Password1"

	for _, char := range allowedSpecial {
		t.Run(fmt.Sprintf("special_char_%c", char), func(t *testing.T) {
			password := basePassword + string(char)
			err := PasswordFormatRule{}.Validate(password)
			if err != nil {
				t.Errorf("Password with allowed special char '%c' should be valid: %q, got error: %v", char, password, err)
			} else {
				t.Logf("Password with allowed special char '%c' is valid: %q", char, password)
			}
		})
	}

}

type uuidEmbed struct{ uuid.UUID }

type aliasUUID uuid.UUID

func TestRequiredUUID(t *testing.T) {
	tests := []struct {
		name          string
		uuid          any
		expectedError bool
	}{
		{
			name:          "valid string UUID",
			uuid:          "123e4567-e89b-12d3-a456-426614174000",
			expectedError: false,
		},
		{
			name:          "valid UUID object",
			uuid:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
			expectedError: false,
		},
		{
			name:          "valid alias UUID",
			uuid:          aliasUUID(uuid.MustParse("123e4567-e89b-12d3-a456-426614174002")),
			expectedError: false,
		},
		{
			name:          "valid UUID in struct",
			uuid:          uuidEmbed{UUID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174003")},
			expectedError: false,
		},
		{
			name:          "empty string UUID",
			uuid:          "",
			expectedError: true,
		},
		{
			name:          "empty alias UUID",
			uuid:          aliasUUID{},
			expectedError: true,
		},
		{
			name:          "empty UUID object",
			uuid:          uuid.UUID{},
			expectedError: true,
		},
		{
			name:          "nil UUID",
			uuid:          nil,
			expectedError: true,
		},
		{
			name:          "uuid.Nil",
			uuid:          uuid.Nil,
			expectedError: true,
		},
		{
			name:          "empty UUID in struct",
			uuid:          uuidEmbed{UUID: uuid.Nil},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Required.Validate(tt.uuid)
			if (err == nil) != !tt.expectedError {
				t.Errorf("ValidateGroupID(%v) = %v, expected error: %v", tt.uuid, err, tt.expectedError)
			} else if err != nil && !tt.expectedError {
				t.Errorf("ValidateGroupID(%v) returned unexpected error: %v", tt.uuid, err)
			} else if err == nil && tt.expectedError {
				t.Logf("UUID is valid: %v", tt.uuid)
			}

			t.Logf("Test %s completed successfully", tt.name)
			t.Logf("\n")
		})
	}
}

func TestRequired(t *testing.T) {
	s1 := "123"
	s2 := ""
	var time1 time.Time
	tests := []struct {
		tag   string
		value interface{}
		err   string
	}{
		{"t1", 123, ""},
		{"t2", "", "cannot be blank"},
		{"t3", &s1, ""},
		{"t4", &s2, "cannot be blank"},
		{"t5", nil, "cannot be blank"},
		{"t6", time1, "cannot be blank"},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			r := Required
			err := r.Validate(tt.value)
			if err == nil {
				assert.Empty(t, tt.err, tt.tag)
			} else {
				assert.Equal(t, tt.err, err.Error(), tt.tag)
			}
		})
	}
}
