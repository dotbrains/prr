package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/dotbrains/prr/internal/agent"
)

var allSeverities = []string{"critical", "suggestion", "nit", "praise"}

func TestFilterComments_NoFilters(t *testing.T) {
	comments := []agent.ReviewComment{
		{Severity: "critical", Body: "bug"},
		{Severity: "praise", Body: "nice"},
	}
	got := filterComments(comments, commentFilterOpts{allowedSeverities: allSeverities})
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
}

func TestFilterComments_NoPraise(t *testing.T) {
	comments := []agent.ReviewComment{
		{Severity: "critical", Body: "bug"},
		{Severity: "praise", Body: "nice"},
		{Severity: "suggestion", Body: "refactor"},
	}
	got := filterComments(comments, commentFilterOpts{allowedSeverities: allSeverities, noPraise: true})
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
	for _, c := range got {
		if c.Severity == "praise" {
			t.Error("praise should be filtered")
		}
	}
}

func TestFilterComments_MinSeverity(t *testing.T) {
	comments := []agent.ReviewComment{
		{Severity: "critical", Body: "a"},
		{Severity: "suggestion", Body: "b"},
		{Severity: "nit", Body: "c"},
		{Severity: "praise", Body: "d"},
	}

	// min=suggestion filters out nit and praise
	got := filterComments(comments, commentFilterOpts{allowedSeverities: allSeverities, minSeverity: "suggestion"})
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}

	// min=critical filters out everything except critical
	got = filterComments(comments, commentFilterOpts{allowedSeverities: allSeverities, minSeverity: "critical"})
	if len(got) != 1 {
		t.Errorf("expected 1, got %d", len(got))
	}
}

func TestFilterComments_NoPraiseAndMinSeverity(t *testing.T) {
	comments := []agent.ReviewComment{
		{Severity: "critical", Body: "a"},
		{Severity: "nit", Body: "b"},
		{Severity: "praise", Body: "c"},
	}
	// min=nit + no-praise: keeps critical and nit
	got := filterComments(comments, commentFilterOpts{allowedSeverities: allSeverities, noPraise: true, minSeverity: "nit"})
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
}

func TestFilterComments_UnknownSeverity(t *testing.T) {
	comments := []agent.ReviewComment{
		{Severity: "critical", Body: "a"},
	}
	// Unknown min severity should not filter anything.
	got := filterComments(comments, commentFilterOpts{allowedSeverities: allSeverities, minSeverity: "unknown"})
	if len(got) != 1 {
		t.Errorf("expected 1, got %d", len(got))
	}
}

func TestFilterComments_ConfigSeverities(t *testing.T) {
	comments := []agent.ReviewComment{
		{Severity: "critical", Body: "a"},
		{Severity: "suggestion", Body: "b"},
		{Severity: "nit", Body: "c"},
		{Severity: "praise", Body: "d"},
	}

	// Only allow critical and suggestion via config
	got := filterComments(comments, commentFilterOpts{allowedSeverities: []string{"critical", "suggestion"}})
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
	for _, c := range got {
		if c.Severity == "nit" || c.Severity == "praise" {
			t.Errorf("severity %q should be filtered", c.Severity)
		}
	}

	// Only critical
	got = filterComments(comments, commentFilterOpts{allowedSeverities: []string{"critical"}})
	if len(got) != 1 {
		t.Errorf("expected 1, got %d", len(got))
	}
}

