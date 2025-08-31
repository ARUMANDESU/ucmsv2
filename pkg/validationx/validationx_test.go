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
		name     string
		password string
		notError bool
	}{
		// Valid passwords
		{
			name:     "valid password with all requirements",
			password: "Password1!",
			notError: true,
		},
		{
			name:     "valid password with @ symbol",
			password: "MyPass123@",
			notError: true,
		},
		{
			name:     "valid password with $ symbol",
			password: "SecureP4$$",
			notError: true,
		},
		{
			name:     "valid password with % symbol",
			password: "Strong9%Pass",
			notError: true,
		},
		{
			name:     "valid password with * symbol",
			password: "Test123*Word",
			notError: true,
		},
		{
			name:     "valid password with ? symbol",
			password: "Question8?Mark",
			notError: true,
		},
		{
			name:     "valid password with & symbol",
			password: "Ampersand7&",
			notError: true,
		},
		{
			name:     "valid long password",
			password: "ThisIsAVeryLongPassword123!",
			notError: true,
		},
		{
			name:     "valid password with multiple special chars",
			password: "Multi9@!Special",
			notError: true,
		},

		// Invalid passwords - too short
		{
			name:     "too short - 7 characters",
			password: "Pass1!",
			notError: false,
		},
		{
			name:     "too short - empty string",
			password: "",
			notError: false,
		},
		{
			name:     "too short - 1 character",
			password: "P",
			notError: false,
		},

		// Invalid passwords - missing lowercase
		{
			name:     "missing lowercase letter",
			password: "PASSWORD1!",
			notError: false,
		},
		{
			name:     "only uppercase, digits, and special",
			password: "TESTPASS123@",
			notError: false,
		},

		// Invalid passwords - missing uppercase
		{
			name:     "missing uppercase letter",
			password: "password1!",
			notError: false,
		},
		{
			name:     "only lowercase, digits, and special",
			password: "testpass123@",
			notError: false,
		},

		// Invalid passwords - missing digit
		{
			name:     "missing digit",
			password: "Password!",
			notError: false,
		},
		{
			name:     "only letters and special chars",
			password: "TestPassword@",
			notError: false,
		},

		// Invalid passwords - missing special character
		{
			name:     "missing special character",
			password: "Password123",
			notError: false,
		},
		{
			name:     "only letters and digits",
			password: "TestPassword123",
			notError: false,
		},

		// Invalid passwords - invalid characters
		{
			name:     "contains space",
			password: "Pass word1!",
			notError: false,
		},
		{
			name:     "contains unicode",
			password: "Pássword1!",
			notError: false,
		},
		// Valid passwords with special characters
		{
			name:     "contains hyphen",
			password: "Pass-word1!",
			notError: true,
		},
		{
			name:     "contains underscore",
			password: "Pass_word1!",
			notError: true,
		},
		{
			name:     "contains period",
			password: "Pass.word1!",
			notError: true,
		},
		{
			name:     "contains comma",
			password: "Pass,word1!",
			notError: true,
		},
		{
			name:     "contains plus",
			password: "Password1+",
			notError: true,
		},
		{
			name:     "contains hash",
			password: "Password1#",
			notError: true,
		},

		// Edge cases
		{
			name:     "exactly 8 characters - valid",
			password: "Pass123!",
			notError: true,
		},
		{
			name:     "exactly 8 characters - missing requirement",
			password: "Pass123a",
			notError: false,
		},
		{
			name:     "minimum valid with each special char",
			password: "aB3@efgh",
			notError: true,
		},
		{
			name:     "all special characters allowed",
			password: "aB3@$!%*?&ef",
			notError: true,
		},
		{
			name:     "multiple of same character type",
			password: "AAAaaa111@@@",
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
		_ = PasswordFormatRule{}.Validate(password)
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
		name       string
		personName string
		valid      bool
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
		value any
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

func TestNoDuplicate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
		valid bool
	}{
		// Valid cases - no duplicates
		{"no duplicate string slice", []string{"1", "2", "3"}, true},
		{"no duplicate string array", [...]string{"1", "2", "3"}, true},
		{"empty string slice", []string{}, true},
		{"empty string array", [...]string{}, true},
		{"single element string slice", []string{"only"}, true},
		{"single element string array", [...]string{"only"}, true},
		{"no duplicate int slice", []int{1, 2, 3}, true},
		{"no duplicate int array", [...]int{1, 2, 3}, true},
		{"single element int slice", []int{42}, true},
		{"single element int array", [...]int{42}, true},
		{"nil value", nil, true}, // Let Required handle nil if needed
		{"no duplicate float slice", []float64{1.1, 2.2, 3.3}, true},
		{"no duplicate float array", [...]float64{1.1, 2.2, 3.3}, true},
		{"no duplicate byte slice", []byte{0x1, 0x2, 0x3}, true},
		{"no duplicate byte array", [...]byte{0x1, 0x2, 0x3}, true},
		{"large string slice no duplicates", []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}, true},
		{"negative and positive ints", []int{-5, -3, 0, 2, 7}, true},
		{"zero and non-zero floats", []float64{0.0, 1.5, -2.5, 3.14}, true},

		// Integer type variations
		{"no duplicate int8 slice", []int8{1, 2, 3}, true},
		{"no duplicate int16 slice", []int16{100, 200, 300}, true},
		{"no duplicate int32 slice", []int32{1000, 2000, 3000}, true},
		{"no duplicate int64 slice", []int64{10000, 20000, 30000}, true},
		{"no duplicate uint slice", []uint{1, 2, 3}, true},
		{"no duplicate uint8 slice", []uint8{10, 20, 30}, true},
		{"no duplicate uint16 slice", []uint16{100, 200, 300}, true},
		{"no duplicate uint32 slice", []uint32{1000, 2000, 3000}, true},
		{"no duplicate uint64 slice", []uint64{10000, 20000, 30000}, true},
		{"no duplicate float32 slice", []float32{1.1, 2.2, 3.3}, true},

		// Edge cases with zeros and empty values
		{"string slice with empty string", []string{"", "a", "b"}, true},
		{"int slice with zero", []int{0, 1, 2}, true},
		{"float slice with zero", []float64{0.0, 1.1, 2.2}, true},

		// Invalid cases - duplicates
		{"duplicate", []string{"1", "2", "2"}, false},
		{"two duplicates", []string{"1", "1", "no", "no"}, false},
		{"duplicate int", []int{1, 2, 2}, false},
		{"duplicate float", []float64{1.1, 2.2, 2.2}, false},
		{"duplicate byte", []byte{0x1, 0x2, 0x2}, false},
		{"duplicate at beginning", []string{"same", "same", "different"}, false},
		{"duplicate at end", []string{"first", "second", "first"}, false},
		{"all same elements", []string{"same", "same", "same"}, false},
		{"duplicate empty strings", []string{"", "", "a"}, false},
		{"duplicate zeros", []int{0, 1, 0}, false},
		{"duplicate negative numbers", []int{-1, 2, -1}, false},

		// Integer type duplicates
		{"duplicate int8", []int8{1, 2, 1}, false},
		{"duplicate int16", []int16{100, 200, 100}, false},
		{"duplicate int32", []int32{1000, 2000, 1000}, false},
		{"duplicate int64", []int64{10000, 20000, 10000}, false},
		{"duplicate uint", []uint{1, 2, 1}, false},
		{"duplicate uint8", []uint8{10, 20, 10}, false},
		{"duplicate uint16", []uint16{100, 200, 100}, false},
		{"duplicate uint32", []uint32{1000, 2000, 1000}, false},
		{"duplicate uint64", []uint64{10000, 20000, 10000}, false},
		{"duplicate float32", []float32{1.1, 2.2, 1.1}, false},

		// Invalid cases - unsupported types
		{"not a slice or array", "not a slice", false},
		{"no duplicate interface slice", []any{"1", 2, 3}, false},
		{"no duplicate interface array", [...]any{"1", 2, 3}, false},
		{"boolean slice", []bool{true, false, true}, false},
		{"duplicate interface", []any{"1", 2, "1"}, false},
		{"complex struct slice", []struct{ A int }{{1}, {2}, {1}}, false},
		{"complex struct array", [...]struct{ A int }{{1}, {2}, {1}}, false},
		{"map slice", []map[string]int{{"a": 1}, {"b": 2}, {"a": 1}}, false},
		{"map array", [...]map[string]int{{"a": 1}, {"b": 2}, {"a": 1}}, false},
		{"channel slice", []chan int{make(chan int), make(chan int)}, false},
		{"function slice", []func(){func() {}, func() {}}, false},
		{"pointer slice", []*string{&[]string{"a"}[0], &[]string{"b"}[0]}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NoDuplicate.Validate(tt.value)
			if (err == nil) != tt.valid {
				t.Errorf("NoDuplicate(%v) = %v, expected valid: %v, got error: %v", tt.value, err == nil, tt.valid, err)
			}
		})
	}
}

