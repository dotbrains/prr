package git

import (
	"context"
	"fmt"
	"testing"
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

func TestIsRepo_Valid(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /tmp/repo rev-parse --git-dir": ".git\n",
		},
	}
	client := NewClient(mock)

	err := client.IsRepo(context.Background(), "/tmp/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsRepo_NotARepo(t *testing.T) {
	mock := &mockExecutor{
		errors: map[string]error{
			"git -C /tmp/not-repo rev-parse --git-dir": fmt.Errorf("not a git repo"),
		},
	}
	client := NewClient(mock)

	err := client.IsRepo(context.Background(), "/tmp/not-repo")
	if err == nil {
		t.Fatal("expected error for non-repo path")
	}
}

func TestGetCurrentBranch(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /tmp/repo rev-parse --abbrev-ref HEAD": "feature-branch\n",
		},
	}
	client := NewClient(mock)

	branch, err := client.GetCurrentBranch(context.Background(), "/tmp/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "feature-branch" {
		t.Errorf("expected 'feature-branch', got %q", branch)
	}
}

func TestGetCurrentBranch_Detached(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /tmp/repo rev-parse --abbrev-ref HEAD": "HEAD\n",
		},
	}
	client := NewClient(mock)

	branch, err := client.GetCurrentBranch(context.Background(), "/tmp/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "HEAD" {
		t.Errorf("expected 'HEAD', got %q", branch)
	}
}

func TestGetDefaultBranch_FromOriginHead(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /tmp/repo symbolic-ref refs/remotes/origin/HEAD": "refs/remotes/origin/main\n",
		},
	}
	client := NewClient(mock)

	branch, err := client.GetDefaultBranch(context.Background(), "/tmp/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected 'main', got %q", branch)
	}
}

func TestGetDefaultBranch_FallbackToMain(t *testing.T) {
	mock := &mockExecutor{
		errors: map[string]error{
			"git -C /tmp/repo symbolic-ref refs/remotes/origin/HEAD": fmt.Errorf("not set"),
		},
		outputs: map[string]string{
			"git -C /tmp/repo rev-parse --verify main": "abc123\n",
		},
	}
	client := NewClient(mock)

	branch, err := client.GetDefaultBranch(context.Background(), "/tmp/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected 'main', got %q", branch)
	}
}

func TestGetDefaultBranch_FallbackToMaster(t *testing.T) {
	mock := &mockExecutor{
		errors: map[string]error{
			"git -C /tmp/repo symbolic-ref refs/remotes/origin/HEAD": fmt.Errorf("not set"),
			"git -C /tmp/repo rev-parse --verify main":              fmt.Errorf("not found"),
		},
		outputs: map[string]string{
			"git -C /tmp/repo rev-parse --verify master": "abc123\n",
		},
	}
	client := NewClient(mock)

	branch, err := client.GetDefaultBranch(context.Background(), "/tmp/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "master" {
		t.Errorf("expected 'master', got %q", branch)
	}
}

func TestGetDefaultBranch_NoneFound(t *testing.T) {
	mock := &mockExecutor{
		errors: map[string]error{
			"git -C /tmp/repo symbolic-ref refs/remotes/origin/HEAD": fmt.Errorf("not set"),
			"git -C /tmp/repo rev-parse --verify main":              fmt.Errorf("not found"),
			"git -C /tmp/repo rev-parse --verify master":            fmt.Errorf("not found"),
		},
	}
	client := NewClient(mock)

	_, err := client.GetDefaultBranch(context.Background(), "/tmp/repo")
	if err == nil {
		t.Fatal("expected error when no default branch found")
	}
}

func TestGetDiff(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /tmp/repo diff main...feature": "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-old\n+new\n",
		},
	}
	client := NewClient(mock)

	d, err := client.GetDiff(context.Background(), "/tmp/repo", "main", "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d == "" {
		t.Error("expected non-empty diff")
	}
}

func TestGetDiff_Error(t *testing.T) {
	mock := &mockExecutor{
		errors: map[string]error{
			"git -C /tmp/repo diff main...feature": fmt.Errorf("unknown revision"),
		},
	}
	client := NewClient(mock)

	_, err := client.GetDiff(context.Background(), "/tmp/repo", "main", "feature")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetCommitCount(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /tmp/repo rev-list --count main..feature": "5\n",
		},
	}
	client := NewClient(mock)

	count, err := client.GetCommitCount(context.Background(), "/tmp/repo", "main", "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5, got %d", count)
	}
}

func TestGetCommitCount_Error(t *testing.T) {
	mock := &mockExecutor{
		errors: map[string]error{
			"git -C /tmp/repo rev-list --count main..feature": fmt.Errorf("unknown revision"),
		},
	}
	client := NewClient(mock)

	_, err := client.GetCommitCount(context.Background(), "/tmp/repo", "main", "feature")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListFiles(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /tmp/repo ls-tree --name-only main src/": "src/handler.go\nsrc/auth.go\n",
		},
	}
	client := NewClient(mock)

	files, err := client.ListFiles(context.Background(), "/tmp/repo", "main", "src")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0] != "src/handler.go" {
		t.Errorf("expected src/handler.go, got %q", files[0])
	}
}

func TestListFiles_RootDir(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /tmp/repo ls-tree --name-only main ": "README.md\nmain.go\n",
		},
	}
	client := NewClient(mock)

	files, err := client.ListFiles(context.Background(), "/tmp/repo", "main", ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestListFiles_Empty(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /tmp/repo ls-tree --name-only main empty/": "\n",
		},
	}
	client := NewClient(mock)

	files, err := client.ListFiles(context.Background(), "/tmp/repo", "main", "empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if files != nil {
		t.Errorf("expected nil for empty dir, got %v", files)
	}
}

func TestListFiles_Error(t *testing.T) {
	mock := &mockExecutor{
		errors: map[string]error{
			"git -C /tmp/repo ls-tree --name-only main bad/": fmt.Errorf("not found"),
		},
	}
	client := NewClient(mock)

	_, err := client.ListFiles(context.Background(), "/tmp/repo", "main", "bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReadFile(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /tmp/repo show main:src/handler.go": "package src\n\nfunc Handler() {}\n",
		},
	}
	client := NewClient(mock)

	content, err := client.ReadFile(context.Background(), "/tmp/repo", "main", "src/handler.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content == "" {
		t.Error("expected non-empty content")
	}
}

func TestReadFile_Error(t *testing.T) {
	mock := &mockExecutor{
		errors: map[string]error{
			"git -C /tmp/repo show main:missing.go": fmt.Errorf("path not found"),
		},
	}
	client := NewClient(mock)

	_, err := client.ReadFile(context.Background(), "/tmp/repo", "main", "missing.go")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFileReaderAdapter_ListFiles(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /tmp/repo ls-tree --name-only main src/": "src/a.go\nsrc/b.go\n",
		},
	}
	client := NewClient(mock)
	adapter := NewFileReaderAdapter(client, "/tmp/repo")

	files, err := adapter.ListFiles(context.Background(), "main", "src")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestFileReaderAdapter_ReadFile(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"git -C /tmp/repo show main:src/a.go": "package src\n",
		},
	}
	client := NewClient(mock)
	adapter := NewFileReaderAdapter(client, "/tmp/repo")

	content, err := adapter.ReadFile(context.Background(), "main", "src/a.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "package src\n" {
		t.Errorf("unexpected content: %q", content)
	}
}
