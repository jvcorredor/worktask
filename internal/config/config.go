package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

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
}

type fileConfig struct {
	TasksDir           string `toml:"tasks_dir"`
	Editor             string `toml:"editor"`
	ResearchPromptPath string `toml:"research_prompt_path"`
	WorkingLogPath     string `toml:"working_log_path"`
	ResearchModel      string `toml:"research_model"`
}

func Load() (Config, error) {
	var fc fileConfig
	path := configFilePath()
	if _, err := toml.DecodeFile(path, &fc); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}

	cfg := Config{
		TasksDir:           fc.TasksDir,
		Editor:             fc.Editor,
		ResearchPromptPath: fc.ResearchPromptPath,
		WorkingLogPath:     fc.WorkingLogPath,
		ResearchModel:      fc.ResearchModel,
	}
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

func defaultTasksDir() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "worktask")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "worktask")
}
