package context

import (
	"context"
	"fmt"
	"testing"

	"github.com/dotbrains/prr/internal/agent"
)

// mockFileReader implements the FileReader interface for testing.
type mockFileReader struct {
	files map[string][]string // ref:dir → list of paths
	data  map[string]string   // ref:path → content
	errs  map[string]error    // ref:key → error
}

func newMockFileReader() *mockFileReader {
	return &mockFileReader{
		files: make(map[string][]string),
		data:  make(map[string]string),
		errs:  make(map[string]error),
	}
}

func (m *mockFileReader) ListFiles(_ context.Context, ref, dir string) ([]string, error) {
	key := ref + ":" + dir
	if err, ok := m.errs[key]; ok {
		return nil, err
	}
	if files, ok := m.files[key]; ok {
		return files, nil
	}
	return nil, nil
}

func (m *mockFileReader) ReadFile(_ context.Context, ref, path string) (string, error) {
	key := ref + ":" + path
	if err, ok := m.errs[key]; ok {
		return "", err
	}
	if data, ok := m.data[key]; ok {
		return data, nil
	}
	return "", fmt.Errorf("file not found: %s", key)
}

func TestCollectContext_Basic(t *testing.T) {
	mock := newMockFileReader()
	mock.files["main:src"] = []string{"src/handler.go", "src/auth.go", "src/utils.go"}
	mock.data["main:src/auth.go"] = "package src\n\nfunc Auth() {}\n"
	mock.data["main:src/utils.go"] = "package src\n\nfunc Utils() {}\n"

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), mock, "main", files, 2000)

	if len(result) != 2 {
		t.Fatalf("expected 2 context files, got %d", len(result))
	}
	if result[0].Path != "src/auth.go" {
		t.Errorf("expected src/auth.go, got %s", result[0].Path)
	}
	if result[1].Path != "src/utils.go" {
		t.Errorf("expected src/utils.go, got %s", result[1].Path)
	}
}

func TestCollectContext_SkipsChangedFiles(t *testing.T) {
	mock := newMockFileReader()
	mock.files["main:src"] = []string{"src/handler.go", "src/auth.go"}
	mock.data["main:src/auth.go"] = "package src\n"

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), mock, "main", files, 2000)

	if len(result) != 1 {
		t.Fatalf("expected 1 context file (handler.go excluded), got %d", len(result))
	}
	if result[0].Path != "src/auth.go" {
		t.Errorf("expected src/auth.go, got %s", result[0].Path)
	}
}

func TestCollectContext_SkipsBinaryExtensions(t *testing.T) {
	mock := newMockFileReader()
	mock.files["main:src"] = []string{"src/handler.go", "src/logo.png", "src/data.bin"}

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), mock, "main", files, 2000)

	if len(result) != 0 {
		t.Errorf("expected 0 context files (all skipped), got %d", len(result))
	}
}

func TestCollectContext_SkipsVendoredPaths(t *testing.T) {
	mock := newMockFileReader()
	mock.files["main:vendor"] = []string{"vendor/lib.go"}

	files := []agent.FileDiff{
		{Path: "vendor/dep.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), mock, "main", files, 2000)

	if len(result) != 0 {
		t.Errorf("expected 0 context files (vendored skipped), got %d", len(result))
	}
}

func TestCollectContext_SkipsTestFilesUnlessPRHasTests(t *testing.T) {
	mock := newMockFileReader()
	mock.files["main:src"] = []string{"src/handler.go", "src/handler_test.go", "src/auth.go"}
	mock.data["main:src/auth.go"] = "package src\n"

	// No test files in the PR
	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), mock, "main", files, 2000)

	if len(result) != 1 {
		t.Fatalf("expected 1 context file, got %d", len(result))
	}
	if result[0].Path != "src/auth.go" {
		t.Errorf("expected src/auth.go, got %s", result[0].Path)
	}
}

func TestCollectContext_IncludesTestFilesWhenPRHasTests(t *testing.T) {
	mock := newMockFileReader()
	mock.files["main:src"] = []string{"src/handler.go", "src/handler_test.go", "src/auth_test.go"}
	mock.data["main:src/auth_test.go"] = "package src\n\nfunc TestAuth(t *testing.T) {}\n"

	// PR modifies a test file
	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
		{Path: "src/handler_test.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), mock, "main", files, 2000)

	if len(result) != 1 {
		t.Fatalf("expected 1 context file (auth_test.go), got %d", len(result))
	}
	if result[0].Path != "src/auth_test.go" {
		t.Errorf("expected src/auth_test.go, got %s", result[0].Path)
	}
}

func TestCollectContext_RespectsMaxLines(t *testing.T) {
	mock := newMockFileReader()
	mock.files["main:src"] = []string{"src/handler.go", "src/big.go", "src/small.go"}
	mock.data["main:src/big.go"] = "line1\nline2\nline3\nline4\nline5\n"
	mock.data["main:src/small.go"] = "a\n"

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	// Only 3 lines allowed — big.go won't fully fit, remaining too small to truncate
	result := CollectContext(context.Background(), mock, "main", files, 3)

	if len(result) != 0 {
		t.Errorf("expected 0 files (big.go exceeds budget, remaining too small to truncate), got %d", len(result))
	}
}

func TestCollectContext_ZeroMaxLines(t *testing.T) {
	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), newMockFileReader(), "main", files, 0)
	if result != nil {
		t.Errorf("expected nil for maxLines=0, got %v", result)
	}
}

