// Package config loads and saves user settings for the desktop client.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"bg/internal/game"
	"bg/internal/ollama"
)

const (
	DefaultModel       = "openai/gpt-4o-mini"
	DefaultProvider    = "openrouter"
	DefaultOllamaModel = "llama3.2"
	DefaultMatchTarget = 5
	DefaultTheme       = "persian"
)

// Config is persisted under the user config directory.
type Config struct {
	APIKey      string `json:"api_key"`
	Model       string `json:"model"`
	Provider    string `json:"provider"` // openrouter | ollama
	OllamaURL   string `json:"ollama_url"`
	OllamaModel string `json:"ollama_model"`
	MatchTarget int    `json:"match_target"`
	Theme       string `json:"theme"`
}

// Load reads the config file, then fills blanks from the environment.
func Load() (Config, error) {
	var cfg Config
	path, err := Path()
	if err != nil {
		return applyEnv(cfg), nil
	}
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return Config{}, err
	}
	if err == nil {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return Config{}, err
		}
	}
	return applyEnv(cfg), nil
}

func applyEnv(cfg Config) Config {
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if cfg.Model == "" {
		if v := os.Getenv("OPENROUTER_MODEL"); v != "" {
			cfg.Model = v
		} else {
			cfg.Model = DefaultModel
		}
	}
	if cfg.Provider == "" {
		cfg.Provider = DefaultProvider
	}
	if cfg.OllamaURL == "" {
		if v := os.Getenv("OLLAMA_HOST"); v != "" {
			cfg.OllamaURL = v
		} else {
			cfg.OllamaURL = ollama.DefaultBaseURL
		}
	}
	if cfg.OllamaModel == "" {
		cfg.OllamaModel = DefaultOllamaModel
	}
	if !game.ValidTarget(cfg.MatchTarget) {
		cfg.MatchTarget = DefaultMatchTarget
	}
	switch cfg.Theme {
	case "classic", "persian", "chinese", "modern":
	default:
		cfg.Theme = DefaultTheme
	}
	return cfg
}

// Save writes the config, creating the directory if needed.
func Save(cfg Config) error {
	cfg = applyEnv(cfg)
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

// Path is $XDG_CONFIG_HOME/backgammon/config.json.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "backgammon", "config.json"), nil
}
