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
		tc := tc
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
