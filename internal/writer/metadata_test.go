package writer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dotbrains/prr/internal/agent"
)

func TestWriteAndReadMetadata(t *testing.T) {
	dir := t.TempDir()

	meta := &ReviewMetadata{
		PRNumber:  42,
		RepoSlug:  "owner/repo",
		HeadSHA:   "abc123",
		AgentName: "claude",
		Model:     "opus",
		CreatedAt: "2025-03-10T10:00:00Z",
		Summary:   "Test summary",
		Comments: []agent.ReviewComment{
			{File: "main.go", StartLine: 10, EndLine: 10, Severity: "critical", Body: "bug here"},
			{File: "main.go", StartLine: 20, EndLine: 25, Severity: "nit", Body: "rename this"},
		},
	}

	if err := WriteMetadata(dir, meta); err != nil {
		t.Fatalf("WriteMetadata failed: %v", err)
	}

	// Verify file exists.
	if _, err := os.Stat(filepath.Join(dir, "metadata.json")); err != nil {
		t.Fatalf("metadata.json not created: %v", err)
	}

	// Read back.
	got, err := ReadMetadata(dir)
	if err != nil {
		t.Fatalf("ReadMetadata failed: %v", err)
	}

	if got.PRNumber != 42 {
		t.Errorf("PRNumber = %d, want 42", got.PRNumber)
	}
	if got.RepoSlug != "owner/repo" {
		t.Errorf("RepoSlug = %q, want owner/repo", got.RepoSlug)
	}
	if got.HeadSHA != "abc123" {
		t.Errorf("HeadSHA = %q, want abc123", got.HeadSHA)
	}
	if len(got.Comments) != 2 {
		t.Errorf("len(Comments) = %d, want 2", len(got.Comments))
	}
	if got.Summary != "Test summary" {
		t.Errorf("Summary = %q, want 'Test summary'", got.Summary)
	}
}

func TestReadMetadata_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadMetadata(dir)
	if err == nil {
		t.Fatal("expected error for missing metadata.json")
	}
}

func TestFindLatestPRReview(t *testing.T) {
	baseDir := t.TempDir()

	// Create a PR review directory.
	reviewDir := filepath.Join(baseDir, "pr-42-20250310-100000")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := &ReviewMetadata{PRNumber: 42, AgentName: "claude", CreatedAt: "2025-03-10"}
	if err := WriteMetadata(reviewDir, meta); err != nil {
		t.Fatal(err)
	}

	// Create a local review directory (should be skipped).
	localDir := filepath.Join(baseDir, "review-main-vs-feature-20250310-110000")
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		t.Fatal(err)
	}

	dir, gotMeta, err := FindLatestPRReview(baseDir)
	if err != nil {
		t.Fatalf("FindLatestPRReview failed: %v", err)
	}
	if dir == "" {
		t.Fatal("expected a directory, got empty string")
	}
	if gotMeta.PRNumber != 42 {
		t.Errorf("PRNumber = %d, want 42", gotMeta.PRNumber)
	}
}

func TestFindLatestPRReview_NoneFound(t *testing.T) {
	baseDir := t.TempDir()
	dir, meta, err := FindLatestPRReview(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir != "" || meta != nil {
		t.Errorf("expected empty result, got dir=%q meta=%v", dir, meta)
	}
}

func TestFindLatestReviewForPR(t *testing.T) {
	baseDir := t.TempDir()

	// Create review for PR 42.
	reviewDir := filepath.Join(baseDir, "pr-42-20250310-100000")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteMetadata(reviewDir, &ReviewMetadata{PRNumber: 42, HeadSHA: "abc"}); err != nil {
		t.Fatal(err)
	}

	// Create review for PR 99.
	review2 := filepath.Join(baseDir, "pr-99-20250310-110000")
	if err := os.MkdirAll(review2, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteMetadata(review2, &ReviewMetadata{PRNumber: 99, HeadSHA: "def"}); err != nil {
		t.Fatal(err)
	}

	// Find PR 42.
	dir, err := FindLatestReviewForPR(baseDir, 42)
	if err != nil {
		t.Fatal(err)
	}
	if dir == "" {
		t.Fatal("expected to find PR 42 review")
	}

	// Find PR 999 (doesn't exist).
	dir, err = FindLatestReviewForPR(baseDir, 999)
	if err != nil {
		t.Fatal(err)
	}
	if dir != "" {
		t.Errorf("expected empty, got %q", dir)
	}
}

func TestReadReviewContext(t *testing.T) {
	dir := t.TempDir()
	meta := &ReviewMetadata{
		PRNumber:  10,
		AgentName: "claude",
		CreatedAt: "2025-03-10",
		Summary:   "Good PR",
		Comments: []agent.ReviewComment{
			{File: "main.go", StartLine: 5, EndLine: 5, Severity: "critical", Body: "nil check"},
		},
	}
	if err := WriteMetadata(dir, meta); err != nil {
		t.Fatal(err)
	}

	ctx, err := ReadReviewContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ctx == "" {
		t.Fatal("expected non-empty context")
	}
	if !contains(ctx, "PR #10") {
		t.Errorf("context missing PR number: %s", ctx)
	}
	if !contains(ctx, "nil check") {
		t.Errorf("context missing comment body: %s", ctx)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
