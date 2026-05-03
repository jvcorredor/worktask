package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoad_defaultsWhenNoConfigOrEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := filepath.Join(home, ".local", "share", "worktask")
	if cfg.TasksDir != want {
		t.Errorf("TasksDir = %q; want %q", cfg.TasksDir, want)
	}
}

func TestLoad_xdgDataHomeOverridesDefault(t *testing.T) {
	home := t.TempDir()
	xdgData := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("XDG_CONFIG_HOME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := filepath.Join(xdgData, "worktask")
	if cfg.TasksDir != want {
		t.Errorf("TasksDir = %q; want %q", cfg.TasksDir, want)
	}
}

func TestResolveEditor_configWinsOverEnv(t *testing.T) {
	t.Setenv("EDITOR", "nano")
	cfg := Config{Editor: "code --wait"}
	if got := cfg.ResolveEditor(); got != "code --wait" {
		t.Errorf("ResolveEditor() = %q; want %q", got, "code --wait")
	}
}

func TestResolveEditor_fallsBackToEditorEnv(t *testing.T) {
	t.Setenv("EDITOR", "nano")
	cfg := Config{}
	if got := cfg.ResolveEditor(); got != "nano" {
		t.Errorf("ResolveEditor() = %q; want %q", got, "nano")
	}
}

func TestResolveEditor_fallsBackToViWhenNothingSet(t *testing.T) {
	t.Setenv("EDITOR", "")
	cfg := Config{}
	if got := cfg.ResolveEditor(); got != "vi" {
		t.Errorf("ResolveEditor() = %q; want %q", got, "vi")
	}
}

func TestLoad_researchPromptPathDefaultsToSiblingOfConfig(t *testing.T) {
	home := t.TempDir()
	xdgConfig := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)
	t.Setenv("XDG_DATA_HOME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := filepath.Join(xdgConfig, "worktask", "research-prompt.md")
	if cfg.ResearchPromptPath != want {
		t.Errorf("ResearchPromptPath = %q; want %q", cfg.ResearchPromptPath, want)
	}
}

func TestLoad_researchPromptPathFromConfigFile(t *testing.T) {
	home := t.TempDir()
	xdgConfig := t.TempDir()
	customPrompt := filepath.Join(t.TempDir(), "my-prompt.md")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)
	t.Setenv("XDG_DATA_HOME", "")

	configDir := filepath.Join(xdgConfig, "worktask")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	contents := []byte(`research_prompt_path = "` + customPrompt + `"` + "\n")
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ResearchPromptPath != customPrompt {
		t.Errorf("ResearchPromptPath = %q; want %q", cfg.ResearchPromptPath, customPrompt)
	}
}

func TestLoad_researchModelDefaultsToEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ResearchModel != "" {
		t.Errorf("ResearchModel = %q; want empty (so the runner's DefaultModel applies)", cfg.ResearchModel)
	}
}

func TestLoad_researchModelFromConfigFile(t *testing.T) {
	home := t.TempDir()
	xdgConfig := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)
	t.Setenv("XDG_DATA_HOME", "")

	configDir := filepath.Join(xdgConfig, "worktask")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	contents := []byte(`research_model = "claude-haiku-4-5-20251001"` + "\n")
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ResearchModel != "claude-haiku-4-5-20251001" {
		t.Errorf("ResearchModel = %q; want claude-haiku-4-5-20251001", cfg.ResearchModel)
	}
}

func TestLoad_researchExtraToolsDecodesFromConfig(t *testing.T) {
	home := t.TempDir()
	xdgConfig := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)
	t.Setenv("XDG_DATA_HOME", "")

	configDir := filepath.Join(xdgConfig, "worktask")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	contents := []byte("research_extra_tools = [\"mcp__claude_ai_Slack__slack_read_thread\", \"Bash(gh pr view:*)\"]\n")
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{"mcp__claude_ai_Slack__slack_read_thread", "Bash(gh pr view:*)"}
	if !reflect.DeepEqual(cfg.ResearchExtraTools, want) {
		t.Errorf("ResearchExtraTools = %v; want %v", cfg.ResearchExtraTools, want)
	}
}

func TestValidateExtraTools_acceptsLegitimateExtras(t *testing.T) {
	// Realistic shape: a Slack read MCP plus a pattern-restricted gh Bash entry.
	// The validator's job is only to block write-capable built-ins and
	// unrestricted Bash; everything else (per-MCP safety, gh subcommand choice)
	// is the user's responsibility per the threat model.
	extras := []string{
		"mcp__claude_ai_Slack__slack_read_thread",
		"Bash(gh pr view:*)",
	}
	if err := ValidateExtraTools(extras, "/tmp/config.toml"); err != nil {
		t.Errorf("ValidateExtraTools(%v) = %v; want nil", extras, err)
	}
}

