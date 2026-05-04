package cmd

import (
	"fmt"

	"github.com/jvcorredor/bytheway/internal/tag"
)

// validateResearchTag normalizes raw and verifies it is a syntactically
// valid tag, returning the normalized form. An empty raw value yields an
// empty result with no error, signalling "no tag filter."
func validateResearchTag(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	n := tag.Normalize(raw)
	if err := tag.Validate(n); err != nil {
		return "", fmt.Errorf("invalid --tag %q: %w", raw, err)
	}
	return n, nil
}
