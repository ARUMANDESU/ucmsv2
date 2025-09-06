package logging

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "valid - normal ascii",
			email:    "valid@gmail.com",
			expected: "va****@gmail.com",
		},
		{
			name:     "empty",
			email:    "",
			expected: "",
		},
		{
			name:     "too short local - 1 rune",
			email:    "a@b.c",
			expected: "a@b.c", // Not enough characters to redact
		},
		{
			name:     "too short local - 2 runes",
			email:    "ab@b.c",
			expected: "ab@b.c", // Not enough characters to redact
		},
		{
			name:     "exact threshold - 3 runes",
			email:    "abc@domain.com",
			expected: "ab****@domain.com",
		},
		{
			name:     "unicode local (cyrillic)",
			email:    "абвгд@пример.рф",
			expected: "аб****@пример.рф",
		},
		{
			name:     "unicode emoji",
			email:    "😀😀😀@ex.com",
			expected: "😀😀****@ex.com",
		},
		{
			name:     "leading and trailing whitespace",
			email:    "   elise@example.com   ",
			expected: "el****@example.com",
		},
		{
			name:     "malformed - no at",
			email:    "nonsense",
			expected: "nonsense", // returned unchanged
		},
		{
			name:     "malformed - at at start",
			email:    "@example.com",
			expected: "@example.com", // returned unchanged
		},
		{
			name:     "malformed - at at end",
			email:    "local@",
			expected: "local@", // returned unchanged
		},
		{
			name:     "multiple ats - redacts up to first at",
			email:    "first@second@domain.com",
			expected: "fi****@second@domain.com",
		},
		{
			name:     "preserve domain - deep subdomain",
			email:    "abcdef@sub.example.co.uk",
			expected: "ab****@sub.example.co.uk",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := RedactEmail(tc.email)
			assert.Equal(t, tc.expected, result, "Redacted email should match expected value")
		})
	}
}

func TestRedactEmail_PreservesDomainSuffix(t *testing.T) {
	t.Parallel()

	in := "abcdef@sub.example.co.uk"
	out := RedactEmail(in)

	// Whatever masking happens to the local part, the domain must be intact
	assert.True(t, strings.HasSuffix(out, "@sub.example.co.uk"))
}

func TestRedactUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		username string
		expected string
	}{
		{
			name:     "empty username",
			username: "",
			expected: "",
		},
		{
			name:     "1 rune - unchanged",
			username: "a",
			expected: "a",
		},
		{
			name:     "2 runes - unchanged",
			username: "ab",
			expected: "ab",
		},
		{
			name:     "3 runes - threshold",
			username: "abc",
			expected: "ab****",
		},
		{
			name:     "normal ascii username",
			username: "john_doe",
			expected: "jo****",
		},
		{
			name:     "long username",
			username: "verylongusername123",
			expected: "ve****",
		},
		{
			name:     "unicode cyrillic",
			username: "пользователь",
			expected: "по****",
		},
		{
			name:     "unicode emoji",
			username: "😀😁😂😃",
			expected: "😀😁****",
		},
		{
			name:     "mixed unicode and ascii",
			username: "user_пользователь",
			expected: "us****",
		},
		{
			name:     "numbers and symbols",
			username: "user123!@#",
			expected: "us****",
		},
		{
			name:     "whitespace handling",
			username: "  user  ",
			expected: "us****",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := RedactUsername(tc.username)
			assert.Equal(t, tc.expected, result, "Redacted username should match expected value")
		})
	}
}

func TestRedactKeepPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tag      string
		input    string
		keep     int
		expected string
	}{
		{"t1", "", 3, ""},
		{"t2", "a", 3, "a"},
		{"t3", "ab", 3, "ab"},
		{"t4", "abc", 3, "abc"},
		{"t5", "abcd", 3, "abc****"},
		{"t6", "abcdefg", 4, "abcd****"},
		{"t7", "  spaced  ", 3, "spa****"},
		{"t8", "пользователь", 3, "пол****"},
		{"t9", "😀😁😂😃", 2, "😀😁****"},
		{"t10", "mixedユーザー", 5, "mixed****"},
		{"t11", "short", 10, "short"},
		{"t12", "exactlyten", 10, "exactlyten"},
		{"t13", "elevenchars", 10, "elevenchar****"},
	}

	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) {
			t.Parallel()
			result := RedactKeepPrefix(tc.input, tc.keep)
			assert.Equal(t, tc.expected, result, "Redacted string should match expected value")
		})
	}
}