func TestFormatStats(t *testing.T) {
	tests := []struct {
		name  string
		stats map[string]int
		want  string
	}{
		{"empty", map[string]int{}, "no comments"},
		{"single critical", map[string]int{"critical": 1}, "1 critical"},
		{"plural criticals", map[string]int{"critical": 3}, "3 criticals"},
		{"single nit", map[string]int{"nit": 1}, "1 nit"},
		{"plural nits", map[string]int{"nit": 2}, "2 nits"},
		{"mixed", map[string]int{"critical": 1, "suggestion": 2}, "1 critical, 2 suggestions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatStats(tt.stats)
			if got != tt.want {
				t.Errorf("formatStats() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPathToSafeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"main.go", "main-go.md"},
		{"src/internal/handler.go", "src-internal-handler-go.md"},
		{"a/b.c.d", "a-b-c-d.md"},
	}
	for _, tt := range tests {
		got := pathToSafeName(tt.input)
		if got != tt.want {
			t.Errorf("pathToSafeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// initGitRepo creates a temp git repo with an initial commit on "main".
// gitEnv returns environment variables for git commands in tests.
func gitEnv() []string {
	return []string{
		"HOME=/dev/null",
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@test.com",
		"GIT_CONFIG_NOSYSTEM=1",
	}
}

// initGitRepo creates a temp git repo with an initial commit on "main".
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = gitEnv()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %s: %s", args, err, out)
		}
	}
	run("init", "-b", "main")
	run("commit", "--allow-empty", "--no-gpg-sign", "-m", "init")
	return dir
}

func TestIsLocalMode(t *testing.T) {
	// Save and restore
	oldRepo, oldBase := flagRepo, flagBase
	defer func() { flagRepo, flagBase = oldRepo, oldBase }()

	flagRepo = ""
	flagBase = ""
	if isLocalMode() {
		t.Error("expected false when no flags set")
	}

	flagRepo = "/some/path"
	flagBase = ""
	if !isLocalMode() {
		t.Error("expected true when --repo is set")
	}

	flagRepo = ""
	flagBase = "main"
	if !isLocalMode() {
		t.Error("expected true when --base is set")
	}

	flagRepo = "/some/path"
	flagBase = "main"
	if !isLocalMode() {
		t.Error("expected true when both flags set")
	}
}

func TestRunLocalReview_NotARepo(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--repo", filepath.Join(tmp, "nonexistent"), "--base", "main"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for non-repo path")
	}
}

func TestRunLocalReview_SameBranch(t *testing.T) {
	repoDir := initGitRepo(t)
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--repo", repoDir, "--base", "main", "--head", "main"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for same base and head branch")
	}
	if !contains(err.Error(), "same") {
		t.Errorf("expected 'same' in error, got: %v", err)
	}
}

func TestRunLocalReview_EmptyDiff(t *testing.T) {
	repoDir := initGitRepo(t)
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Create a branch with no changes
	cmd := exec.Command("git", "-C", repoDir, "branch", "feature")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git branch failed: %s: %s", err, out)
	}

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--repo", repoDir, "--base", "main", "--head", "feature"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !contains(buf.String(), "No diff") {
		t.Errorf("expected 'No diff' message, got: %s", buf.String())
	}
}

func TestRunLocalReview_WithDiff_AgentError(t *testing.T) {
	repoDir := initGitRepo(t)
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Create a config with a non-existent agent
	configDir := filepath.Join(tmp, ".config", "prr")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(`
default_agent: fake
agents:
  fake:
    provider: nonexistent-provider
    model: fake-model
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a branch with changes
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repoDir}, args...)...)
		cmd.Env = gitEnv()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %s: %s", args, err, out)
		}
	}
	runGit("checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(repoDir, "hello.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("add", ".")
	runGit("commit", "--no-gpg-sign", "-m", "add file")

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--repo", repoDir, "--base", "main", "--head", "feature"})

	err := root.Execute()
	// Should fail at agent creation (unknown provider)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	// But the local review code path was exercised up to the agent call
	out := buf.String()
	if !contains(out, "Local review") {
		t.Errorf("expected 'Local review' in output, got: %s", out)
	}
}

func TestReplaceAll(t *testing.T) {
	tests := []struct {
		s, old, new, want string
	}{
		{"hello world", "world", "go", "hello go"},
		{"aaa", "a", "b", "bbb"},
		{"no match", "x", "y", "no match"},
		{"", "a", "b", ""},
		{"abc", "abc", "xyz", "xyz"},
	}
	for _, tt := range tests {
		got := replaceAll(tt.s, tt.old, tt.new)
		if got != tt.want {
			t.Errorf("replaceAll(%q, %q, %q) = %q, want %q", tt.s, tt.old, tt.new, got, tt.want)
		}
	}
}
