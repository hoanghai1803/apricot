package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds all application configuration.
type Config struct {
	AI     AIConfig     `toml:"ai"`
	Server ServerConfig `toml:"server"`
	Feeds  FeedsConfig  `toml:"feeds"`
}

// AIConfig holds AI provider settings.
type AIConfig struct {
	Provider string `toml:"provider"`
	APIKey   string `toml:"api_key"`
	Model    string `toml:"model"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port            int  `toml:"port"`
	AutoOpenBrowser bool `toml:"auto_open_browser"`
}

// FeedsConfig holds RSS feed settings.
type FeedsConfig struct {
	RefreshIntervalMinutes int `toml:"refresh_interval_minutes"`
	MaxArticlesPerFeed     int `toml:"max_articles_per_feed"`
	LookbackDays           int `toml:"lookback_days"`
}

const defaultConfigContent = `[ai]
provider = "anthropic"            # "anthropic" or "openai"
api_key = ""                      # Your API key (or set AI_API_KEY env var)
model = "claude-haiku-4-5"        # See README for supported models

[server]
port = 8080
auto_open_browser = true

[feeds]
refresh_interval_minutes = 60
max_articles_per_feed = 20
lookback_days = 7
`

// Load reads and parses the TOML config from the given path. If the file does
// not exist, it creates a default config file at that path. Environment
// variables override values from the file with highest priority.
func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		if err := createDefault(path); err != nil {
			return nil, fmt.Errorf("creating default config: %w", err)
		}
		slog.Info("created default config file", "path", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	md, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Validate explicitly-set values before applying defaults, so that
	// explicitly writing "port = 0" is an error rather than silently
	// being replaced with the default.
	if err := validateExplicit(&cfg, md); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	applyDefaults(&cfg)
	applyEnvOverrides(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

// createDefault writes the default config content to the given path,
// creating any parent directories as needed.
func createDefault(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(defaultConfigContent), 0o644); err != nil {
		return fmt.Errorf("writing default config: %w", err)
	}
	return nil
}

// validateExplicit checks values that were explicitly set in the TOML file.
// This catches cases like "port = 0" which would otherwise be silently
// replaced by the default value.
func validateExplicit(cfg *Config, md toml.MetaData) error {
	if md.IsDefined("server", "port") {
		if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
			return fmt.Errorf("invalid server.port %d: must be between 1 and 65535", cfg.Server.Port)
		}
	}
	if md.IsDefined("feeds", "lookback_days") {
		if cfg.Feeds.LookbackDays < 1 {
			return fmt.Errorf("invalid feeds.lookback_days %d: must be >= 1", cfg.Feeds.LookbackDays)
		}
	}
	return nil
}

// applyDefaults sets default values for any zero-valued fields.
func applyDefaults(cfg *Config) {
	if cfg.AI.Provider == "" {
		cfg.AI.Provider = "anthropic"
	}
	if cfg.AI.Model == "" {
		cfg.AI.Model = "claude-haiku-4-5"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	// Note: auto_open_browser defaults to true, but TOML parses missing bool
	// as false, so we cannot distinguish "explicitly set to false" from "not
	// set" using a plain bool. The default config file sets it to true, so
	// this is only relevant for hand-edited configs that omit the field.
	// We leave this as-is to respect explicit false values.
	if cfg.Feeds.RefreshIntervalMinutes == 0 {
		cfg.Feeds.RefreshIntervalMinutes = 60
	}
	if cfg.Feeds.MaxArticlesPerFeed == 0 {
		cfg.Feeds.MaxArticlesPerFeed = 20
	}
	if cfg.Feeds.LookbackDays == 0 {
		cfg.Feeds.LookbackDays = 7
	}
}

// applyEnvOverrides applies environment variable overrides. Environment
// variables take highest priority over config file values.
//
// Priority for ai.api_key:
//  1. AI_API_KEY (generic, highest)
//  2. ANTHROPIC_API_KEY (when provider is "anthropic")
//  3. OPENAI_API_KEY (when provider is "openai")
func applyEnvOverrides(cfg *Config) {
	// Apply provider-specific env var first (lower priority).
	switch cfg.AI.Provider {
	case "anthropic":
		if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
			cfg.AI.APIKey = v
		}
	case "openai":
		if v := os.Getenv("OPENAI_API_KEY"); v != "" {
			cfg.AI.APIKey = v
		}
	}

	// AI_API_KEY overrides everything (highest priority).
	if v := os.Getenv("AI_API_KEY"); v != "" {
		cfg.AI.APIKey = v
	}
}

// validate checks that configuration values are within acceptable ranges.
func validate(cfg *Config) error {
	switch cfg.AI.Provider {
	case "anthropic", "openai":
		// valid
	default:
		return fmt.Errorf("invalid ai.provider %q: must be \"anthropic\" or \"openai\"", cfg.AI.Provider)
	}

	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server.port %d: must be between 1 and 65535", cfg.Server.Port)
	}

	if cfg.Feeds.LookbackDays < 1 {
		return fmt.Errorf("invalid feeds.lookback_days %d: must be >= 1", cfg.Feeds.LookbackDays)
	}

	if cfg.AI.APIKey == "" {
		slog.Warn("ai.api_key is empty: set it in the config file or via AI_API_KEY environment variable")
	}

	return nil
}
