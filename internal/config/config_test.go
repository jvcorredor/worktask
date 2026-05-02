package config

import (
	"os"
	"path/filepath"
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
