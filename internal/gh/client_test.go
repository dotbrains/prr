package gh

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

func TestResolvePRNumber_ExplicitArg(t *testing.T) {
	client := NewClient(&mockExecutor{})

	n, err := client.ResolvePRNumber(context.Background(), "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 42 {
		t.Errorf("expected 42, got %d", n)
	}
}

func TestResolvePRNumber_InvalidArg(t *testing.T) {
	client := NewClient(&mockExecutor{})

	_, err := client.ResolvePRNumber(context.Background(), "abc")
	if err == nil {
		t.Fatal("expected error for non-integer arg")
	}
}

func TestResolvePRNumber_NegativeArg(t *testing.T) {
	client := NewClient(&mockExecutor{})

	_, err := client.ResolvePRNumber(context.Background(), "-1")
	if err == nil {
		t.Fatal("expected error for negative arg")
	}
}

func TestResolvePRNumber_AutoDetect(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh pr status --json number": `{"currentBranch":{"number":123}}`,
		},
	}
	client := NewClient(mock)

	n, err := client.ResolvePRNumber(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 123 {
		t.Errorf("expected 123, got %d", n)
	}
}

func TestResolvePRNumber_AutoDetect_NoPR(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh pr status --json number": `{"currentBranch":{"number":0}}`,
		},
	}
	client := NewClient(mock)

	_, err := client.ResolvePRNumber(context.Background(), "")
	if err == nil {
		t.Fatal("expected error when no PR found")
	}
}

func TestResolvePRNumber_AutoDetect_GHFails(t *testing.T) {
	mock := &mockExecutor{
		errors: map[string]error{
			"gh pr status --json number": fmt.Errorf("gh not found"),
		},
	}
	client := NewClient(mock)

	_, err := client.ResolvePRNumber(context.Background(), "")
	if err == nil {
		t.Fatal("expected error when gh fails")
	}
}

