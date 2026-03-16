package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".prr.yaml")

	content := `rules:
  - "All errors must be wrapped with fmt.Errorf and %w"
  - "No direct SQL queries outside the repository layer"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	rules, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}
	if rules == nil {
		t.Fatal("expected rules, got nil")
	}
	if len(rules.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(rules.Rules))
	}
}

func TestLoadFromFile_NotFound(t *testing.T) {
	rules, err := LoadFromFile("/nonexistent/.prr.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rules != nil {
		t.Error("expected nil rules for missing file")
	}
}

func TestLoadFromFile_EmptyRules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".prr.yaml")

	if err := os.WriteFile(path, []byte("rules: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rules, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rules != nil {
		t.Error("expected nil for empty rules")
	}
}

func TestLoadFromFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".prr.yaml")

	if err := os.WriteFile(path, []byte("{{invalid\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFromFile(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