func TestCollectContext_NegativeMaxLines(t *testing.T) {
	result := CollectContext(context.Background(), newMockFileReader(), "main", nil, -1)
	if result != nil {
		t.Errorf("expected nil for maxLines=-1, got %v", result)
	}
}

func TestCollectContext_ListFilesError(t *testing.T) {
	mock := newMockFileReader()
	mock.errs["main:src"] = fmt.Errorf("not found")

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), mock, "main", files, 2000)
	if len(result) != 0 {
		t.Errorf("expected 0 files on list error, got %d", len(result))
	}
}

func TestCollectContext_ReadFileError(t *testing.T) {
	mock := newMockFileReader()
	mock.files["main:src"] = []string{"src/handler.go", "src/auth.go"}
	mock.errs["main:src/auth.go"] = fmt.Errorf("binary file")

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), mock, "main", files, 2000)
	if len(result) != 0 {
		t.Errorf("expected 0 files on read error, got %d", len(result))
	}
}

func TestUniqueDirs(t *testing.T) {
	files := []agent.FileDiff{
		{Path: "src/handler.go"},
		{Path: "src/auth.go"},
		{Path: "pkg/util.go"},
		{Path: "src/router.go"},
	}

	dirs := uniqueDirs(files)
	if len(dirs) != 2 {
		t.Fatalf("expected 2 unique dirs, got %d: %v", len(dirs), dirs)
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"src/handler_test.go", true},
		{"src/handler.test.ts", true},
		{"src/handler.test.js", true},
		{"src/handler.spec.ts", true},
		{"src/__tests__/handler.js", true},
		{"test_handler.py", true},
		{"src/handler.go", false},
		{"src/main.ts", false},
	}

	for _, tt := range tests {
		got := isTestFile(tt.path)
		if got != tt.want {
			t.Errorf("isTestFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestShouldSkipPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"vendor/lib.go", true},
		{"node_modules/pkg/index.js", true},
		{".git/config", true},
		{"dist/bundle.js", true},
		{"build/output.js", true},
		{"src/handler.go", false},
		{"internal/config/config.go", false},
	}

	for _, tt := range tests {
		got := shouldSkipPath(tt.path)
		if got != tt.want {
			t.Errorf("shouldSkipPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestCollectFileContents_Basic(t *testing.T) {
	mock := newMockFileReader()
	mock.data["feature:src/handler.go"] = "package main\nfunc Handler() {}\n"
	mock.data["feature:src/auth.go"] = "package main\nfunc Auth() {}\n"

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
		{Path: "src/auth.go", Status: "added"},
	}

	result := CollectFileContents(context.Background(), mock, "feature", files)

	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result["src/handler.go"] == "" {
		t.Error("expected content for src/handler.go")
	}
	if result["src/auth.go"] == "" {
		t.Error("expected content for src/auth.go")
	}
}

func TestCollectFileContents_SkipsDeleted(t *testing.T) {
	mock := newMockFileReader()
	mock.data["feature:src/handler.go"] = "package main\n"

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
		{Path: "src/removed.go", Status: "deleted"},
	}

	result := CollectFileContents(context.Background(), mock, "feature", files)

	if len(result) != 1 {
		t.Fatalf("expected 1 entry (deleted skipped), got %d", len(result))
	}
	if _, ok := result["src/removed.go"]; ok {
		t.Error("deleted file should not be in results")
	}
}

func TestCollectFileContents_ReadError(t *testing.T) {
	mock := newMockFileReader()
	mock.errs["feature:src/handler.go"] = fmt.Errorf("permission denied")

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	result := CollectFileContents(context.Background(), mock, "feature", files)

	if len(result) != 0 {
		t.Errorf("expected 0 entries on read error, got %d", len(result))
	}
}

func TestCollectFileContents_Empty(t *testing.T) {
	result := CollectFileContents(context.Background(), newMockFileReader(), "feature", nil)
	if len(result) != 0 {
		t.Errorf("expected 0 entries for nil files, got %d", len(result))
	}
}

func TestTruncateToLines(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5"

	truncated := truncateToLines(content, 3)
	expected := "line1\nline2\nline3\n// ... truncated"
	if truncated != expected {
		t.Errorf("expected %q, got %q", expected, truncated)
	}
}

func TestTruncateToLines_FitsExactly(t *testing.T) {
	content := "line1\nline2\nline3"

	truncated := truncateToLines(content, 5)
	expected := "line1\nline2\nline3\n// ... truncated"
	if truncated != expected {
		t.Errorf("expected %q, got %q", expected, truncated)
	}
}
