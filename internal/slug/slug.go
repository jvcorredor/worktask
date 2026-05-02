package slug

import "strings"

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
