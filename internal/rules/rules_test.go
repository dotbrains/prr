package rules

import (
	"context"
	"fmt"
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

// mockFileReader implements context.FileReader for testing.
type mockFileReader struct {
	files map[string]string
}

func (m *mockFileReader) ListFiles(_ context.Context, _, _ string) ([]string, error) {
	return nil, nil
}

func (m *mockFileReader) ReadFile(_ context.Context, _, path string) (string, error) {
	if content, ok := m.files[path]; ok {
		return content, nil
	}
	return "", fmt.Errorf("not found: %s", path)
}

func TestLoadFromReader(t *testing.T) {
	reader := &mockFileReader{
		files: map[string]string{
			".prr.yaml": "rules:\n  - \"wrap errors\"\n  - \"no panics\"\n",
		},
	}

	rules, err := LoadFromReader(context.Background(), reader, "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rules == nil {
		t.Fatal("expected rules, got nil")
	}
	if len(rules.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(rules.Rules))
	}
}

func TestLoadFromReader_NotFound(t *testing.T) {
	reader := &mockFileReader{files: map[string]string{}}

	rules, err := LoadFromReader(context.Background(), reader, "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rules != nil {
		t.Error("expected nil when file not found")
	}
}

func TestLoadFromReader_InvalidYAML(t *testing.T) {
	reader := &mockFileReader{
		files: map[string]string{
			".prr.yaml": "{{invalid",
		},
	}

	_, err := LoadFromReader(context.Background(), reader, "main")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
