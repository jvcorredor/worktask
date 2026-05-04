package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jvcorredor/bytheway/internal/task"
)

func TestLoad_emptyOverrideReturnsEmbeddedDefault(t *testing.T) {
	got, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\"): %v", err)
	}
	if got == "" {
		t.Errorf("embedded default is empty")
	}
	if !strings.Contains(got, "{{.Body}}") {
		t.Errorf("embedded default missing {{.Body}} placeholder; got:\n%s", got)
	}
}

func TestLoad_overridePathBeatsEmbedded(t *testing.T) {
	dir := t.TempDir()
	override := filepath.Join(dir, "research-prompt.md")
	contents := "OVERRIDE TEMPLATE\n{{.Body}}\n"
	if err := os.WriteFile(override, []byte(contents), 0o644); err != nil {
		t.Fatalf("write override: %v", err)
	}

	got, err := Load(override)
	if err != nil {
		t.Fatalf("Load(override): %v", err)
	}
	if got != contents {
		t.Errorf("Load(override) = %q; want %q", got, contents)
	}
}

func TestLoad_missingOverrideFallsBackToEmbedded(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.md")
	got, err := Load(missing)
	if err != nil {
		t.Fatalf("Load(missing): %v", err)
	}
	if got == "" {
		t.Errorf("expected embedded fallback, got empty")
	}
}

func TestEmbeddedDefault_namesNoInstallSpecificMCPsOrGhPatterns(t *testing.T) {
	// The shipped binary cannot assume any particular MCP server is
	// installed. Naming `mcp__claude_ai_*`, `mcp__unblocked__*`,
	// `mcp__context7__*`, or `Bash(gh ...)` identifiers in the embedded
	// prompt would push the agent toward tools a fresh-install user does
	// not have. Claude Code injects the actually-available tools into its
	// system prompt, so the agent discovers them without `btw`
	// naming them.
	got, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\"): %v", err)
	}
	forbidden := []string{
		"mcp__claude_ai_",
		"mcp__unblocked__",
		"mcp__context7__",
		"Bash(gh",
	}
	for _, sub := range forbidden {
		if strings.Contains(got, sub) {
			t.Errorf("embedded prompt must not name install-specific identifier %q", sub)
		}
	}
}

func TestEmbeddedDefault_retainsAbstractAnchorGuidance(t *testing.T) {
	// The prompt should still steer the agent to the right kind of tool
	// for each anchor type, just abstractly. Loss of this guidance would
	// degrade research quality in lockstep with the identifier strip.
	got, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\"): %v", err)
	}
	lower := strings.ToLower(got)
	mustMention := []string{"slack", "jira", "datadog", "library", "gh "}
	for _, sub := range mustMention {
		if !strings.Contains(lower, sub) {
			t.Errorf("embedded prompt missing abstract anchor guidance for %q", sub)
		}
	}
}

func TestRender_substitutesBodyVerbatim(t *testing.T) {
	tmpl := "Research this task body:\n{{.Body}}\n--end--\n"
	tk := task.Task{
		ID:      "abcdef12",
		Created: time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC),
		Body:    "investigate slack thread\nlink: https://slack.com/x/y\n",
	}

	got, err := Render(tmpl, tk)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	want := "Research this task body:\ninvestigate slack thread\nlink: https://slack.com/x/y\n\n--end--\n"
	if got != want {
		t.Errorf("Render output:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	if !strings.Contains(got, "investigate slack thread") {
		t.Errorf("Render output missing body line: %q", got)
	}
}
