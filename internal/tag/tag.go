// Package tag provides pure functions for normalising, validating, and
// deduplicating task tags. Tags are lowercase alphanumeric strings that may
// contain hyphens, up to 40 characters long.
package tag

import (
	"fmt"
	"strings"
)

const maxLen = 40

// Normalize lowercases and trims leading and trailing whitespace.
func Normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// Validate returns an error if s contains characters outside [a-z0-9-]
// or exceeds maxLen characters.
func Validate(s string) error {
	if len(s) > maxLen {
		return fmt.Errorf("tag: %q exceeds %d characters", s, maxLen)
	}
	for _, r := range s {
		if !isAlnum(r) && r != '-' {
			return fmt.Errorf("tag: %q contains invalid character %q", s, r)
		}
	}
	return nil
}

func isAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// Deduplicate removes duplicate tags, preserving first-occurrence order.
func Deduplicate(tags []string) []string {
	seen := make(map[string]bool, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}
