package agent

import (
	"strings"
	"testing"
)

func TestReviewOutput_Stats(t *testing.T) {
	output := &ReviewOutput{
		Comments: []ReviewComment{
			{Severity: "critical"},
			{Severity: "critical"},
			{Severity: "suggestion"},
			{Severity: "nit"},
			{Severity: "nit"},
			{Severity: "nit"},
			{Severity: "praise"},
		},
	}

	stats := output.Stats()
	if stats["critical"] != 2 {
		t.Errorf("expected 2 critical, got %d", stats["critical"])
	}
	if stats["suggestion"] != 1 {
		t.Errorf("expected 1 suggestion, got %d", stats["suggestion"])
	}
	if stats["nit"] != 3 {
		t.Errorf("expected 3 nit, got %d", stats["nit"])
	}
	if stats["praise"] != 1 {
		t.Errorf("expected 1 praise, got %d", stats["praise"])
	}
}

func TestReviewOutput_Stats_Empty(t *testing.T) {
	output := &ReviewOutput{}
	stats := output.Stats()
	if len(stats) != 0 {
		t.Errorf("expected empty stats, got %v", stats)
	}
}

func TestReviewOutput_CommentsByFile(t *testing.T) {
	output := &ReviewOutput{
		Comments: []ReviewComment{
			{File: "main.go", Body: "comment 1"},
			{File: "main.go", Body: "comment 2"},
			{File: "cmd/root.go", Body: "comment 3"},
		},
	}

	byFile := output.CommentsByFile()
	if len(byFile) != 2 {
		t.Errorf("expected 2 files, got %d", len(byFile))
	}
	if len(byFile["main.go"]) != 2 {
		t.Errorf("expected 2 comments for main.go, got %d", len(byFile["main.go"]))
	}
	if len(byFile["cmd/root.go"]) != 1 {
		t.Errorf("expected 1 comment for cmd/root.go, got %d", len(byFile["cmd/root.go"]))
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	prompt := BuildSystemPrompt()

	// Should contain key directives
	if !strings.Contains(prompt, "senior software engineer") {
		t.Error("prompt should mention senior software engineer")
	}
	if !strings.Contains(prompt, "critical") {
		t.Error("prompt should mention critical severity")
	}
	if !strings.Contains(prompt, "JSON") {
		t.Error("prompt should mention JSON response format")
	}
	if !strings.Contains(prompt, "DO NOT") {
		t.Error("prompt should contain DO NOT directives")
	}
}

func TestBuildUserPrompt(t *testing.T) {
	input := &ReviewInput{
		PRNumber:   42,
		PRTitle:    "Fix bug",
		PRBody:     "This fixes the auth bug.",
		BaseBranch: "main",
		HeadBranch: "fix-bug",
		Files: []FileDiff{
			{Path: "main.go", Status: "modified", Diff: "+new line"},
		},
	}

	prompt := BuildUserPrompt(input)

	if !strings.Contains(prompt, "PR #42") {
		t.Error("prompt should contain PR number")
	}
	if !strings.Contains(prompt, "Fix bug") {
		t.Error("prompt should contain PR title")
	}
	if !strings.Contains(prompt, "This fixes the auth bug") {
		t.Error("prompt should contain PR body")
	}
	if !strings.Contains(prompt, "main.go") {
		t.Error("prompt should contain file path")
	}
}

func TestBuildUserPrompt_NoBody(t *testing.T) {
	input := &ReviewInput{
		PRNumber:   1,
		PRTitle:    "Test",
		BaseBranch: "main",
		HeadBranch: "test",
		Diff:       "some diff",
	}

	prompt := BuildUserPrompt(input)
	if strings.Contains(prompt, "PR Description") {
		t.Error("prompt should not contain PR Description when body is empty")
	}
}
