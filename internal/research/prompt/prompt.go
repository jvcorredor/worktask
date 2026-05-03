// Package prompt loads and renders the research prompt template
// passed to the headless claude agent.
package prompt

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"text/template"

	"github.com/jvcorredor/worktask/internal/task"
)

//go:embed prompt.md
var defaultTemplate string

// Load returns the override template at overridePath if the file exists,
// otherwise returns the embedded default.
func Load(overridePath string) (string, error) {
	if overridePath == "" {
		return defaultTemplate, nil
	}
	data, err := os.ReadFile(overridePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return defaultTemplate, nil
		}
		return "", fmt.Errorf("prompt: read override %s: %w", overridePath, err)
	}
	return string(data), nil
}

// Render parses tmpl as a text/template and executes it against t,
// returning the rendered prompt. Parse and execution errors are wrapped
// with a "prompt:" prefix so callers can distinguish template failures
// from unrelated errors.
func Render(tmpl string, t task.Task) (string, error) {
	parsed, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("prompt: parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := parsed.Execute(&buf, t); err != nil {
		return "", fmt.Errorf("prompt: execute template: %w", err)
	}
	return buf.String(), nil
}
