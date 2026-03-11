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
	if cfg.Output.Dir != "reviews" {
		t.Errorf("expected output dir reviews, got %q", cfg.Output.Dir)
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
	if cli.Model != "sonnet" {
		t.Errorf("expected model sonnet, got %q", cli.Model)
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
