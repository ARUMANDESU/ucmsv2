package sanitizex

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

func TestCleanSingleLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic trimming",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "collapse multiple spaces",
			input:    "hello    world",
			expected: "hello world",
		},
		{
			name:     "remove newlines",
			input:    "hello\nworld",
			expected: "hello world",
		},
		{
			name:     "remove tabs",
			input:    "hello\tworld",
			expected: "hello world",
		},
		{
			name:     "remove carriage returns",
			input:    "hello\rworld",
			expected: "hello world",
		},
		{
			name:     "mixed whitespace",
			input:    "  hello \n\t  world \r  ",
			expected: "hello world",
		},
		{
			name:     "control characters",
			input:    "hello\x00\x01\x02world",
			expected: "hello world",
		},
		{
			name:     "unicode normalization - decomposed",
			input:    "café", // e with combining acute accent
			expected: "café", // composed form
		},
		{
			name:     "unicode normalization - composed",
			input:    "café", // already composed
			expected: "café",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \n\t\r   ",
			expected: "",
		},
		{
			name:     "special characters preserved",
			input:    "hello@world.com",
			expected: "hello@world.com",
		},
		{
			name:     "unicode characters preserved",
			input:    "héllo wörld 你好",
			expected: "héllo wörld 你好",
		},
		{
			name:     "emojis preserved",
			input:    "hello 👋 world 🌍",
			expected: "hello 👋 world 🌍",
		},
		{
			name:     "multiple consecutive control chars",
			input:    "hello\x00\x01\x02\x03world",
			expected: "hello world",
		},
		{
			name:     "leading and trailing control chars",
			input:    "\x00hello world\x1F",
			expected: "hello world",
		},
		{
			name:     "only control characters",
			input:    "\x00\x01\x02\x1F",
			expected: "",
		},
		{
			name:     "mixed unicode and control chars",
			input:    "  héllo\x00\nwörld\t  ",
			expected: "héllo wörld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanSingleLine(tt.input)
			if result != tt.expected {
				t.Errorf("CleanSingleLine(%q) = %q, want %q", tt.input, result, tt.expected)
			}

			// Additional checks for single line constraints
			if strings.Contains(result, "\n") {
				t.Errorf("CleanSingleLine(%q) = %q, should not contain newlines", tt.input, result)
			}
			if strings.Contains(result, "\t") {
				t.Errorf("CleanSingleLine(%q) = %q, should not contain tabs", tt.input, result)
			}
			if strings.Contains(result, "\r") {
				t.Errorf("CleanSingleLine(%q) = %q, should not contain carriage returns", tt.input, result)
			}

			// Check for control characters
			for _, r := range result {
				if unicode.IsControl(r) {
					t.Errorf("CleanSingleLine(%q) = %q, should not contain control characters", tt.input, result)
					break
				}
			}

			// Check for multiple consecutive spaces
			if strings.Contains(result, "  ") {
				t.Errorf("CleanSingleLine(%q) = %q, should not contain multiple consecutive spaces", tt.input, result)
			}

			// Check for leading/trailing whitespace
			if result != strings.TrimSpace(result) {
				t.Errorf("CleanSingleLine(%q) = %q, should not have leading/trailing whitespace", tt.input, result)
			}
		})
	}
}

