package agent

import (
	"strings"
	"testing"

	"github.com/dotbrains/prr/internal/gh"
)

func TestBuildUserPrompt_NoExistingComments(t *testing.T) {
	input := &ReviewInput{
		PRNumber:   42,
		PRTitle:    "Fix bug",
		BaseBranch: "main",
		HeadBranch: "fix-bug",
		Diff:       "some diff",
	}

	prompt := BuildUserPrompt(input)
	if strings.Contains(prompt, "EXISTING") {
		t.Error("prompt should not contain EXISTING sections when no comments provided")
	}
}

func TestBuildUserPrompt_WithConversationComments(t *testing.T) {
	input := &ReviewInput{
		PRNumber:   42,
		PRTitle:    "Fix bug",
		BaseBranch: "main",
		HeadBranch: "fix-bug",
		Diff:       "some diff",
		ExistingComments: []gh.ExistingComment{
			{Author: "alice", Body: "Looks good overall"},
			{Author: "bob", Body: "Need to fix the race condition"},
		},
	}

	prompt := BuildUserPrompt(input)
	if !strings.Contains(prompt, "EXISTING PR COMMENTS") {
		t.Error("prompt should contain EXISTING PR COMMENTS section")
	}
	if !strings.Contains(prompt, "@alice") {
		t.Error("prompt should contain @alice")
	}
	if !strings.Contains(prompt, "Looks good overall") {
		t.Error("prompt should contain comment body")
	}
	if !strings.Contains(prompt, "@bob") {
		t.Error("prompt should contain @bob")
	}
}

func TestBuildUserPrompt_WithReviews(t *testing.T) {
	input := &ReviewInput{
		PRNumber:   42,
		PRTitle:    "Fix bug",
		BaseBranch: "main",
		HeadBranch: "fix-bug",
		Diff:       "some diff",
		ExistingReviews: []gh.ExistingReview{
			{Author: "alice", Body: "Approve with minor nits", State: "APPROVED"},
		},
	}

	prompt := BuildUserPrompt(input)
	if !strings.Contains(prompt, "EXISTING REVIEWS") {
		t.Error("prompt should contain EXISTING REVIEWS section")
	}
	if !strings.Contains(prompt, "@alice [APPROVED]") {
		t.Error("prompt should contain author and state")
	}
}

func TestBuildUserPrompt_WithReviewComments(t *testing.T) {
	input := &ReviewInput{
		PRNumber:   42,
		PRTitle:    "Fix bug",
		BaseBranch: "main",
		HeadBranch: "fix-bug",
		Diff:       "some diff",
		ExistingReviewComments: []gh.ExistingReviewComment{
			{Author: "alice", Body: "This will deadlock", Path: "src/auth.go", Line: 42},
			{Author: "bob", Body: "Nit: rename this", Path: "src/auth.go", Line: 55},
			{Author: "alice", Body: "Missing error check", Path: "src/handler.go", Line: 10},
		},
	}

	prompt := BuildUserPrompt(input)
	if !strings.Contains(prompt, "EXISTING CODE COMMENTS") {
		t.Error("prompt should contain EXISTING CODE COMMENTS section")
	}
	if !strings.Contains(prompt, "src/auth.go:") {
		t.Error("prompt should contain file path")
	}
	if !strings.Contains(prompt, "Line 42") {
		t.Error("prompt should contain line numbers")
	}
	if !strings.Contains(prompt, "src/handler.go:") {
		t.Error("prompt should contain second file path")
	}
}

func TestBuildUserPrompt_AllExistingCommentTypes(t *testing.T) {
	input := &ReviewInput{
		PRNumber:   42,
		PRTitle:    "Fix bug",
		BaseBranch: "main",
		HeadBranch: "fix-bug",
		Diff:       "some diff",
		ExistingComments: []gh.ExistingComment{
			{Author: "alice", Body: "LGTM"},
		},
		ExistingReviews: []gh.ExistingReview{
			{Author: "bob", Body: "Some concerns", State: "CHANGES_REQUESTED"},
		},
		ExistingReviewComments: []gh.ExistingReviewComment{
			{Author: "bob", Body: "Fix this", Path: "main.go", Line: 1},
		},
	}

	prompt := BuildUserPrompt(input)

	// All three sections should appear
	if !strings.Contains(prompt, "EXISTING PR COMMENTS") {
		t.Error("missing EXISTING PR COMMENTS section")
	}
	if !strings.Contains(prompt, "EXISTING REVIEWS") {
		t.Error("missing EXISTING REVIEWS section")
	}
	if !strings.Contains(prompt, "EXISTING CODE COMMENTS") {
		t.Error("missing EXISTING CODE COMMENTS section")
	}

	// Sections should appear after the diff
	diffIdx := strings.Index(prompt, "some diff")
	commentsIdx := strings.Index(prompt, "EXISTING PR COMMENTS")
	if commentsIdx < diffIdx {
		t.Error("existing comments should appear after the diff")
	}
}

func TestBuildSystemPrompt_ContainsExistingCommentsInstructions(t *testing.T) {
	prompt := BuildSystemPrompt()
	if !strings.Contains(prompt, "EXISTING COMMENTS CONTEXT") {
		t.Error("system prompt should contain EXISTING COMMENTS CONTEXT section")
	}
	if !strings.Contains(prompt, "Do NOT repeat") {
		t.Error("system prompt should instruct not to repeat existing feedback")
	}
}
