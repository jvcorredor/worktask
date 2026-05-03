package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	TasksDir           string
	Editor             string
	ResearchPromptPath string
	WorkingLogPath     string
	// ResearchModel overrides the model used by the headless research agent.
	// Empty means the runner picks its DefaultModel.
	ResearchModel string
	// ResearchExtraTools is appended verbatim to the research agent's
	// --allowed-tools baseline. Validated at Load(); a Config that has
	// escaped this package is known-clean.
	ResearchExtraTools []string
}

type fileConfig struct {
	TasksDir           string   `toml:"tasks_dir"`
	Editor             string   `toml:"editor"`
	ResearchPromptPath string   `toml:"research_prompt_path"`
	WorkingLogPath     string   `toml:"working_log_path"`
	ResearchModel      string   `toml:"research_model"`
	ResearchExtraTools []string `toml:"research_extra_tools"`
}

func Load() (Config, error) {
	var fc fileConfig
	path := configFilePath()
	if _, err := toml.DecodeFile(path, &fc); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}

	if err := ValidateExtraTools(fc.ResearchExtraTools, path); err != nil {
		return Config{}, err
	}

	cfg := Config(fc)
	if cfg.TasksDir == "" {
		cfg.TasksDir = defaultTasksDir()
	}
	if cfg.ResearchPromptPath == "" {
		cfg.ResearchPromptPath = filepath.Join(filepath.Dir(path), "research-prompt.md")
	}
	if cfg.WorkingLogPath == "" {
		cfg.WorkingLogPath = filepath.Join(os.Getenv("HOME"), "workspace", "github.com", "WORKING.md")
	}
	return cfg, nil
}

func (c Config) ResolveEditor() string {
	if c.Editor != "" {
		return c.Editor
	}
	if env := os.Getenv("EDITOR"); env != "" {
		return env
	}
	return "vi"
}

func configFilePath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "worktask", "config.toml")
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "worktask", "config.toml")
}

// ValidateExtraTools enforces the security policy on user-supplied additions
// to the research agent's --allowed-tools list. Because the agent runs
// unattended with --permission-mode=bypassPermissions, the allowlist is the
// security boundary; entries that would let it write to disk or run arbitrary
// shell commands must be rejected before a Config escapes Load().
//
// configPath is the path of the config.toml the entries came from; it is
// included in the error so the user can locate the offending value.
func ValidateExtraTools(extras []string, configPath string) error {
	bareWriteCapable := map[string]bool{
		"Edit":         true,
		"Write":        true,
		"NotebookEdit": true,
		"TodoWrite":    true,
		"Bash":         true,
	}
	for i, entry := range extras {
		if bareWriteCapable[entry] {
			return fmt.Errorf("config: research_extra_tools[%d] = %q is write-capable and cannot be granted to the unattended research agent (in %s)", i, entry, configPath)
		}
		// Bash(*) is the unrestricted-shell escape hatch; Bash(*:...) puts a
		// wildcard binary in front of any subcommand and is equally unsafe.
		if entry == "Bash(*)" || strings.HasPrefix(entry, "Bash(*:") {
			return fmt.Errorf("config: research_extra_tools[%d] = %q permits unrestricted shell execution and cannot be granted to the unattended research agent (in %s)", i, entry, configPath)
		}
	}
	return nil
}

func defaultTasksDir() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "worktask")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "worktask")
}
