package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type appConfig struct {
	GitRootRealDirs            string   `json:"git_root_real_dirs"`
	ToursDone                  []string `json:"tours_done"`
	DefaultShell               string   `json:"default_shell"`
	PollWorkers                int      `json:"poll_workers"`
	NumEmptyPollBeforeCLIReset int      `json:"num_empty_poll_before_cli_reset"`
	SessionCleanupAfterMinutes int      `json:"session_cleanup_after_minutes"`
	FolderCleanupAfterMinutes  int      `json:"folder_cleanup_after_minutes"`
	PiDefaultModel             string   `json:"pi_default_model"`
	PiDefaultReasoning         string   `json:"pi_default_reasoning"`
	PrimeDefaultModel          string   `json:"prime_default_model"`
	PrimeDefaultReasoning      string   `json:"prime_default_reasoning"`
	MaxImageMB                 int      `json:"max_image_MB"`
	MaxPromptImages            int      `json:"max_prompt_images"`
	OpencodeDefaultModel       string   `json:"opencode_default_model"`
	OpencodeDefaultReasoning   string   `json:"opencode_default_reasoning"`
	ClaudeDefaultModel         string   `json:"claude_default_model"`
	ClaudeDefaultReasoning     string   `json:"claude_default_reasoning"`
	CodexDefaultModel          string   `json:"codex_default_model"`
	CodexDefaultReasoning      string   `json:"codex_default_reasoning"`
	CursorDefaultModel         string   `json:"cursor_default_model"`
	CursorDefaultReasoning     string   `json:"cursor_default_reasoning"`
}

func (c appConfig) validate() error {
	var errs []error
	if c.SessionCleanupAfterMinutes < 1 {
		errs = append(errs, fmt.Errorf("session_cleanup_after_minutes must be at least 1"))
	}
	if c.FolderCleanupAfterMinutes < 1 {
		errs = append(errs, fmt.Errorf("folder_cleanup_after_minutes must be at least 1"))
	}
	if c.PollWorkers < 1 {
		errs = append(errs, fmt.Errorf("poll_workers must be at least 1"))
	}
	if c.MaxImageMB < 1 {
		errs = append(errs, fmt.Errorf("max_image_MB must be at least 1"))
	}
	if c.MaxPromptImages < 1 {
		errs = append(errs, fmt.Errorf("max_prompt_images must be at least 1"))
	}
	return errors.Join(errs...)
}

var defaultConfig = appConfig{NumEmptyPollBeforeCLIReset: 5, PollWorkers: 4, SessionCleanupAfterMinutes: 5, FolderCleanupAfterMinutes: 5, MaxImageMB: 10, MaxPromptImages: 4}

func loadConfig() (appConfig, error) {
	config := defaultConfig
	path, err := configPath()
	if err != nil {
		return config, err
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return config, nil
	}
	if err != nil {
		return config, fmt.Errorf("could not read %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		return config, fmt.Errorf("could not decode %s: %w", path, err)
	}
	if config.MaxImageMB == 0 {
		config.MaxImageMB = defaultConfig.MaxImageMB
	}
	if config.FolderCleanupAfterMinutes == 0 {
		config.FolderCleanupAfterMinutes = defaultConfig.FolderCleanupAfterMinutes
	}
	if config.MaxPromptImages == 0 {
		config.MaxPromptImages = defaultConfig.MaxPromptImages
	}
	defaultConfig = config
	return config, nil
}

func saveConfig(config appConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}
	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode config: %w", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0600); err != nil {
		return fmt.Errorf("could not save config file: %w", err)
	}
	return nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find home directory: %w", err)
	}
	return filepath.Join(home, ".ahweb", "config.json"), nil
}
