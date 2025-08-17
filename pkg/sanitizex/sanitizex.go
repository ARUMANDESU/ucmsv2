package sanitizex

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// CleanSingleLine sanitizes a single-line string by normalizing Unicode, trimming whitespace,
// removing control characters, and collapsing internal whitespace to a single ASCII space.
// It is suitable for fields that should not contain newlines or tabs, such as names or titles.
func CleanSingleLine(s string) string {
	if s == "" {
		return ""
	}
	s = norm.NFC.String(s)
	s = strings.Map(func(r rune) rune {
		if r == '\u007f' || unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	// Collapse internal whitespace to a single ASCII space
	var b strings.Builder
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !space {
				b.WriteByte(' ')
				space = true
			}
		} else {
			b.WriteRune(r)
			space = false
		}
	}
	return b.String()
}

// CleanMultiline sanitizes a multiline string by normalizing Unicode, removing control characters,
// and trimming whitespace from each line. It preserves newlines and tabs, making it suitable
// for fields that may contain multiline text, such as descriptions or comments.
func CleanMultiline(s string) string {
	if s == "" {
		return ""
	}
	s = norm.NFC.String(s)
	// Strip control chars except \n and \t
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if r == '\u007f' || unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	// Normalize line endings and trim trailing spaces per line
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.Join(lines, "\n")
}
