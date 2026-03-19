package agent

import (
	"context"

	"github.com/dotbrains/prr/internal/gh"
)

// Agent is the interface all AI review providers must implement.
type Agent interface {
	// Name returns the agent's configured name (e.g. "claude").
	Name() string

	// Review sends a PR diff to the AI and returns structured review output.
	Review(ctx context.Context, input *ReviewInput) (*ReviewOutput, error)

	// Generate sends a system+user prompt and returns the raw text response.
	// Used by describe, ask, and other non-review features.
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// CodebaseFile represents an existing file from the repo used as pattern context.
type CodebaseFile struct {
	Path    string
	Content string
}

// ReviewInput contains everything the agent needs to perform a review.
type ReviewInput struct {
	PRNumber   int
	PRTitle    string
	PRBody     string
	BaseBranch string
	HeadBranch string
	Diff       string
	Files      []FileDiff

	// Existing PR comments for context
	ExistingComments       []gh.ExistingComment
	ExistingReviews        []gh.ExistingReview
	ExistingReviewComments []gh.ExistingReviewComment

	// Codebase context: sibling files for pattern analysis
	CodebaseContext []CodebaseFile

	// Project-level review rules from .prr.yaml
	ProjectRules []string

	// Focus modes (security, performance, testing)
	FocusModes []string

	// Metadata for persistence
	RepoSlug string
	HeadSHA  string

	// Full source file contents at HEAD ref (for verification context)
	FileContents map[string]string

	// Incremental review: only review changes since this commit
	SinceCommit string
}

// FileDiff represents a single file's diff within the PR.
type FileDiff struct {
	Path   string
	Diff   string
	Status string // added, modified, deleted, renamed
}

// ReviewOutput is the structured result from an AI review.
type ReviewOutput struct {
	Summary   string
	Comments  []ReviewComment
	Truncated bool `json:"-"` // true if the response was repaired from truncated JSON
}

// VerificationResult holds the outcome of a veracity check on a review comment.
type VerificationResult struct {
	Verdict string `json:"verdict"` // "verified", "inaccurate", or "uncertain"
	Reason  string `json:"reason"`
}

// ReviewComment is a single review comment on a specific location in the code.
type ReviewComment struct {
	File      string `json:"file"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Severity  string `json:"severity"` // critical, suggestion, nit, praise
	Body      string `json:"body"`

	// Verification is populated when --verify is used. Nil otherwise.
	Verification *VerificationResult `json:"verification,omitempty"`
}

// Stats returns counts of each severity level.
func (o *ReviewOutput) Stats() map[string]int {
	counts := make(map[string]int)
	for _, c := range o.Comments {
		counts[c.Severity]++
	}
	return counts
}

// CommentsByFile groups comments by file path.
func (o *ReviewOutput) CommentsByFile() map[string][]ReviewComment {
	grouped := make(map[string][]ReviewComment)
	for _, c := range o.Comments {
		grouped[c.File] = append(grouped[c.File], c)
	}
	return grouped
}
