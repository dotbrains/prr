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

func TestBuildSystemPrompt_CodebasePatternSection(t *testing.T) {
	prompt := BuildSystemPrompt()
	if !strings.Contains(prompt, "CODEBASE PATTERN CONSISTENCY") {
		t.Error("system prompt should contain CODEBASE PATTERN CONSISTENCY section")
	}
	if !strings.Contains(prompt, "ESTABLISHED PATTERNS") {
		t.Error("system prompt should reference established patterns")
	}
}

func TestBuildUserPrompt_WithCodebaseContext(t *testing.T) {
	input := &ReviewInput{
		PRNumber:   42,
		PRTitle:    "Fix bug",
		BaseBranch: "main",
		HeadBranch: "fix-bug",
		Diff:       "some diff",
		CodebaseContext: []CodebaseFile{
			{Path: "src/auth.go", Content: "package src\nfunc Auth() {}\n"},
			{Path: "src/utils.go", Content: "package src\nfunc Utils() {}\n"},
		},
	}

	prompt := BuildUserPrompt(input)
	if !strings.Contains(prompt, "CODEBASE CONTEXT") {
		t.Error("prompt should contain CODEBASE CONTEXT section")
	}
	if !strings.Contains(prompt, "src/auth.go") {
		t.Error("prompt should contain auth.go path")
	}
	if !strings.Contains(prompt, "src/utils.go") {
		t.Error("prompt should contain utils.go path")
	}
	if !strings.Contains(prompt, "pattern consistency") {
		t.Error("prompt should mention pattern consistency")
	}
}

func TestBuildUserPrompt_WithoutCodebaseContext(t *testing.T) {
	input := &ReviewInput{
		PRNumber:   42,
		PRTitle:    "Fix bug",
		BaseBranch: "main",
		HeadBranch: "fix-bug",
		Diff:       "some diff",
	}

	prompt := BuildUserPrompt(input)
	if strings.Contains(prompt, "CODEBASE CONTEXT") {
		t.Error("prompt should not contain CODEBASE CONTEXT when none provided")
	}
}

func TestBuildUserPrompt_CodebaseContextBeforeExistingComments(t *testing.T) {
	input := &ReviewInput{
		PRNumber:   42,
		PRTitle:    "Fix bug",
		BaseBranch: "main",
		HeadBranch: "fix-bug",
		Diff:       "some diff",
		CodebaseContext: []CodebaseFile{
			{Path: "src/auth.go", Content: "package src\n"},
		},
		ExistingComments: []gh.ExistingComment{
			{Author: "alice", Body: "LGTM"},
		},
	}

	prompt := BuildUserPrompt(input)
	contextIdx := strings.Index(prompt, "CODEBASE CONTEXT")
	commentsIdx := strings.Index(prompt, "EXISTING PR COMMENTS")
	if contextIdx > commentsIdx {
		t.Error("codebase context should appear before existing comments")
	}
}

func TestBuildSystemPrompt_WithFocusModes(t *testing.T) {
	prompt := BuildSystemPrompt("security", "performance")
	if !strings.Contains(prompt, "FOCUS MODE ACTIVE") {
		t.Error("prompt should contain FOCUS MODE ACTIVE section")
	}
	if !strings.Contains(prompt, "SECURITY") {
		t.Error("prompt should contain SECURITY focus")
	}
	if !strings.Contains(prompt, "PERFORMANCE") {
		t.Error("prompt should contain PERFORMANCE focus")
	}
	if !strings.Contains(prompt, "Deprioritize comments outside these focus areas") {
		t.Error("prompt should contain deprioritize instruction")
	}
}

func TestBuildSystemPrompt_UnknownFocusModeIgnored(t *testing.T) {
	prompt := BuildSystemPrompt("nonexistent")
	if strings.Contains(prompt, "FOCUS MODE ACTIVE") {
		t.Error("unknown focus mode should not produce a FOCUS MODE section")
	}
}

func TestBuildSystemPrompt_NoFocusModes(t *testing.T) {
	prompt := BuildSystemPrompt()
	if strings.Contains(prompt, "FOCUS MODE ACTIVE") {
		t.Error("prompt without focus modes should not contain FOCUS MODE section")
	}
}

