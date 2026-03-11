package writer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dotbrains/prr/internal/agent"
)

func TestWrite_SingleAgent(t *testing.T) {
	dir := t.TempDir()

	output := &agent.ReviewOutput{
		Summary: "Good PR with minor issues.",
		Comments: []agent.ReviewComment{
			{File: "main.go", StartLine: 10, EndLine: 10, Severity: "critical", Body: "Bug here."},
			{File: "main.go", StartLine: 20, EndLine: 25, Severity: "suggestion", Body: "Refactor this."},
			{File: "cmd/root.go", StartLine: 5, EndLine: 5, Severity: "nit", Body: "Naming."},
		},
	}

	opts := WriteOptions{
		BaseDir:    dir,
		PRNumber:   42,
		AgentName:  "claude",
		Model:      "claude-sonnet-4-20250514",
		MultiAgent: false,
	}

	reviewDir, err := Write(output, opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Check summary.md exists
	summaryPath := filepath.Join(reviewDir, "summary.md")
	if _, err := os.Stat(summaryPath); os.IsNotExist(err) {
		t.Error("summary.md not created")
	}

	summaryContent, _ := os.ReadFile(summaryPath)
	if !strings.Contains(string(summaryContent), "PR #42") {
		t.Error("summary.md missing PR number")
	}
	if !strings.Contains(string(summaryContent), "claude") {
		t.Error("summary.md missing agent name")
	}

	// Check files directory exists
	filesDir := filepath.Join(reviewDir, "files")
	if _, err := os.Stat(filesDir); os.IsNotExist(err) {
		t.Error("files/ directory not created")
	}

	// Check per-file comment files
	mainComments := filepath.Join(filesDir, "main-go.md")
	if _, err := os.Stat(mainComments); os.IsNotExist(err) {
		t.Error("main-go.md not created")
	}

	mainContent, _ := os.ReadFile(mainComments)
	if !strings.Contains(string(mainContent), "Line 10") {
		t.Error("main-go.md missing line 10 comment")
	}
	if !strings.Contains(string(mainContent), "Lines 20-25") {
		t.Error("main-go.md missing lines 20-25 comment")
	}

	rootComments := filepath.Join(filesDir, "cmd-root-go.md")
	if _, err := os.Stat(rootComments); os.IsNotExist(err) {
		t.Error("cmd-root-go.md not created")
	}
}

func TestWrite_MultiAgentFlag(t *testing.T) {
	dir := t.TempDir()

	output := &agent.ReviewOutput{
		Summary:  "Review.",
		Comments: []agent.ReviewComment{{File: "a.go", StartLine: 1, Severity: "nit", Body: "ok"}},
	}

	opts := WriteOptions{
		BaseDir:    dir,
		PRNumber:   7,
		AgentName:  "claude",
		Model:      "sonnet",
		MultiAgent: true,
	}

	reviewDir, err := Write(output, opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	// MultiAgent nests under agent name
	if !strings.Contains(reviewDir, "claude") {
		t.Errorf("expected agent subdirectory, got %q", reviewDir)
	}
	if _, err := os.Stat(filepath.Join(reviewDir, "summary.md")); os.IsNotExist(err) {
		t.Error("summary.md not created")
	}
}

func TestWrite_NoModel(t *testing.T) {
	dir := t.TempDir()

	output := &agent.ReviewOutput{
		Summary:  "Review.",
		Comments: nil,
	}

	opts := WriteOptions{
		BaseDir:   dir,
		PRNumber:  8,
		AgentName: "test",
		Model:     "",
	}

	reviewDir, err := Write(output, opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	summary, _ := os.ReadFile(filepath.Join(reviewDir, "summary.md"))
	if strings.Contains(string(summary), "()") {
		t.Error("empty model should not produce parentheses")
	}
}

func TestWrite_MultiAgent(t *testing.T) {
	dir := t.TempDir()

	outputs := map[string]*AgentOutput{
		"claude": {
			Output: &agent.ReviewOutput{
				Summary:  "Claude review.",
				Comments: []agent.ReviewComment{{File: "main.go", StartLine: 1, Severity: "nit", Body: "Style."}},
			},
			Model: "claude-sonnet-4-20250514",
		},
		"gpt": {
			Output: &agent.ReviewOutput{
				Summary:  "GPT review.",
				Comments: []agent.ReviewComment{{File: "main.go", StartLine: 2, Severity: "suggestion", Body: "Better."}},
			},
			Model: "gpt-4o",
		},
	}

	reviewDir, err := WriteMulti(outputs, WriteMultiOptions{
		BaseDir:  dir,
		PRNumber: 100,
	})
	if err != nil {
		t.Fatalf("WriteMulti failed: %v", err)
	}

	// Check both agent directories
	for _, name := range []string{"claude", "gpt"} {
		agentDir := filepath.Join(reviewDir, name)
		if _, err := os.Stat(filepath.Join(agentDir, "summary.md")); os.IsNotExist(err) {
			t.Errorf("%s/summary.md not created", name)
		}
		if _, err := os.Stat(filepath.Join(agentDir, "files", "main-go.md")); os.IsNotExist(err) {
			t.Errorf("%s/files/main-go.md not created", name)
		}
	}
}

func TestSafeBranchName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"main", "main"},
		{"feature/auth", "feature-auth"},
		{"user/nick/fix", "user-nick-fix"},
		{"branch with spaces", "branch-with-spaces"},
		{"", ""},
	}
	for _, tt := range tests {
		got := safeBranchName(tt.input)
		if got != tt.want {
			t.Errorf("safeBranchName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestWrite_LocalMode(t *testing.T) {
	dir := t.TempDir()

	output := &agent.ReviewOutput{
		Summary:  "Local review.",
		Comments: []agent.ReviewComment{{File: "main.go", StartLine: 1, Severity: "nit", Body: "ok"}},
	}

	opts := WriteOptions{
		BaseDir:    dir,
		PRNumber:   0,
		AgentName:  "claude",
		Model:      "sonnet",
		BaseBranch: "main",
		HeadBranch: "feature/auth",
	}

	reviewDir, err := Write(output, opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Directory should use branch-based naming
	if !strings.Contains(reviewDir, "review-main-vs-feature-auth") {
		t.Errorf("expected branch-based dir name, got %q", reviewDir)
	}

	// Summary should show branch comparison, not PR #0
	summary, _ := os.ReadFile(filepath.Join(reviewDir, "summary.md"))
	if strings.Contains(string(summary), "PR #0") {
		t.Error("summary should not contain PR #0 for local reviews")
	}
	if !strings.Contains(string(summary), "Review: main") {
		t.Error("summary should contain branch comparison header")
	}
}

func TestWriteMulti_LocalMode(t *testing.T) {
	dir := t.TempDir()

	outputs := map[string]*AgentOutput{
		"claude": {
			Output: &agent.ReviewOutput{
				Summary:  "Claude.",
				Comments: []agent.ReviewComment{{File: "a.go", StartLine: 1, Severity: "nit", Body: "ok"}},
			},
			Model: "sonnet",
		},
	}

	reviewDir, err := WriteMulti(outputs, WriteMultiOptions{
		BaseDir:    dir,
		PRNumber:   0,
		BaseBranch: "develop",
		HeadBranch: "feature/new-thing",
	})
	if err != nil {
		t.Fatalf("WriteMulti failed: %v", err)
	}

	if !strings.Contains(reviewDir, "review-develop-vs-feature-new-thing") {
		t.Errorf("expected branch-based dir name, got %q", reviewDir)
	}

	// Check agent subdirectory
	if _, err := os.Stat(filepath.Join(reviewDir, "claude", "summary.md")); os.IsNotExist(err) {
		t.Error("claude/summary.md not created")
	}
}

func TestPathToFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"main.go", "main-go.md"},
		{"src/auth/handler.go", "src-auth-handler-go.md"},
		{"cmd/root.go", "cmd-root-go.md"},
		{"README.md", "README-md.md"},
	}

	for _, tt := range tests {
		got := pathToFilename(tt.input)
		if got != tt.want {
			t.Errorf("pathToFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestListReviewDirs(t *testing.T) {
	dir := t.TempDir()

	// Create some fake review dirs
	if err := os.MkdirAll(filepath.Join(dir, "pr-100-20250101-120000"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "pr-200-20250102-120000"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "not-a-review"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "somefile.txt"), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := ListReviewDirs(dir)
	if err != nil {
		t.Fatalf("ListReviewDirs failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 review dirs, got %d", len(entries))
	}
}

func TestListReviewDirs_Empty(t *testing.T) {
	dir := t.TempDir()

	entries, err := ListReviewDirs(dir)
	if err != nil {
		t.Fatalf("ListReviewDirs failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestListReviewDirs_NonExistent(t *testing.T) {
	entries, err := ListReviewDirs("/nonexistent/path")
	if err != nil {
		t.Fatalf("expected no error for nonexistent dir, got %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries, got %v", entries)
	}
}

func TestCleanOlderThan(t *testing.T) {
	dir := t.TempDir()

	// Create an old review dir
	oldDir := filepath.Join(dir, "pr-100-20240101-120000")
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Set mtime to 90 days ago
	oldTime := time.Now().Add(-90 * 24 * time.Hour)
	if err := os.Chtimes(oldDir, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Create a new review dir
	newDir := filepath.Join(dir, "pr-200-20250301-120000")
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatal(err)
	}

	removed, err := CleanOlderThan(dir, 30*24*time.Hour, false)
	if err != nil {
		t.Fatalf("CleanOlderThan failed: %v", err)
	}

	if len(removed) != 1 {
		t.Errorf("expected 1 removed, got %d", len(removed))
	}

	// Old dir should be gone
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Error("old dir should have been removed")
	}

	// New dir should still exist
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("new dir should still exist")
	}
}

func TestCleanOlderThan_DryRun(t *testing.T) {
	dir := t.TempDir()

	oldDir := filepath.Join(dir, "pr-100-20240101-120000")
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-90 * 24 * time.Hour)
	if err := os.Chtimes(oldDir, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	removed, err := CleanOlderThan(dir, 30*24*time.Hour, true)
	if err != nil {
		t.Fatalf("CleanOlderThan failed: %v", err)
	}

	if len(removed) != 1 {
		t.Errorf("expected 1 in dry run, got %d", len(removed))
	}

	// Dir should still exist (dry run)
	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		t.Error("dir should still exist in dry run")
	}
}
