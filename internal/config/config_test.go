package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTestConfig is a helper that writes a TOML config file to a temp directory
// and returns its path.
func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test config: %v", err)
	}
	return path
}

func TestLoad_ValidConfig(t *testing.T) {
	content := `
[ai]
provider = "openai"
api_key = "sk-test-key-123"
model = "gpt-4o"

[server]
port = 9090
auto_open_browser = false

[feeds]
refresh_interval_minutes = 30
max_articles_per_feed = 50
lookback_days = 14
`
	path := writeTestConfig(t, content)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%q) unexpected error: %v", path, err)
	}

	// AI config
	if cfg.AI.Provider != "openai" {
		t.Errorf("AI.Provider = %q, want %q", cfg.AI.Provider, "openai")
	}
	if cfg.AI.APIKey != "sk-test-key-123" {
		t.Errorf("AI.APIKey = %q, want %q", cfg.AI.APIKey, "sk-test-key-123")
	}
	if cfg.AI.Model != "gpt-4o" {
		t.Errorf("AI.Model = %q, want %q", cfg.AI.Model, "gpt-4o")
	}

	// Server config
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 9090)
	}
	if cfg.Server.AutoOpenBrowser != false {
		t.Errorf("Server.AutoOpenBrowser = %v, want %v", cfg.Server.AutoOpenBrowser, false)
	}

	// Feeds config
	if cfg.Feeds.RefreshIntervalMinutes != 30 {
		t.Errorf("Feeds.RefreshIntervalMinutes = %d, want %d", cfg.Feeds.RefreshIntervalMinutes, 30)
	}
	if cfg.Feeds.MaxArticlesPerFeed != 50 {
		t.Errorf("Feeds.MaxArticlesPerFeed = %d, want %d", cfg.Feeds.MaxArticlesPerFeed, 50)
	}
	if cfg.Feeds.LookbackDays != 14 {
		t.Errorf("Feeds.LookbackDays = %d, want %d", cfg.Feeds.LookbackDays, 14)
	}
}

func TestLoad_MissingFile_CreatesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%q) unexpected error: %v", path, err)
	}

	// File should have been created.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("default config file not created at %q: %v", path, err)
	}

	// Should have default values.
	if cfg.AI.Provider != "anthropic" {
		t.Errorf("AI.Provider = %q, want %q", cfg.AI.Provider, "anthropic")
	}
	if cfg.AI.Model != "claude-haiku-4-5" {
		t.Errorf("AI.Model = %q, want %q", cfg.AI.Model, "claude-haiku-4-5")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 8080)
	}
	if cfg.Server.AutoOpenBrowser != true {
		t.Errorf("Server.AutoOpenBrowser = %v, want %v", cfg.Server.AutoOpenBrowser, true)
	}
	if cfg.Feeds.RefreshIntervalMinutes != 60 {
		t.Errorf("Feeds.RefreshIntervalMinutes = %d, want %d", cfg.Feeds.RefreshIntervalMinutes, 60)
	}
	if cfg.Feeds.MaxArticlesPerFeed != 20 {
		t.Errorf("Feeds.MaxArticlesPerFeed = %d, want %d", cfg.Feeds.MaxArticlesPerFeed, 20)
	}
	if cfg.Feeds.LookbackDays != 7 {
		t.Errorf("Feeds.LookbackDays = %d, want %d", cfg.Feeds.LookbackDays, 7)
	}
}

func TestLoad_DefaultsApplied(t *testing.T) {
	// Minimal config: only provide required valid provider, let everything
	// else fall through to defaults.
	content := `
[ai]
api_key = "sk-test"

[server]

[feeds]
`
	path := writeTestConfig(t, content)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%q) unexpected error: %v", path, err)
	}

	if cfg.AI.Provider != "anthropic" {
		t.Errorf("AI.Provider = %q, want default %q", cfg.AI.Provider, "anthropic")
	}
	if cfg.AI.Model != "claude-haiku-4-5" {
		t.Errorf("AI.Model = %q, want default %q", cfg.AI.Model, "claude-haiku-4-5")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want default %d", cfg.Server.Port, 8080)
	}
	if cfg.Feeds.RefreshIntervalMinutes != 60 {
		t.Errorf("Feeds.RefreshIntervalMinutes = %d, want default %d", cfg.Feeds.RefreshIntervalMinutes, 60)
	}
	if cfg.Feeds.MaxArticlesPerFeed != 20 {
		t.Errorf("Feeds.MaxArticlesPerFeed = %d, want default %d", cfg.Feeds.MaxArticlesPerFeed, 20)
	}
	if cfg.Feeds.LookbackDays != 7 {
		t.Errorf("Feeds.LookbackDays = %d, want default %d", cfg.Feeds.LookbackDays, 7)
	}
}

