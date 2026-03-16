package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dotbrains/prr/internal/agent"
	"github.com/dotbrains/prr/internal/writer"
)

func TestExecute_Version(t *testing.T) {
	// Execute with --version to hit the Execute function without requiring GH.
	os.Args = []string{"prr", "--version"}
	err := Execute("0.0.1-test")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestNewRootCmd(t *testing.T) {
	root := newRootCmd("0.1.0")
	if root.Use != "prr [PR_NUMBER]" {
		t.Errorf("Use = %q", root.Use)
	}

	// Verify subcommands.
	cmds := make(map[string]bool)
	for _, c := range root.Commands() {
		cmds[c.Name()] = true
	}
	for _, want := range []string{"agents", "config", "history", "clean", "post", "describe", "ask", "diff"} {
		if !cmds[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}

func TestNewRootCmd_Version(t *testing.T) {
	root := newRootCmd("1.2.3")
	if root.Version != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %q", root.Version)
	}
}

func TestRunAgents(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Create config with agents.
	configDir := filepath.Join(tmp, ".config", "prr")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(`
default_agent: test-cli
agents:
  test-cli:
    provider: claude-cli
    model: sonnet
  test-api:
    provider: anthropic
    model: claude-3
    api_key_env: TEST_KEY
`), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"agents"})

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !contains(out, "test-cli") {
		t.Error("expected test-cli in output")
	}
	if !contains(out, "test-api") {
		t.Error("expected test-api in output")
	}
	if !contains(out, "(default)") {
		t.Error("expected (default) marker")
	}
	if !contains(out, "✓ (cli)") {
		t.Error("expected CLI provider marker")
	}
}


func TestRunConfigInit(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"config", "init"})

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	// Config file should exist.
	configPath := filepath.Join(tmp, ".config", "prr", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file not created")
	}

	if !contains(buf.String(), "Wrote default config") {
		t.Error("expected success message")
	}
}

func TestRunConfigInit_AlreadyExists(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Pre-create config.
	configDir := filepath.Join(tmp, ".config", "prr")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"config", "init"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when config exists")
	}
}

func TestRunConfigInit_Force(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Pre-create config.
	configDir := filepath.Join(tmp, ".config", "prr")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"config", "init", "--force"})

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	if !contains(buf.String(), "Wrote default config") {
		t.Error("expected success message with --force")
	}
}

func TestRunHistory_Empty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"history", "--output-dir", filepath.Join(tmp, "reviews")})

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	if !contains(buf.String(), "No reviews found") {
		t.Error("expected no reviews message")
	}
}

func TestRunHistory_WithEntries(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	reviewsDir := filepath.Join(tmp, "reviews")
	if err := os.MkdirAll(filepath.Join(reviewsDir, "pr-42-20260101-120000"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(reviewsDir, "pr-99-20260102-150000"), 0o755); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"history", "--output-dir", reviewsDir})

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !contains(out, "pr-42") {
		t.Error("expected pr-42 in output")
	}
	if !contains(out, "pr-99") {
		t.Error("expected pr-99 in output")
	}
}

func TestRunClean_Empty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"clean", "--output-dir", filepath.Join(tmp, "reviews")})

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	if !contains(buf.String(), "No reviews to clean") {
		t.Error("expected no reviews message")
	}
}

func TestRunClean_DryRun(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	reviewsDir := filepath.Join(tmp, "reviews")
	oldDir := filepath.Join(reviewsDir, "pr-1-20240101-120000")
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Set mod time to 60 days ago.
	if err := os.Chtimes(oldDir, time.Now().Add(-60*24*time.Hour), time.Now().Add(-60*24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"clean", "--output-dir", reviewsDir, "--days", "30", "--dry-run"})

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !contains(out, "would remove") {
		t.Error("expected dry-run output")
	}
	// Dir should still exist.
	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		t.Error("dry-run should not delete")
	}
}

func TestRunClean_ActualDelete(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	reviewsDir := filepath.Join(tmp, "reviews")
	oldDir := filepath.Join(reviewsDir, "pr-5-20240101-120000")
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(oldDir, time.Now().Add(-60*24*time.Hour), time.Now().Add(-60*24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"clean", "--output-dir", reviewsDir, "--days", "30"})

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !contains(out, "removed:") {
		t.Error("expected removed message")
	}
	if !contains(out, "Cleaned up") {
		t.Error("expected cleaned up message")
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Error("old dir should have been deleted")
	}
}

func TestRunAgents_WithAPIKeySet(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("MY_API_KEY", "secret")

	configDir := filepath.Join(tmp, ".config", "prr")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(`
default_agent: api-agent
agents:
  api-agent:
    provider: anthropic
    model: claude-3
    api_key_env: MY_API_KEY
`), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"agents"})

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	// API key is set, so should show ✓ for our agent.
	if !contains(out, "✓") {
		t.Error("expected ✓ for agent with set API key")
	}
	if !contains(out, "api-agent") {
		t.Error("expected api-agent in output")
	}
}