func TestGetPRMetadata(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh pr view 42 --json number,title,body,baseRefName,headRefName": `{
				"number": 42,
				"title": "Fix bug",
				"body": "This fixes the bug.",
				"baseRefName": "main",
				"headRefName": "fix-bug"
			}`,
		},
	}
	client := NewClient(mock)

	meta, err := client.GetPRMetadata(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Number != 42 {
		t.Errorf("expected number 42, got %d", meta.Number)
	}
	if meta.Title != "Fix bug" {
		t.Errorf("expected title 'Fix bug', got %q", meta.Title)
	}
	if meta.BaseBranch != "main" {
		t.Errorf("expected base main, got %q", meta.BaseBranch)
	}
	if meta.HeadBranch != "fix-bug" {
		t.Errorf("expected head fix-bug, got %q", meta.HeadBranch)
	}
}

func TestGetPRDiff(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh pr diff 42": "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-old\n+new",
		},
	}
	client := NewClient(mock)

	diff, err := client.GetPRDiff(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff == "" {
		t.Error("expected non-empty diff")
	}
}

func TestGetPRComments(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh pr view 42 --json comments,reviews": `{
				"comments": [
					{"author": {"login": "alice"}, "body": "Looks good overall", "createdAt": "2025-03-10T10:00:00Z"},
					{"author": {"login": "bob"}, "body": "Need to fix the race condition", "createdAt": "2025-03-10T11:00:00Z"}
				],
				"reviews": [
					{"author": {"login": "alice"}, "body": "Approve with minor nits", "state": "APPROVED", "submittedAt": "2025-03-10T12:00:00Z"},
					{"author": {"login": "charlie"}, "body": "", "state": "APPROVED", "submittedAt": "2025-03-10T13:00:00Z"}
				]
			}`,
		},
	}
	client := NewClient(mock)

	comments, reviews, err := client.GetPRComments(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
	if comments[0].Author != "alice" || comments[0].Body != "Looks good overall" {
		t.Errorf("unexpected first comment: %+v", comments[0])
	}
	if comments[1].Author != "bob" {
		t.Errorf("unexpected second comment author: %s", comments[1].Author)
	}

	// Only 1 review — the empty-body one should be filtered
	if len(reviews) != 1 {
		t.Fatalf("expected 1 review (empty body filtered), got %d", len(reviews))
	}
	if reviews[0].Author != "alice" || reviews[0].State != "APPROVED" {
		t.Errorf("unexpected review: %+v", reviews[0])
	}
}

func TestGetPRComments_Empty(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh pr view 10 --json comments,reviews": `{"comments": [], "reviews": []}`,
		},
	}
	client := NewClient(mock)

	comments, reviews, err := client.GetPRComments(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(comments))
	}
	if len(reviews) != 0 {
		t.Errorf("expected 0 reviews, got %d", len(reviews))
	}
}

func TestGetPRComments_Error(t *testing.T) {
	mock := &mockExecutor{
		errors: map[string]error{
			"gh pr view 42 --json comments,reviews": fmt.Errorf("gh auth error"),
		},
	}
	client := NewClient(mock)

	_, _, err := client.GetPRComments(context.Background(), 42)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetPRReviewComments(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh repo view --json nameWithOwner --jq .nameWithOwner": "dotbrains/prr\n",
			"gh api repos/dotbrains/prr/pulls/42/comments --paginate": `[
				{"user": {"login": "alice"}, "body": "This will deadlock", "path": "src/auth.go", "line": 42, "diff_hunk": "@@ -40,5 +40,5 @@", "created_at": "2025-03-10T10:00:00Z"},
				{"user": {"login": "bob"}, "body": "Nit: rename this", "path": "src/auth.go", "line": 55, "diff_hunk": "@@ -50,5 +50,5 @@", "created_at": "2025-03-10T11:00:00Z"},
				{"user": {"login": "alice"}, "body": "Missing error check", "path": "src/handler.go", "line": 10, "diff_hunk": "@@ -8,5 +8,5 @@", "created_at": "2025-03-10T12:00:00Z"}
			]`,
		},
	}
	client := NewClient(mock)

	comments, err := client.GetPRReviewComments(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(comments) != 3 {
		t.Fatalf("expected 3 review comments, got %d", len(comments))
	}
	if comments[0].Author != "alice" || comments[0].Path != "src/auth.go" || comments[0].Line != 42 {
		t.Errorf("unexpected first comment: %+v", comments[0])
	}
	if comments[2].Path != "src/handler.go" {
		t.Errorf("unexpected third comment path: %s", comments[2].Path)
	}
}

func TestGetPRReviewComments_EmptyRepo(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh repo view --json nameWithOwner --jq .nameWithOwner": "\n",
		},
	}
	client := NewClient(mock)

	_, err := client.GetPRReviewComments(context.Background(), 42)
	if err == nil {
		t.Fatal("expected error for empty repo slug")
	}
}

func TestGetPRReviewComments_RepoError(t *testing.T) {
	mock := &mockExecutor{
		errors: map[string]error{
			"gh repo view --json nameWithOwner --jq .nameWithOwner": fmt.Errorf("not a git repo"),
		},
	}
	client := NewClient(mock)

	_, err := client.GetPRReviewComments(context.Background(), 42)
	if err == nil {
		t.Fatal("expected error when repo detection fails")
	}
}

// Tests for -R flag injection with NewClientWithRepo

func TestGetPRMetadata_WithRepoSlug(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh pr view -R dotbrains/prr 42 --json number,title,body,baseRefName,headRefName": `{
				"number": 42,
				"title": "Remote PR",
				"body": "From a URL",
				"baseRefName": "main",
				"headRefName": "feature"
			}`,
		},
	}
	client := NewClientWithRepo(mock, "dotbrains/prr")

	meta, err := client.GetPRMetadata(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Title != "Remote PR" {
		t.Errorf("expected title 'Remote PR', got %q", meta.Title)
	}
}

func TestGetPRDiff_WithRepoSlug(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh pr diff -R owner/repo 10": "diff --git a/f.go b/f.go\n-old\n+new",
		},
	}
	client := NewClientWithRepo(mock, "owner/repo")

	diff, err := client.GetPRDiff(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff == "" {
		t.Error("expected non-empty diff")
	}
}

func TestGetPRComments_WithRepoSlug(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh pr view -R owner/repo 10 --json comments,reviews": `{"comments": [], "reviews": []}`,
		},
	}
	client := NewClientWithRepo(mock, "owner/repo")

	comments, reviews, err := client.GetPRComments(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 0 || len(reviews) != 0 {
		t.Errorf("expected empty results")
	}
}

func TestGetPRReviewComments_WithRepoSlug(t *testing.T) {
	// When repoSlug is set, should skip gh repo view auto-detect and use slug directly
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh api repos/owner/repo/pulls/10/comments --paginate": `[]`,
		},
	}
	client := NewClientWithRepo(mock, "owner/repo")

	comments, err := client.GetPRReviewComments(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(comments))
	}
}

func TestGhPRArgs_NoSlug(t *testing.T) {
	client := NewClient(&mockExecutor{})
	args := client.ghPRArgs("view", "42", "--json", "number")
	want := []string{"pr", "view", "42", "--json", "number"}
	if len(args) != len(want) {
		t.Fatalf("got %v, want %v", args, want)
	}
	for i, a := range args {
		if a != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, a, want[i])
		}
	}
}

func TestGhPRArgs_WithSlug(t *testing.T) {
	client := NewClientWithRepo(&mockExecutor{}, "owner/repo")
	args := client.ghPRArgs("diff", "42")
	want := []string{"pr", "diff", "-R", "owner/repo", "42"}
	if len(args) != len(want) {
		t.Fatalf("got %v, want %v", args, want)
	}
	for i, a := range args {
		if a != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, a, want[i])
		}
	}
}

func TestListFiles(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh api repos/owner/repo/contents/src?ref=main": `[
				{"path": "src/handler.go", "type": "file"},
				{"path": "src/auth.go", "type": "file"},
				{"path": "src/internal", "type": "dir"}
			]`,
		},
	}
	client := NewClientWithRepo(mock, "owner/repo")

	files, err := client.ListFiles(context.Background(), "main", "src")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files (dirs excluded), got %d", len(files))
	}
	if files[0] != "src/handler.go" {
		t.Errorf("expected src/handler.go, got %q", files[0])
	}
	if files[1] != "src/auth.go" {
		t.Errorf("expected src/auth.go, got %q", files[1])
	}
}

func TestListFiles_RootDir(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh api repos/owner/repo/contents/?ref=main": `[
				{"path": "README.md", "type": "file"},
				{"path": "main.go", "type": "file"}
			]`,
		},
	}
	client := NewClientWithRepo(mock, "owner/repo")

	files, err := client.ListFiles(context.Background(), "main", ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestListFiles_NoSlug(t *testing.T) {
	client := NewClient(&mockExecutor{})

	_, err := client.ListFiles(context.Background(), "main", "src")
	if err == nil {
		t.Fatal("expected error when no repo slug")
	}
}

func TestListFiles_Error(t *testing.T) {
	mock := &mockExecutor{
		errors: map[string]error{
			"gh api repos/owner/repo/contents/bad?ref=main": fmt.Errorf("404"),
		},
	}
	client := NewClientWithRepo(mock, "owner/repo")

	_, err := client.ListFiles(context.Background(), "main", "bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReadFile(t *testing.T) {
	// "package src\nfunc Auth() {}\n" base64-encoded
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh api repos/owner/repo/contents/src/auth.go?ref=main": `{
				"content": "cGFja2FnZSBzcmMKZnVuYyBBdXRoKCkge30K",
				"encoding": "base64"
			}`,
		},
	}
	client := NewClientWithRepo(mock, "owner/repo")

	content, err := client.ReadFile(context.Background(), "main", "src/auth.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "package src\nfunc Auth() {}\n" {
		t.Errorf("unexpected content: %q", content)
	}
}

func TestReadFile_NoSlug(t *testing.T) {
	client := NewClient(&mockExecutor{})

	_, err := client.ReadFile(context.Background(), "main", "src/auth.go")
	if err == nil {
		t.Fatal("expected error when no repo slug")
	}
}

func TestReadFile_UnsupportedEncoding(t *testing.T) {
	mock := &mockExecutor{
		outputs: map[string]string{
			"gh api repos/owner/repo/contents/src/auth.go?ref=main": `{
				"content": "raw content",
				"encoding": "utf-8"
			}`,
		},
	}
	client := NewClientWithRepo(mock, "owner/repo")

	_, err := client.ReadFile(context.Background(), "main", "src/auth.go")
	if err == nil {
		t.Fatal("expected error for unsupported encoding")
	}
}

func TestReadFile_APIError(t *testing.T) {
	mock := &mockExecutor{
		errors: map[string]error{
			"gh api repos/owner/repo/contents/missing.go?ref=main": fmt.Errorf("404"),
		},
	}
	client := NewClientWithRepo(mock, "owner/repo")

	_, err := client.ReadFile(context.Background(), "main", "missing.go")
	if err == nil {
		t.Fatal("expected error")
	}
}