// Additional edge case tests for NoDuplicate validator
func TestNoDuplicateEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("large slices performance", func(t *testing.T) {
		// Test with large slice to ensure performance is reasonable
		largeSlice := make([]int, 1000)
		for i := range largeSlice {
			largeSlice[i] = i
		}
		err := NoDuplicate.Validate(largeSlice)
		assert.NoError(t, err, "Large slice without duplicates should be valid")

		// Add duplicate at the end
		largeSlice[999] = 0
		err = NoDuplicate.Validate(largeSlice)
		assert.Error(t, err, "Large slice with duplicate should be invalid")
	})

	t.Run("unicode strings", func(t *testing.T) {
		unicodeSlice := []string{"🚀", "🌟", "💻", "🔥"}
		err := NoDuplicate.Validate(unicodeSlice)
		assert.NoError(t, err, "Unicode strings without duplicates should be valid")

		duplicateUnicodeSlice := []string{"🚀", "🌟", "🚀"}
		err = NoDuplicate.Validate(duplicateUnicodeSlice)
		assert.Error(t, err, "Unicode strings with duplicates should be invalid")
	})

	t.Run("case sensitivity", func(t *testing.T) {
		caseSensitiveSlice := []string{"Hello", "hello", "HELLO"}
		err := NoDuplicate.Validate(caseSensitiveSlice)
		assert.NoError(t, err, "Case-sensitive strings should be treated as different")

		exactDuplicateSlice := []string{"Hello", "World", "Hello"}
		err = NoDuplicate.Validate(exactDuplicateSlice)
		assert.Error(t, err, "Exact duplicate strings should be invalid")
	})

	t.Run("floating point precision", func(t *testing.T) {
		floatSlice := []float64{1.0, 1.00000001, 1.00000002}
		err := NoDuplicate.Validate(floatSlice)
		assert.NoError(t, err, "Small floating point differences should be treated as different")

		exactFloatDuplicates := []float64{1.23456789, 2.34567890, 1.23456789}
		err = NoDuplicate.Validate(exactFloatDuplicates)
		assert.Error(t, err, "Exact floating point duplicates should be invalid")
	})

	t.Run("boundary values", func(t *testing.T) {
		// Test with type boundary values
		intBoundarySlice := []int64{9223372036854775807, -9223372036854775808, 0}
		err := NoDuplicate.Validate(intBoundarySlice)
		assert.NoError(t, err, "Boundary integer values should work")

		uintBoundarySlice := []uint64{18446744073709551615, 0, 1}
		err = NoDuplicate.Validate(uintBoundarySlice)
		assert.NoError(t, err, "Boundary unsigned integer values should work")
	})
}