func TestCleanMultiline(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic multiline",
			input:    "line1\nline2\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "trim whitespace from lines",
			input:    "  line1  \n  line2  \n  line3  ",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "preserve tabs within lines",
			input:    "line1\tindented\nline2\tindented",
			expected: "line1\tindented\nline2\tindented",
		},
		{
			name:     "remove control characters",
			input:    "line1\x00test\nline2\x01test",
			expected: "line1test\nline2test",
		},
		{
			name:     "empty lines",
			input:    "line1\n\nline3",
			expected: "line1\n\nline3",
		},
		{
			name:     "lines with only whitespace",
			input:    "line1\n   \t   \nline3",
			expected: "line1\n\nline3",
		},
		{
			name:     "unicode normalization multiline",
			input:    "café\nnaïve\nrésumé",
			expected: "café\nnaïve\nrésumé",
		},
		{
			name:     "mixed line endings",
			input:    "line1\r\nline2\nline3\r",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "preserve internal tabs",
			input:    "column1\tcolumn2\tcolumn3\nvalue1\tvalue2\tvalue3",
			expected: "column1\tcolumn2\tcolumn3\nvalue1\tvalue2\tvalue3",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single line",
			input:    "  single line  ",
			expected: "single line",
		},
		{
			name:     "only newlines",
			input:    "\n\n\n",
			expected: "\n\n\n",
		},
		{
			name:     "control chars mixed with content",
			input:    "hello\x00world\ntest\x1Fline\nfinal",
			expected: "helloworld\ntestline\nfinal",
		},
		{
			name:     "unicode with multiline",
			input:    "  héllo wörld  \n  你好世界  \n  مرحبا بالعالم  ",
			expected: "héllo wörld\n你好世界\nمرحبا بالعالم",
		},
		{
			name:     "emojis multiline",
			input:    "  hello 👋  \n  world 🌍  ",
			expected: "hello 👋\nworld 🌍",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanMultiline(tt.input)
			if result != tt.expected {
				t.Errorf("CleanMultiline(%q) = %q, want %q", tt.input, result, tt.expected)
			}

			// Check that control characters are removed (except newlines and tabs)
			for _, r := range result {
				if unicode.IsControl(r) && r != '\n' && r != '\t' {
					t.Errorf("CleanMultiline(%q) = %q, should not contain control characters other than newlines and tabs", tt.input, result)
					break
				}
			}

			// Check that each line is trimmed
			lines := strings.Split(result, "\n")
			for i, line := range lines {
				if line != strings.TrimSpace(line) {
					t.Errorf("CleanMultiline(%q) line %d = %q, should be trimmed", tt.input, i, line)
				}
			}

			// Check that \r characters are removed
			if strings.Contains(result, "\r") {
				t.Errorf("CleanMultiline(%q) = %q, should not contain carriage returns", tt.input, result)
			}
		})
	}
}

// Benchmark tests
func BenchmarkCleanSingleLine(b *testing.B) {
	input := "  hello\tworld\nwith\rmixed   whitespace  "
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CleanSingleLine(input)
	}
}

func BenchmarkCleanMultiline(b *testing.B) {
	input := "  line1  \n  line2\twith\ttabs  \n  line3  "
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CleanMultiline(input)
	}
}

// Edge case tests for very long strings
func TestCleanSingleLineLongString(t *testing.T) {
	// Create a very long string with mixed whitespace
	var builder strings.Builder
	for i := 0; i < 1000; i++ {
		builder.WriteString("word")
		if i%10 == 0 {
			builder.WriteString("   ")
		} else {
			builder.WriteString(" ")
		}
	}
	input := "  " + builder.String() + "  "

	result := CleanSingleLine(input)

	// Should not contain multiple spaces
	if strings.Contains(result, "  ") {
		t.Error("Long string should not contain multiple consecutive spaces")
	}

	// Should not have leading/trailing whitespace
	if result != strings.TrimSpace(result) {
		t.Error("Long string should not have leading/trailing whitespace")
	}
}

func TestCleanMultilineLongString(t *testing.T) {
	// Create a multiline string with many lines
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("  line %d with content  ", i))
	}
	input := strings.Join(lines, "\n")

	result := CleanMultiline(input)
	resultLines := strings.Split(result, "\n")

	for i, line := range resultLines {
		if line != strings.TrimSpace(line) {
			t.Errorf("Line %d should be trimmed", i)
		}
	}
}

// Table-driven property tests
func TestCleanSingleLineProperties(t *testing.T) {
	properties := []struct {
		name     string
		input    string
		property func(string, string) bool
	}{
		{
			name:  "result should never contain newlines",
			input: "test\nwith\nnewlines",
			property: func(input, result string) bool {
				return !strings.Contains(result, "\n")
			},
		},
		{
			name:  "result should never contain tabs",
			input: "test\twith\ttabs",
			property: func(input, result string) bool {
				return !strings.Contains(result, "\t")
			},
		},
		{
			name:  "result should never have leading whitespace",
			input: "   leading whitespace",
			property: func(input, result string) bool {
				return len(result) == 0 || !unicode.IsSpace(rune(result[0]))
			},
		},
		{
			name:  "result should never have trailing whitespace",
			input: "trailing whitespace   ",
			property: func(input, result string) bool {
				return len(result) == 0 || !unicode.IsSpace(rune(result[len(result)-1]))
			},
		},
	}

	for _, prop := range properties {
		t.Run(prop.name, func(t *testing.T) {
			result := CleanSingleLine(prop.input)
			if !prop.property(prop.input, result) {
				t.Errorf("Property violated for input %q, result %q", prop.input, result)
			}
		})
	}
}
