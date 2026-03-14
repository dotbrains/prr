package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the top-level prr configuration.
type Config struct {
	DefaultAgent string                `yaml:"default_agent"`
	Agents       map[string]AgentConfig `yaml:"agents"`
	Review       ReviewConfig          `yaml:"review"`
	Output       OutputConfig          `yaml:"output"`
}

// AgentConfig defines a single AI agent provider.
type AgentConfig struct {
	Provider  string `yaml:"provider"`
	Model     string `yaml:"model"`
	APIKeyEnv string `yaml:"api_key_env"`
	MaxTokens int    `yaml:"max_tokens"`
}

// ReviewConfig controls review behavior.
type ReviewConfig struct {
	MaxDiffLines   int      `yaml:"max_diff_lines"`
	IgnorePatterns []string `yaml:"ignore_patterns"`
	SeverityLevels []string `yaml:"severity_levels"`
}

// OutputConfig controls where review output is written.
type OutputConfig struct {
	Dir        string   `yaml:"dir"`
	Severities []string `yaml:"severities"`
}

// CLIProviders are providers that use local CLI binaries and don't need API keys.
var CLIProviders = map[string]bool{
	"claude-cli": true,
	"codex-cli":  true,
}

// DefaultDataDir returns the default data directory for prr output.
// Respects $XDG_DATA_HOME if set, otherwise falls back to ~/.local/share/prr.
func DefaultDataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "prr")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "reviews") // fallback
	}
	return filepath.Join(home, ".local", "share", "prr")
}

// DefaultConfig returns the built-in default configuration.
func DefaultConfig() *Config {
	return &Config{
		DefaultAgent: "claude-cli",
		Agents: map[string]AgentConfig{
			"claude-cli": {
				Provider: "claude-cli",
				Model:    "opus",
			},
			"codex-cli": {
				Provider: "codex-cli",
				Model:    "codex",
			},
			"claude-api": {
				Provider:  "anthropic",
				Model:     "claude-opus-4-20250514",
				APIKeyEnv: "ANTHROPIC_API_KEY",
				MaxTokens: 8192,
			},
			"gpt-api": {
				Provider:  "openai",
				Model:     "gpt-4o",
				APIKeyEnv: "OPENAI_API_KEY",
				MaxTokens: 8192,
			},
		},
		Review: ReviewConfig{
			MaxDiffLines: 10000,
			IgnorePatterns: []string{
				"*.lock",
				"go.sum",
				"package-lock.json",
				"yarn.lock",
				"vendor/**",
				"node_modules/**",
				"*.min.js",
				"*.min.css",
				"*.generated.*",
			},
			SeverityLevels: []string{
				"critical",
				"suggestion",
				"nit",
				"praise",
			},
		},
		Output: OutputConfig{
			Dir: filepath.Join(DefaultDataDir(), "reviews"),
			Severities: []string{
				"critical",
				"suggestion",
				"nit",
				"praise",
			},
		},
	}
}

// ConfigDir returns the prr configuration directory path.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "prr"), nil
}

// ConfigPath returns the full path to the config file.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads the config from disk, falling back to defaults if no file exists.
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return cfg, nil
}

// LoadFrom reads the config from a specific path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return cfg, nil
}

// Save writes the config to disk, creating directories as needed.
func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return SaveTo(cfg, path)
}

// SaveTo writes the config to a specific path.
func SaveTo(cfg *Config, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// Exists checks whether a config file exists on disk.
func Exists() (bool, error) {
	path, err := ConfigPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
