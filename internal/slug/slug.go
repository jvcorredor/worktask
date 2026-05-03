// Package slug derives the human-readable component of a task filename
// from a free-form description.
//
// Slugs are lowercase, contain only ASCII letters, digits, and single
// hyphens, never start or end with a hyphen, and are truncated at the last
// hyphen before maxLen so the result reads as whole words.
package slug

import "strings"

// Generate returns a filesystem-safe slug derived from description, no
// longer than maxLen bytes. The slug is lowercased; any run of characters
// that are not ASCII letters or digits collapses to a single hyphen, and
// leading or trailing hyphens are trimmed. When the slug exceeds maxLen,
// it is cut at the last hyphen before the limit so the result ends on a
// word boundary; if there is no such hyphen, it is hard-truncated.
func Generate(description string, maxLen int) string {
	var b strings.Builder
	b.Grow(len(description))
	prevHyphen := true
	for _, r := range strings.ToLower(description) {
		if isAlnum(r) {
			b.WriteRune(r)
			prevHyphen = false
			continue
		}
		if !prevHyphen {
			b.WriteByte('-')
			prevHyphen = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if len(s) <= maxLen {
		return s
	}
	if i := strings.LastIndex(s[:maxLen], "-"); i >= 0 {
		return strings.TrimRight(s[:i], "-")
	}
	return strings.TrimRight(s[:maxLen], "-")
}

func isAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}