func TestBuildUserPrompt_WithProjectRules(t *testing.T) {
	input := &ReviewInput{
		PRNumber:   1,
		PRTitle:    "Test",
		BaseBranch: "main",
		HeadBranch: "feat",
		Diff:       "diff",
		ProjectRules: []string{
			"All errors must be wrapped",
			"No SQL outside repository layer",
		},
	}

	prompt := BuildUserPrompt(input)
	if !strings.Contains(prompt, "PROJECT RULES") {
		t.Error("prompt should contain PROJECT RULES section")
	}
	if !strings.Contains(prompt, "All errors must be wrapped") {
		t.Error("prompt should contain first rule")
	}
	if !strings.Contains(prompt, "No SQL outside repository layer") {
		t.Error("prompt should contain second rule")
	}
}

func TestBuildUserPrompt_NoProjectRules(t *testing.T) {
	input := &ReviewInput{
		PRNumber:   1,
		PRTitle:    "Test",
		BaseBranch: "main",
		HeadBranch: "feat",
		Diff:       "diff",
	}

	prompt := BuildUserPrompt(input)
	if strings.Contains(prompt, "PROJECT RULES") {
		t.Error("prompt should not contain PROJECT RULES when none provided")
	}
}

func TestBuildDescribeSystemPrompt(t *testing.T) {
	prompt := BuildDescribeSystemPrompt()
	if !strings.Contains(prompt, "pull request description") {
		t.Error("describe system prompt should mention PR description")
	}
	if !strings.Contains(prompt, "RESPONSE FORMAT") {
		t.Error("describe system prompt should have response format section")
	}
}

func TestBuildDescribeUserPrompt(t *testing.T) {
	prompt := BuildDescribeUserPrompt("Add auth", "old desc", "diff content", nil)
	if !strings.Contains(prompt, "Add auth") {
		t.Error("describe user prompt should contain PR title")
	}
	if !strings.Contains(prompt, "old desc") {
		t.Error("describe user prompt should contain current body")
	}
	if !strings.Contains(prompt, "diff content") {
		t.Error("describe user prompt should contain diff")
	}
}

func TestBuildDescribeUserPrompt_WithFiles(t *testing.T) {
	files := []FileDiff{
		{Path: "main.go", Status: "modified", Diff: "@@ ..."},
	}
	prompt := BuildDescribeUserPrompt("Fix", "", "", files)
	if !strings.Contains(prompt, "main.go") {
		t.Error("describe user prompt should contain file path when files provided")
	}
}

func TestBuildAskSystemPrompt(t *testing.T) {
	prompt := BuildAskSystemPrompt()
	if !strings.Contains(prompt, "answering questions") {
		t.Error("ask system prompt should mention answering questions")
	}
}

func TestBuildAskUserPrompt(t *testing.T) {
	prompt := BuildAskUserPrompt("review context here", "what about the race condition?")
	if !strings.Contains(prompt, "review context here") {
		t.Error("ask user prompt should contain review context")
	}
	if !strings.Contains(prompt, "what about the race condition?") {
		t.Error("ask user prompt should contain question")
	}
}

func TestBuildSystemPrompt_HumanLikeWritingGuidance(t *testing.T) {
	prompt := BuildSystemPrompt()

	// Must contain the core human-writing mandate
	if !strings.Contains(prompt, "indistinguishable from a real human") {
		t.Error("system prompt should contain the human-indistinguishable mandate")
	}

	// Must have banned phrases section
	if !strings.Contains(prompt, "BANNED PHRASES") {
		t.Error("system prompt should contain BANNED PHRASES section")
	}

	// Must ban common AI tells
	bannedPhrases := []string{
		"I notice that",
		"It appears that",
		"Consider...",
		"It would be beneficial",
		"It's worth noting",
	}
	for _, phrase := range bannedPhrases {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("system prompt should ban the phrase %q", phrase)
		}
	}

	// Must have natural variability guidance
	if !strings.Contains(prompt, "NATURAL VARIABILITY") {
		t.Error("system prompt should contain NATURAL VARIABILITY section")
	}

	// Must have good and bad examples
	if !strings.Contains(prompt, "EXAMPLES OF GOOD COMMENTS") {
		t.Error("system prompt should contain EXAMPLES OF GOOD COMMENTS")
	}
	if !strings.Contains(prompt, "EXAMPLES OF BAD COMMENTS") {
		t.Error("system prompt should contain EXAMPLES OF BAD COMMENTS")
	}

	// Must prohibit compliment sandwich pattern
	if !strings.Contains(prompt, "compliment sandwich") {
		t.Error("system prompt should ban the compliment sandwich pattern")
	}
}
