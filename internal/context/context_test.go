package context

import (
	"context"
	"fmt"
	"testing"

	"github.com/dotbrains/prr/internal/agent"
	"github.com/dotbrains/prr/internal/git"
)

// mockExecutor implements exec.CommandExecutor for testing.
type mockExecutor struct {
	outputs map[string]string
	errors  map[string]error
}

func (m *mockExecutor) Run(ctx context.Context, name string, args ...string) (string, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	if err, ok := m.errors[key]; ok {
		return "", err
	}
	if out, ok := m.outputs[key]; ok {
		return out, nil
	}
	return "", fmt.Errorf("unexpected command: %s", key)
}

func (m *mockExecutor) RunWithStdin(ctx context.Context, stdin string, name string, args ...string) (string, error) {
	return m.Run(ctx, name, args...)
}

func TestCollectContext_Basic(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /repo ls-tree --name-only main src/": "src/handler.go\nsrc/auth.go\nsrc/utils.go\n",
			"git -C /repo show main:src/auth.go":         "package src\n\nfunc Auth() {}\n",
			"git -C /repo show main:src/utils.go":        "package src\n\nfunc Utils() {}\n",
		},
	}
	client := git.NewClient(mock)

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), client, "/repo", "main", files, 2000)

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
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /repo ls-tree --name-only main src/": "src/handler.go\nsrc/auth.go\n",
			"git -C /repo show main:src/auth.go":         "package src\n",
		},
	}
	client := git.NewClient(mock)

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), client, "/repo", "main", files, 2000)

	if len(result) != 1 {
		t.Fatalf("expected 1 context file (handler.go excluded), got %d", len(result))
	}
	if result[0].Path != "src/auth.go" {
		t.Errorf("expected src/auth.go, got %s", result[0].Path)
	}
}

func TestCollectContext_SkipsBinaryExtensions(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /repo ls-tree --name-only main src/": "src/handler.go\nsrc/logo.png\nsrc/data.bin\n",
		},
	}
	client := git.NewClient(mock)

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), client, "/repo", "main", files, 2000)

	if len(result) != 0 {
		t.Errorf("expected 0 context files (all skipped), got %d", len(result))
	}
}

func TestCollectContext_SkipsVendoredPaths(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /repo ls-tree --name-only main vendor/": "vendor/lib.go\n",
		},
	}
	client := git.NewClient(mock)

	files := []agent.FileDiff{
		{Path: "vendor/dep.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), client, "/repo", "main", files, 2000)

	if len(result) != 0 {
		t.Errorf("expected 0 context files (vendored skipped), got %d", len(result))
	}
}

func TestCollectContext_SkipsTestFilesUnlessPRHasTests(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /repo ls-tree --name-only main src/": "src/handler.go\nsrc/handler_test.go\nsrc/auth.go\n",
			"git -C /repo show main:src/auth.go":         "package src\n",
		},
	}
	client := git.NewClient(mock)

	// No test files in the PR
	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), client, "/repo", "main", files, 2000)

	// Should skip handler_test.go, include only auth.go
	if len(result) != 1 {
		t.Fatalf("expected 1 context file, got %d", len(result))
	}
	if result[0].Path != "src/auth.go" {
		t.Errorf("expected src/auth.go, got %s", result[0].Path)
	}
}

func TestCollectContext_IncludesTestFilesWhenPRHasTests(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /repo ls-tree --name-only main src/": "src/handler.go\nsrc/handler_test.go\nsrc/auth_test.go\n",
			"git -C /repo show main:src/auth_test.go":    "package src\n\nfunc TestAuth(t *testing.T) {}\n",
		},
	}
	client := git.NewClient(mock)

	// PR modifies a test file
	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
		{Path: "src/handler_test.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), client, "/repo", "main", files, 2000)

	if len(result) != 1 {
		t.Fatalf("expected 1 context file (auth_test.go), got %d", len(result))
	}
	if result[0].Path != "src/auth_test.go" {
		t.Errorf("expected src/auth_test.go, got %s", result[0].Path)
	}
}

func TestCollectContext_RespectsMaxLines(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /repo ls-tree --name-only main src/": "src/handler.go\nsrc/big.go\nsrc/small.go\n",
			"git -C /repo show main:src/big.go":          "line1\nline2\nline3\nline4\nline5\n",
			"git -C /repo show main:src/small.go":        "a\n",
		},
	}
	client := git.NewClient(mock)

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	// Only 3 lines allowed — big.go won't fully fit (5 lines), should truncate or skip
	result := CollectContext(context.Background(), client, "/repo", "main", files, 3)

	// big.go has 5 lines and max is 3, but truncation only happens if remaining > 10
	// So big.go should not be included (remaining = 3, not > 10)
	if len(result) != 0 {
		t.Errorf("expected 0 files (big.go exceeds budget, remaining too small to truncate), got %d", len(result))
	}
}

func TestCollectContext_ZeroMaxLines(t *testing.T) {
	client := git.NewClient(&mockExecutor{})

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), client, "/repo", "main", files, 0)
	if result != nil {
		t.Errorf("expected nil for maxLines=0, got %v", result)
	}
}

func TestCollectContext_NegativeMaxLines(t *testing.T) {
	client := git.NewClient(&mockExecutor{})

	result := CollectContext(context.Background(), client, "/repo", "main", nil, -1)
	if result != nil {
		t.Errorf("expected nil for maxLines=-1, got %v", result)
	}
}

func TestCollectContext_ListFilesError(t *testing.T) {
	mock := &mockExecutor{
		errors: map[string]error{
			"git -C /repo ls-tree --name-only main src/": fmt.Errorf("not found"),
		},
	}
	client := git.NewClient(mock)

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), client, "/repo", "main", files, 2000)
	if len(result) != 0 {
		t.Errorf("expected 0 files on list error, got %d", len(result))
	}
}

func TestCollectContext_ReadFileError(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /repo ls-tree --name-only main src/": "src/handler.go\nsrc/auth.go\n",
		},
		errors: map[string]error{
			"git -C /repo show main:src/auth.go": fmt.Errorf("binary file"),
		},
	}
	client := git.NewClient(mock)

	files := []agent.FileDiff{
		{Path: "src/handler.go", Status: "modified"},
	}

	result := CollectContext(context.Background(), client, "/repo", "main", files, 2000)
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
