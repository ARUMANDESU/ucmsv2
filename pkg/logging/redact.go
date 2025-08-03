package logging

import (
	"strings"
	"unicode/utf8"
)

// RedactEmail shows first 2 runes of the local part and replaces the rest
// with "****", keeping the domain intact. It leaves the input unchanged if:
// - empty
// - malformed (no '@' or '@' at ends)
// - local part has fewer than 3 runes (too short to meaningfully redact)
func RedactEmail(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		// malformed: no '@', or '@' at start/end
		return s
	}

	local, domain := s[:at], s[at+1:]
	if utf8.RuneCountInString(local) < 3 {
		// not enough characters to redact
		return s
	}

	// take first 2 runes from local (rune-safe)
	offset := 0
	for count := 0; count < 2 && offset < len(local); count++ {
		_, size := utf8.DecodeRuneInString(local[offset:])
		offset += size
	}
	prefix := local[:offset]

	return prefix + "****@" + domain
}