func TestLoad_EnvVar_AIAPIKey(t *testing.T) {
	content := `
[ai]
provider = "anthropic"
api_key = "from-config"
`
	path := writeTestConfig(t, content)
	t.Setenv("AI_API_KEY", "from-env-generic")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%q) unexpected error: %v", path, err)
	}

	if cfg.AI.APIKey != "from-env-generic" {
		t.Errorf("AI.APIKey = %q, want %q (AI_API_KEY should override config)", cfg.AI.APIKey, "from-env-generic")
	}
}

func TestLoad_EnvVar_AnthropicAPIKey(t *testing.T) {
	content := `
[ai]
provider = "anthropic"
api_key = "from-config"
`
	path := writeTestConfig(t, content)
	t.Setenv("ANTHROPIC_API_KEY", "from-env-anthropic")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%q) unexpected error: %v", path, err)
	}

	if cfg.AI.APIKey != "from-env-anthropic" {
		t.Errorf("AI.APIKey = %q, want %q (ANTHROPIC_API_KEY should override for anthropic provider)", cfg.AI.APIKey, "from-env-anthropic")
	}
}

func TestLoad_EnvVar_OpenAIAPIKey(t *testing.T) {
	content := `
[ai]
provider = "openai"
api_key = "from-config"
`
	path := writeTestConfig(t, content)
	t.Setenv("OPENAI_API_KEY", "from-env-openai")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%q) unexpected error: %v", path, err)
	}

	if cfg.AI.APIKey != "from-env-openai" {
		t.Errorf("AI.APIKey = %q, want %q (OPENAI_API_KEY should override for openai provider)", cfg.AI.APIKey, "from-env-openai")
	}
}

func TestLoad_EnvVar_AIAPIKey_TakesPrecedence(t *testing.T) {
	content := `
[ai]
provider = "anthropic"
api_key = "from-config"
`
	path := writeTestConfig(t, content)
	t.Setenv("ANTHROPIC_API_KEY", "from-env-anthropic")
	t.Setenv("AI_API_KEY", "from-env-generic")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%q) unexpected error: %v", path, err)
	}

	if cfg.AI.APIKey != "from-env-generic" {
		t.Errorf("AI.APIKey = %q, want %q (AI_API_KEY should take precedence over ANTHROPIC_API_KEY)", cfg.AI.APIKey, "from-env-generic")
	}
}

func TestLoad_InvalidProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
	}{
		{name: "unknown provider", provider: "gemini"},
		{name: "empty after no default", provider: "invalid"},
		{name: "typo", provider: "anth ropic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := `
[ai]
provider = "` + tt.provider + `"
api_key = "sk-test"
`
			path := writeTestConfig(t, content)

			_, err := Load(path)
			if err == nil {
				t.Fatalf("Load(%q) expected error for provider %q, got nil", path, tt.provider)
			}
		})
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{name: "zero", port: "0"},
		{name: "negative", port: "-1"},
		{name: "too high", port: "70000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := `
[ai]
provider = "anthropic"
api_key = "sk-test"

[server]
port = ` + tt.port + `
`
			path := writeTestConfig(t, content)

			_, err := Load(path)
			if err == nil {
				t.Fatalf("Load(%q) expected error for port %s, got nil", path, tt.port)
			}
		})
	}
}

func TestLoad_InvalidLookbackDays(t *testing.T) {
	tests := []struct {
		name string
		days string
	}{
		{name: "zero", days: "0"},
		{name: "negative", days: "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := `
[ai]
provider = "anthropic"
api_key = "sk-test"

[feeds]
lookback_days = ` + tt.days + `
`
			path := writeTestConfig(t, content)

			_, err := Load(path)
			if err == nil {
				t.Fatalf("Load(%q) expected error for lookback_days %s, got nil", path, tt.days)
			}
		})
	}
}

func TestLoad_EmptyAPIKey_NoError(t *testing.T) {
	content := `
[ai]
provider = "anthropic"
api_key = ""
`
	path := writeTestConfig(t, content)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%q) unexpected error: %v (empty api_key should warn, not fail)", path, err)
	}

	if cfg.AI.APIKey != "" {
		t.Errorf("AI.APIKey = %q, want empty string", cfg.AI.APIKey)
	}
}
