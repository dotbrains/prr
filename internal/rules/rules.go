package rules

import (
	"context"
	"fmt"
	"os"

	contextpkg "github.com/dotbrains/prr/internal/context"
	"gopkg.in/yaml.v3"
)

// ProjectRules represents project-level review rules from .prr.yaml.
type ProjectRules struct {
	Rules []string `yaml:"rules"`
}

// LoadFromFile reads .prr.yaml from a local file path.
func LoadFromFile(path string) (*ProjectRules, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no rules file — not an error
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return parse(data)
}

// LoadFromReader reads .prr.yaml via a FileReader (e.g. GitHub API).
func LoadFromReader(ctx context.Context, reader contextpkg.FileReader, ref string) (*ProjectRules, error) {
	content, err := reader.ReadFile(ctx, ref, ".prr.yaml")
	if err != nil {
		return nil, nil // non-fatal — file may not exist
	}
	return parse([]byte(content))
}

func parse(data []byte) (*ProjectRules, error) {
	var rules ProjectRules
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("parsing .prr.yaml: %w", err)
	}
	if len(rules.Rules) == 0 {
		return nil, nil
	}
	return &rules, nil
}
