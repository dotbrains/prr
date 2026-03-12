package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultAgent != "claude-cli" {
		t.Errorf("expected default agent claude-cli, got %q", cfg.DefaultAgent)
	}
	if len(cfg.Agents) != 4 {
		t.Errorf("expected 4 agents, got %d", len(cfg.Agents))
	}
	for _, name := range []string{"claude-cli", "codex-cli", "claude-api", "gpt-api"} {
		if _, ok := cfg.Agents[name]; !ok {
			t.Errorf("expected %s agent in defaults", name)
		}
	}
	if cfg.Review.MaxDiffLines != 10000 {
		t.Errorf("expected max_diff_lines 10000, got %d", cfg.Review.MaxDiffLines)
	}
	expectedDir := filepath.Join(DefaultDataDir(), "reviews")
	if cfg.Output.Dir != expectedDir {
		t.Errorf("expected output dir %q, got %q", expectedDir, cfg.Output.Dir)
	}
}

func TestSaveToAndLoadFrom(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := DefaultConfig()
	cfg.DefaultAgent = "test-agent"

	if err := SaveTo(cfg, path); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if loaded.DefaultAgent != "test-agent" {
		t.Errorf("expected default_agent test-agent, got %q", loaded.DefaultAgent)
	}
}

func TestLoadFrom_NonExistent(t *testing.T) {
	cfg, err := LoadFrom("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("expected no error for nonexistent file, got %v", err)
	}
	// Should return defaults
	if cfg.DefaultAgent != "claude-cli" {
		t.Errorf("expected default agent claude-cli, got %q", cfg.DefaultAgent)
	}
}

func TestLoadFrom_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFrom(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestSaveTo_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "config.yaml")

	cfg := DefaultConfig()
	if err := SaveTo(cfg, path); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected config file to be created")
	}
}

func TestAgentConfig_Fields(t *testing.T) {
	cfg := DefaultConfig()

	// CLI provider (no API key)
	cli := cfg.Agents["claude-cli"]
	if cli.Provider != "claude-cli" {
		t.Errorf("expected provider claude-cli, got %q", cli.Provider)
	}
	if cli.Model != "opus" {
		t.Errorf("expected model opus, got %q", cli.Model)
	}

	// API provider
	api := cfg.Agents["claude-api"]
	if api.Provider != "anthropic" {
		t.Errorf("expected provider anthropic, got %q", api.Provider)
	}
	if api.APIKeyEnv != "ANTHROPIC_API_KEY" {
		t.Errorf("expected api_key_env ANTHROPIC_API_KEY, got %q", api.APIKeyEnv)
	}
	if api.MaxTokens != 8192 {
		t.Errorf("expected max_tokens 8192, got %d", api.MaxTokens)
	}
}

func TestReviewConfig_IgnorePatterns(t *testing.T) {
	cfg := DefaultConfig()

	expectedPatterns := []string{
		"*.lock", "go.sum", "package-lock.json", "yarn.lock",
		"vendor/**", "node_modules/**",
		"*.min.js", "*.min.css", "*.generated.*",
	}

	if len(cfg.Review.IgnorePatterns) != len(expectedPatterns) {
		t.Errorf("expected %d ignore patterns, got %d", len(expectedPatterns), len(cfg.Review.IgnorePatterns))
	}
}

func TestReviewConfig_SeverityLevels(t *testing.T) {
	cfg := DefaultConfig()

	expected := []string{"critical", "suggestion", "nit", "praise"}
	if len(cfg.Review.SeverityLevels) != len(expected) {
		t.Errorf("expected %d severity levels, got %d", len(expected), len(cfg.Review.SeverityLevels))
	}
}

func TestConfigDir(t *testing.T) {
	dir, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir == "" {
		t.Error("expected non-empty dir")
	}
}

func TestConfigPath(t *testing.T) {
	path, err := ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
}

func TestLoad_NoFile(t *testing.T) {
	// Point HOME to an empty temp dir so no config file exists.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	// Should return defaults.
	if cfg.DefaultAgent != "claude-cli" {
		t.Errorf("expected default agent claude-cli, got %q", cfg.DefaultAgent)
	}
}

func TestLoad_WithFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Create config file at the expected path.
	configDir := filepath.Join(tmp, ".config", "prr")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("default_agent: my-agent\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultAgent != "my-agent" {
		t.Errorf("expected my-agent, got %q", cfg.DefaultAgent)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	configDir := filepath.Join(tmp, ".config", "prr")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("{{bad"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestSave_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg := DefaultConfig()
	cfg.DefaultAgent = "saved-agent"

	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultAgent != "saved-agent" {
		t.Errorf("expected saved-agent, got %q", loaded.DefaultAgent)
	}
}

func TestExists_False(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	exists, err := Exists()
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected false when no config file")
	}
}

func TestExists_True(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	configDir := filepath.Join(tmp, ".config", "prr")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("default_agent: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	exists, err := Exists()
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected true when config file exists")
	}
}

func TestCLIProviders(t *testing.T) {
	if !CLIProviders["claude-cli"] {
		t.Error("expected claude-cli in CLIProviders")
	}
	if !CLIProviders["codex-cli"] {
		t.Error("expected codex-cli in CLIProviders")
	}
	if CLIProviders["anthropic"] {
		t.Error("anthropic should not be a CLI provider")
	}
}
