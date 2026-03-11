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
