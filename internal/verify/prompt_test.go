package verify

import (
	"strings"
	"testing"

	"github.com/dotbrains/prr/internal/agent"
)

func TestBuildVerifySystemPrompt(t *testing.T) {
	prompt := BuildVerifySystemPrompt()

	checks := []string{
		"fact-checker",
		"verified",
		"inaccurate",
		"uncertain",
		"verdict",
		"JSON",
	}
	for _, want := range checks {
		if !strings.Contains(prompt, want) {
			t.Errorf("system prompt missing expected keyword %q", want)
		}
	}
}

func TestBuildVerifyUserPrompt_SingleLine(t *testing.T) {
	comment := agent.ReviewComment{
		File:      "src/handler.go",
		StartLine: 42,
		EndLine:   42,
		Severity:  "critical",
		Body:      "Nil pointer dereference here.",
	}

	prompt := BuildVerifyUserPrompt(comment, "--- a/src/handler.go\n+++ b/src/handler.go\n@@ -40,5 +40,5 @@\n")

	if !strings.Contains(prompt, "src/handler.go") {
		t.Error("prompt should contain the file path")
	}
	if !strings.Contains(prompt, "Line: 42") {
		t.Error("prompt should contain single line number")
	}
	if strings.Contains(prompt, "Lines: 42-42") {
		t.Error("prompt should not show range for single-line comment")
	}
	if !strings.Contains(prompt, "critical") {
		t.Error("prompt should contain severity")
	}
	if !strings.Contains(prompt, "Nil pointer dereference") {
		t.Error("prompt should contain comment body")
	}
	if !strings.Contains(prompt, "--- a/src/handler.go") {
		t.Error("prompt should contain the file diff")
	}
}

func TestBuildVerifyUserPrompt_MultiLine(t *testing.T) {
	comment := agent.ReviewComment{
		File:      "src/auth.go",
		StartLine: 10,
		EndLine:   15,
		Severity:  "suggestion",
		Body:      "Extract this block.",
	}

	prompt := BuildVerifyUserPrompt(comment, "some diff content")

	if !strings.Contains(prompt, "Lines: 10-15") {
		t.Error("prompt should show line range for multi-line comment")
	}
}

func TestBuildVerifyUserPrompt_NoDiff(t *testing.T) {
	comment := agent.ReviewComment{
		File:      "missing.go",
		StartLine: 1,
		EndLine:   1,
		Severity:  "nit",
		Body:      "Rename this.",
	}

	prompt := BuildVerifyUserPrompt(comment, "")

	if !strings.Contains(prompt, "(no diff available for this file)") {
		t.Error("prompt should indicate missing diff")
	}
}