func TestValidateExtraTools_rejectsWriteCapableEntries(t *testing.T) {
	const configPath = "/tmp/worktask/config.toml"
	cases := []struct {
		name  string
		entry string
	}{
		{"bareEdit", "Edit"},
		{"bareWrite", "Write"},
		{"bareNotebookEdit", "NotebookEdit"},
		{"bareTodoWrite", "TodoWrite"},
		{"bareBash", "Bash"},
		{"bashWildcard", "Bash(*)"},
		{"bashWildcardWithSubcommand", "Bash(*:something)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateExtraTools([]string{tc.entry}, configPath)
			if err == nil {
				t.Fatalf("ValidateExtraTools(%q) = nil; want error", tc.entry)
			}
			msg := err.Error()
			if !strings.Contains(msg, tc.entry) {
				t.Errorf("error %q does not mention offending entry %q", msg, tc.entry)
			}
			if !strings.Contains(msg, "0") {
				t.Errorf("error %q does not mention the entry's index 0", msg)
			}
			if !strings.Contains(msg, configPath) {
				t.Errorf("error %q does not mention config path %q", msg, configPath)
			}
		})
	}
}

func TestValidateExtraTools_errorMentionsActualOffendingIndex(t *testing.T) {
	// The bad entry is third (index 2). The message must point at index 2,
	// not 0, so the user can locate it without scanning.
	extras := []string{
		"mcp__claude_ai_Slack__slack_read_thread",
		"Bash(gh pr view:*)",
		"Edit",
	}
	err := ValidateExtraTools(extras, "/tmp/config.toml")
	if err == nil {
		t.Fatalf("ValidateExtraTools = nil; want error")
	}
	if !strings.Contains(err.Error(), "[2]") {
		t.Errorf("error %q does not mention offending index 2", err.Error())
	}
}

func TestValidateExtraTools_acceptsNilAndEmpty(t *testing.T) {
	if err := ValidateExtraTools(nil, "/tmp/config.toml"); err != nil {
		t.Errorf("ValidateExtraTools(nil) = %v; want nil", err)
	}
	if err := ValidateExtraTools([]string{}, "/tmp/config.toml"); err != nil {
		t.Errorf("ValidateExtraTools([]) = %v; want nil", err)
	}
}

func TestLoad_rejectsWriteCapableEntriesInExtraTools(t *testing.T) {
	// Each row writes a config.toml that contains the bad entry, then asserts
	// Load() returns an error that names the entry, its index, and the
	// config.toml path. The validator runs at the system boundary so a Config
	// with a write-capable extra never escapes the package.
	const goodEntry = "mcp__claude_ai_Slack__slack_read_thread"
	cases := []struct {
		name    string
		extras  []string
		bad     string
		wantIdx string // bracketed form expected in the error message
	}{
		{"bareEdit", []string{"Edit"}, "Edit", "[0]"},
		{"bareWrite", []string{goodEntry, "Write"}, "Write", "[1]"},
		{"bareNotebookEdit", []string{"NotebookEdit"}, "NotebookEdit", "[0]"},
		{"bareTodoWrite", []string{"TodoWrite"}, "TodoWrite", "[0]"},
		{"bareBash", []string{"Bash"}, "Bash", "[0]"},
		{"bashWildcard", []string{goodEntry, goodEntry, "Bash(*)"}, "Bash(*)", "[2]"},
		{"bashWildcardWithSubcommand", []string{"Bash(*:rm -rf /)"}, "Bash(*:rm -rf /)", "[0]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			xdgConfig := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("XDG_CONFIG_HOME", xdgConfig)
			t.Setenv("XDG_DATA_HOME", "")

			configDir := filepath.Join(xdgConfig, "worktask")
			if err := os.MkdirAll(configDir, 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			quoted := make([]string, len(tc.extras))
			for i, e := range tc.extras {
				quoted[i] = `"` + e + `"`
			}
			contents := []byte("research_extra_tools = [" + strings.Join(quoted, ", ") + "]\n")
			configPath := filepath.Join(configDir, "config.toml")
			if err := os.WriteFile(configPath, contents, 0o644); err != nil {
				t.Fatalf("write config: %v", err)
			}

			_, err := Load()
			if err == nil {
				t.Fatalf("Load() = nil error; want one rejecting %q", tc.bad)
			}
			msg := err.Error()
			if !strings.Contains(msg, tc.bad) {
				t.Errorf("error %q does not mention offending entry %q", msg, tc.bad)
			}
			if !strings.Contains(msg, tc.wantIdx) {
				t.Errorf("error %q does not mention index %s", msg, tc.wantIdx)
			}
			if !strings.Contains(msg, configPath) {
				t.Errorf("error %q does not mention config path %q", msg, configPath)
			}
		})
	}
}

func TestLoad_researchExtraToolsAbsentYieldsNil(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ResearchExtraTools != nil {
		t.Errorf("ResearchExtraTools = %v; want nil when key is absent", cfg.ResearchExtraTools)
	}
}

func TestLoad_configFileOverridesEnv(t *testing.T) {
	home := t.TempDir()
	xdgConfig := t.TempDir()
	xdgData := t.TempDir()
	customTasksDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)
	t.Setenv("XDG_DATA_HOME", xdgData)

	configDir := filepath.Join(xdgConfig, "worktask")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	contents := []byte(`tasks_dir = "` + customTasksDir + `"` + "\n")
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TasksDir != customTasksDir {
		t.Errorf("TasksDir = %q; want %q", cfg.TasksDir, customTasksDir)
	}
}
