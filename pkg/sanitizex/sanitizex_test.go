package sanitizex

import (
	"fmt"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
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

// Edge case tests for very long strings
func TestCleanSingleLineLongString(t *testing.T) {
	// Create a very long string with mixed whitespace
	var builder strings.Builder
	for i := range 1000 {
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
	for i := range 100 {
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

func TestDeduplicateSlice(t *testing.T) {
	t.Parallel()

	// Cleaning functions for testing
	trimSpace := func(s string) string {
		return strings.TrimSpace(s)
	}

	toLower := func(s string) string {
		return strings.ToLower(s)
	}

	toUpper := func(s string) string {
		return strings.ToUpper(s)
	}

	removeSpecial := func(s string) string {
		s = strings.ReplaceAll(s, "-", "")
		s = strings.ReplaceAll(s, "_", "")
		return s
	}

	// Combined functions for backward compatibility
	trimLower := func(s string) string {
		return strings.ToLower(strings.TrimSpace(s))
	}

	identity := func(s string) string {
		return s
	}

	t.Run("string slices with identity function", func(t *testing.T) {
		tests := []struct {
			name     string
			input    []string
			expected []string
		}{
			{
				name:     "no duplicates",
				input:    []string{"apple", "banana", "cherry"},
				expected: []string{"apple", "banana", "cherry"},
			},
			{
				name:     "exact duplicates",
				input:    []string{"apple", "banana", "apple", "cherry"},
				expected: []string{"apple", "banana", "cherry"},
			},
			{
				name:     "empty slice",
				input:    []string{},
				expected: []string{},
			},
			{
				name:     "single element",
				input:    []string{"single"},
				expected: []string{"single"},
			},
			{
				name:     "all same elements",
				input:    []string{"same", "same", "same"},
				expected: []string{"same"},
			},
			{
				name:     "empty strings",
				input:    []string{"", "apple", "", "banana"},
				expected: []string{"", "apple", "banana"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := DeduplicateSlice(tt.input, identity)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("string slices with trimLower function", func(t *testing.T) {
		tests := []struct {
			name     string
			input    []string
			expected []string
		}{
			{
				name:     "case insensitive duplicates",
				input:    []string{"Apple", "BANANA", "apple", "Cherry"},
				expected: []string{"apple", "banana", "cherry"},
			},
			{
				name:     "whitespace trimming creates duplicates",
				input:    []string{"  apple  ", "banana", "apple", "  cherry  "},
				expected: []string{"apple", "banana", "cherry"},
			},
			{
				name:     "mixed case and whitespace",
				input:    []string{"  APPLE  ", "banana", "  apple  ", "BANANA"},
				expected: []string{"apple", "banana"},
			},
			{
				name:     "only whitespace differences",
				input:    []string{"test", "  test  ", "   test   "},
				expected: []string{"test"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := DeduplicateSlice(tt.input, trimLower)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("integer slices", func(t *testing.T) {
		intSlice := []int{1, 2, 3, 2, 4}
		result := DeduplicateSlice(intSlice, identity)
		expected := []int{1, 2, 3, 4}
		assert.Equal(t, expected, result)
	})

	t.Run("complex cleaning function", func(t *testing.T) {
		complexClean := func(s string) string {
			s = strings.TrimSpace(s)
			s = strings.ToLower(s)
			s = strings.ReplaceAll(s, "-", "")
			s = strings.ReplaceAll(s, "_", "")
			return s
		}

		tests := []struct {
			name     string
			input    []string
			expected []string
		}{
			{
				name:     "normalize different formats",
				input:    []string{"user-name", "user_name", "USERNAME", "  User-Name  "},
				expected: []string{"username"},
			},
			{
				name:     "mixed separators",
				input:    []string{"hello-world", "hello_world", "HELLO-WORLD", "helloworld"},
				expected: []string{"helloworld"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := DeduplicateSlice(tt.input, complexClean)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("nil slice", func(t *testing.T) {
			var nilSlice []string
			result := DeduplicateSlice(nilSlice, identity)
			assert.Nil(t, result)
		})

		t.Run("large slice performance", func(t *testing.T) {
			largeInput := make([]string, 1000)
			for i := range 1000 {
				largeInput[i] = fmt.Sprintf("item%d", i%100)
			}
			result := DeduplicateSlice(largeInput, identity)
			assert.Len(t, result, 100)
		})

		t.Run("cleaning function returns empty", func(t *testing.T) {
			alwaysEmpty := func(s string) string {
				return ""
			}
			input := []string{"different", "values", "here"}
			result := DeduplicateSlice(input, alwaysEmpty)
			assert.Equal(t, []string{""}, result)
		})

		t.Run("unicode normalization", func(t *testing.T) {
			unicodeClean := func(s string) string {
				return strings.ToLower(strings.TrimSpace(s))
			}
			input := []string{"Café", "café", "CAFÉ", "  café  "}
			result := DeduplicateSlice(input, unicodeClean)
			assert.Equal(t, []string{"café"}, result)
		})
	})

	t.Run("order preservation", func(t *testing.T) {
		input := []string{"first", "second", "first", "third", "second"}
		result := DeduplicateSlice(input, identity)
		expected := []string{"first", "second", "third"}
		assert.Equal(t, expected, result)
	})

	t.Run("different comparable types", func(t *testing.T) {
		t.Run("float64 slice", func(t *testing.T) {
			floatInput := []float64{1.1, 2.2, 1.1, 3.3}
			floatResult := DeduplicateSlice(floatInput, identity)
			assert.Equal(t, []float64{1.1, 2.2, 3.3}, floatResult)
		})

		t.Run("byte slice", func(t *testing.T) {
			byteInput := []byte{0x1, 0x2, 0x1, 0x3}
			byteResult := DeduplicateSlice(byteInput, identity)
			assert.Equal(t, []byte{0x1, 0x2, 0x3}, byteResult)
		})
	})

	t.Run("multiple opts - no opts", func(t *testing.T) {
		input := []string{"  Apple  ", "BANANA", "apple", "Cherry"}
		result := DeduplicateSlice(input)
		expected := []string{"  Apple  ", "BANANA", "apple", "Cherry"}
		assert.Equal(t, expected, result, "No opts should preserve original strings")
	})

	t.Run("multiple opts - single opt", func(t *testing.T) {
		input := []string{"  Apple  ", "banana", "  apple  ", "Cherry"}
		result := DeduplicateSlice(input, trimSpace)
		expected := []string{"Apple", "banana", "apple", "Cherry"}
		assert.Equal(t, expected, result, "Single opt should apply trimming")
	})

	t.Run("multiple opts - chained transformation", func(t *testing.T) {
		tests := []struct {
			name     string
			input    []string
			opts     []StringTransformFunc
			expected []string
		}{
			{
				name:     "trim then lowercase",
				input:    []string{"  APPLE  ", "banana", "  Apple  ", "CHERRY"},
				opts:     []StringTransformFunc{trimSpace, toLower},
				expected: []string{"apple", "banana", "cherry"},
			},
			{
				name:     "lowercase then trim",
				input:    []string{"  APPLE  ", "banana", "  Apple  ", "CHERRY"},
				opts:     []StringTransformFunc{toLower, trimSpace},
				expected: []string{"apple", "banana", "cherry"},
			},
			{
				name:     "trim, lowercase, remove special",
				input:    []string{"  USER-NAME  ", "user_name", "  User-Name  ", "other"},
				opts:     []StringTransformFunc{trimSpace, toLower, removeSpecial},
				expected: []string{"username", "other"},
			},
			{
				name:     "uppercase then remove special",
				input:    []string{"user-name", "USER_NAME", "username", "test-case"},
				opts:     []StringTransformFunc{toUpper, removeSpecial},
				expected: []string{"USERNAME", "TESTCASE"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := DeduplicateSlice(tt.input, tt.opts...)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("multiple opts - complex combinations", func(t *testing.T) {
		// Multiple cleaning functions in different orders
		addPrefix := func(s string) string {
			return "clean_" + s
		}

		removeVowels := func(s string) string {
			vowels := "aeiouAEIOU"
			result := s
			for _, v := range vowels {
				result = strings.ReplaceAll(result, string(v), "")
			}
			return result
		}

		tests := []struct {
			name     string
			input    []string
			opts     []StringTransformFunc
			expected []string
		}{
			{
				name:     "prefix then remove vowels",
				input:    []string{"hello", "world", "hello"},
				opts:     []StringTransformFunc{addPrefix, removeVowels},
				expected: []string{"cln_hll", "cln_wrld"},
			},
			{
				name:     "remove vowels then prefix",
				input:    []string{"hello", "world", "hello"},
				opts:     []StringTransformFunc{removeVowels, addPrefix},
				expected: []string{"clean_hll", "clean_wrld"},
			},
			{
				name:     "trim, lowercase, remove special, remove vowels",
				input:    []string{"  HELLO-WORLD  ", "hello_world", "  Hello-World  "},
				opts:     []StringTransformFunc{trimSpace, toLower, removeSpecial, removeVowels},
				expected: []string{"hllwrld"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := DeduplicateSlice(tt.input, tt.opts...)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("multiple opts - edge cases", func(t *testing.T) {
		t.Run("many opts same function", func(t *testing.T) {
			input := []string{"  HELLO  ", "hello", "  hello  "}
			// Apply toLower multiple times (should be idempotent)
			result := DeduplicateSlice(input, toLower, toLower, toLower, trimSpace)
			expected := []string{"hello"}
			assert.Equal(t, expected, result)
		})

		t.Run("conflicting opts", func(t *testing.T) {
			input := []string{"hello", "HELLO", "Hello"}
			// Apply toLower then toUpper (last one wins)
			result := DeduplicateSlice(input, toLower, toUpper)
			expected := []string{"HELLO"}
			assert.Equal(t, expected, result)
		})

		t.Run("opts that create empty strings", func(t *testing.T) {
			removeAll := func(s string) string {
				return ""
			}
			input := []string{"hello", "world", "test"}
			result := DeduplicateSlice(input, removeAll)
			expected := []string{""}
			assert.Equal(t, expected, result)
		})

		t.Run("opts with nil function behavior", func(t *testing.T) {
			input := []string{"hello", "world", "hello"}
			// Test with empty opts slice
			result := DeduplicateSlice(input)
			expected := []string{"hello", "world"}
			assert.Equal(t, expected, result)
		})
	})

	t.Run("multiple opts - performance with large slices", func(t *testing.T) {
		largeInput := make([]string, 1000)
		for i := range 1000 {
			largeInput[i] = fmt.Sprintf("  ITEM-%d  ", i%50) // 50 unique items, 20 copies each
		}

		result := DeduplicateSlice(largeInput, trimSpace, toLower, removeSpecial)
		assert.Len(t, result, 50, "Should have 50 unique cleaned items")

		// Verify all results are properly cleaned
		for _, item := range result {
			assert.Equal(t, strings.TrimSpace(item), item, "Should be trimmed")
			assert.Equal(t, strings.ToLower(item), item, "Should be lowercase")
			assert.NotContains(t, item, "-", "Should not contain special chars")
		}
	})

	t.Run("multiple opts - unicode handling", func(t *testing.T) {
		normalizeUnicode := func(s string) string {
			return strings.ToLower(s)
		}

		input := []string{"Café", "CAFÉ", "café", "Naïve", "NAÏVE"}
		result := DeduplicateSlice(input, trimSpace, normalizeUnicode)
		expected := []string{"café", "naïve"}
		assert.Equal(t, expected, result)
	})

	t.Run("multiple opts - numeric types ignore opts", func(t *testing.T) {
		// Integer slices should work regardless of StringTransformFunc opts
		intInput := []int{1, 2, 3, 2, 4, 1}
		intResult := DeduplicateSlice(intInput, trimSpace, toLower, toUpper)
		expected := []int{1, 2, 3, 4}
		assert.Equal(t, expected, intResult, "Numeric types should ignore string transform opts")

		floatInput := []float64{1.1, 2.2, 1.1, 3.3}
		floatResult := DeduplicateSlice(floatInput, trimSpace, removeSpecial)
		expectedFloat := []float64{1.1, 2.2, 3.3}
		assert.Equal(t, expectedFloat, floatResult, "Float types should ignore string transform opts")
	})
}

// Benchmark tests
func BenchmarkCleanSingleLine(b *testing.B) {
	input := "  hello\tworld\nwith\rmixed   whitespace  "
	for b.Loop() {
		CleanSingleLine(input)
	}
}

func BenchmarkCleanMultiline(b *testing.B) {
	input := "  line1  \n  line2\twith\ttabs  \n  line3  "
	for b.Loop() {
		CleanMultiline(input)
	}
}

func BenchmarkDeduplicateSlice(b *testing.B) {
	trimSpace := func(s string) string {
		return strings.TrimSpace(s)
	}

	toLower := func(s string) string {
		return strings.ToLower(s)
	}

	b.Run("string slice no opts", func(b *testing.B) {
		input := []string{"apple", "banana", "cherry", "apple", "date", "banana"}
		b.ResetTimer()
		for b.Loop() {
			DeduplicateSlice(input)
		}
	})

	b.Run("string slice single opt", func(b *testing.B) {
		input := []string{"  apple  ", "banana", "  cherry  ", "apple", "  date  "}
		b.ResetTimer()
		for b.Loop() {
			DeduplicateSlice(input, trimSpace)
		}
	})

	b.Run("string slice multiple opts", func(b *testing.B) {
		input := []string{"  APPLE  ", "banana", "  Cherry  ", "APPLE", "  DATE  "}
		b.ResetTimer()
		for b.Loop() {
			DeduplicateSlice(input, trimSpace, toLower)
		}
	})

	b.Run("large string slice no opts", func(b *testing.B) {
		input := make([]string, 10000)
		for i := range input {
			input[i] = fmt.Sprintf("item%d", i%1000) // 1000 unique items, 10 copies each
		}
		b.ResetTimer()
		for b.Loop() {
			DeduplicateSlice(input)
		}
	})

	b.Run("large string slice with opts", func(b *testing.B) {
		input := make([]string, 10000)
		for i := range input {
			input[i] = fmt.Sprintf("  ITEM-%d  ", i%1000)
		}
		b.ResetTimer()
		for b.Loop() {
			DeduplicateSlice(input, trimSpace, toLower)
		}
	})

	b.Run("int slice", func(b *testing.B) {
		input := make([]int, 10000)
		for i := range input {
			input[i] = i % 1000
		}
		b.ResetTimer()
		for b.Loop() {
			DeduplicateSlice(input)
		}
	})

	b.Run("worst case many duplicates", func(b *testing.B) {
		input := make([]string, 10000)
		for i := range input {
			input[i] = "same"
		}
		b.ResetTimer()
		for b.Loop() {
			DeduplicateSlice(input)
		}
	})
}