func TestRunPost_DryRun(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Create a review dir with metadata.
	reviewsDir := filepath.Join(tmp, "reviews")
	reviewDir := filepath.Join(reviewsDir, "pr-42-20260101-120000")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := writer.ReviewMetadata{
		PRNumber:  42,
		RepoSlug:  "owner/repo",
		HeadSHA:   "abc123",
		AgentName: "claude",
		Summary:   "Found issues",
		Comments: []agent.ReviewComment{
			{File: "main.go", StartLine: 10, EndLine: 10, Severity: "critical", Body: "nil deref"},
			{File: "main.go", StartLine: 20, EndLine: 25, Severity: "suggestion", Body: "refactor"},
		},
	}
	data, _ := json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(reviewDir, "metadata.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"post", reviewDir, "--dry-run"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !contains(out, "PR #42") {
		t.Error("expected PR #42 in output")
	}
	if !contains(out, "dry run") {
		t.Error("expected dry run note")
	}
	if !contains(out, "REQUEST_CHANGES") {
		t.Error("expected REQUEST_CHANGES in payload")
	}
}

func TestRunPost_NotPRReview(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	reviewDir := filepath.Join(tmp, "review")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// metadata with pr_number=0 (local review)
	meta := writer.ReviewMetadata{PRNumber: 0, AgentName: "test", Summary: "ok"}
	data, _ := json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(reviewDir, "metadata.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"post", reviewDir})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for non-PR review")
	}
}

func TestRunPost_NoReviews(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"post", "--output-dir", filepath.Join(tmp, "reviews")})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no reviews exist")
	}
}

func TestRunPost_InvalidDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"post", filepath.Join(tmp, "nonexistent")})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent review dir")
	}
}

func TestRunAsk_NoReviews(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"ask", "what is this?", "--output-dir", filepath.Join(tmp, "reviews")})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no reviews exist")
	}
}

func TestRunAsk_InvalidDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"ask", "what is this?", filepath.Join(tmp, "nonexistent")})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent review dir")
	}
}

func TestRunDescribe_NoGH(t *testing.T) {
	// describe with no gh available should fail at PR resolution
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("PATH", tmp) // no gh on PATH

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"describe"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when gh not available")
	}
}

func TestBuildReviewPayload_RequestChangesOverride(t *testing.T) {
	meta := &writer.ReviewMetadata{
		PRNumber:  1,
		AgentName: "test",
		Summary:   "ok",
		Comments: []agent.ReviewComment{
			{File: "a.go", StartLine: 1, EndLine: 1, Severity: "nit", Body: "nit"},
		},
	}
	p := buildReviewPayload(meta, "request_changes")
	if p.Event != "REQUEST_CHANGES" {
		t.Errorf("expected REQUEST_CHANGES, got %q", p.Event)
	}
}

func TestBuildReviewPayload_ApproveOverride(t *testing.T) {
	meta := &writer.ReviewMetadata{
		PRNumber:  1,
		AgentName: "test",
		Summary:   "ok",
		Comments:  []agent.ReviewComment{},
	}
	p := buildReviewPayload(meta, "approve")
	if p.Event != "APPROVE" {
		t.Errorf("expected APPROVE, got %q", p.Event)
	}
}

func TestBuildReviewPayload_UnknownOverrideFallsBackToComment(t *testing.T) {
	meta := &writer.ReviewMetadata{
		PRNumber:  1,
		AgentName: "test",
		Summary:   "ok",
		Comments:  []agent.ReviewComment{},
	}
	p := buildReviewPayload(meta, "invalid")
	if p.Event != "COMMENT" {
		t.Errorf("expected COMMENT fallback, got %q", p.Event)
	}
}

func TestBuildReviewPayload_SingleLineStartLineOnly(t *testing.T) {
	meta := &writer.ReviewMetadata{
		PRNumber:  1,
		AgentName: "test",
		Summary:   "ok",
		Comments: []agent.ReviewComment{
			{File: "a.go", StartLine: 15, EndLine: 15, Severity: "suggestion", Body: "fix"},
		},
	}
	p := buildReviewPayload(meta, "")
	if len(p.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(p.Comments))
	}
	c := p.Comments[0]
	// When StartLine == EndLine, should use StartLine as Line
	if c.Line != 15 {
		t.Errorf("line = %d, want 15", c.Line)
	}
}

func TestPrintSummary(t *testing.T) {
	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)

	output := &agent.ReviewOutput{
		Summary: "looks good",
		Comments: []agent.ReviewComment{
			{File: "main.go", StartLine: 10, Severity: "critical", Body: "bug"},
			{File: "main.go", StartLine: 20, Severity: "nit", Body: "style"},
			{File: "lib.go", StartLine: 5, Severity: "suggestion", Body: "refactor"},
		},
	}

	printSummary(root, output, "/tmp/reviews/pr-1")

	out := buf.String()
	if !contains(out, "Review complete") {
		t.Error("expected review complete message")
	}
	if !contains(out, "1 critical") {
		t.Error("expected critical stat")
	}
	if !contains(out, "summary.md") {
		t.Error("expected summary.md in file listing")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
