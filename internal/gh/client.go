package gh

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dotbrains/prr/internal/exec"
)

// PRMetadata contains information about a pull request.
type PRMetadata struct {
	Number     int    `json:"number"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	BaseBranch string `json:"baseRefName"`
	HeadBranch string `json:"headRefName"`
}

// ExistingComment represents a conversation-level comment on the PR.
type ExistingComment struct {
	Author    string `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

// ExistingReview represents a submitted review on the PR.
type ExistingReview struct {
	Author      string `json:"author"`
	Body        string `json:"body"`
	State       string `json:"state"` // APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED
	SubmittedAt string `json:"submittedAt"`
}

// ExistingReviewComment represents a line-level review comment on a specific file.
type ExistingReviewComment struct {
	Author    string `json:"author"`
	Body      string `json:"body"`
	Path      string `json:"path"`
	Line      int    `json:"line"`
	DiffHunk  string `json:"diffHunk"`
	CreatedAt string `json:"createdAt"`
}

// Client wraps the gh CLI for PR operations.
type Client struct {
	exec     exec.CommandExecutor
	repoSlug string // optional "owner/repo" for remote operations via -R
}

// NewClient creates a new gh CLI client that operates on the local repo.
func NewClient(executor exec.CommandExecutor) *Client {
	return &Client{exec: executor}
}

// NewClientWithRepo creates a gh CLI client that targets a specific remote repo via -R.
func NewClientWithRepo(executor exec.CommandExecutor, repoSlug string) *Client {
	return &Client{exec: executor, repoSlug: repoSlug}
}

// ghPRArgs builds a gh pr subcommand argument list, injecting -R if a repo slug is set.
func (c *Client) ghPRArgs(subcommand string, extra ...string) []string {
	args := []string{"pr", subcommand}
	if c.repoSlug != "" {
		args = append(args, "-R", c.repoSlug)
	}
	args = append(args, extra...)
	return args
}

// ResolvePRNumber resolves a PR number from an explicit argument or auto-detects from the current branch.
func (c *Client) ResolvePRNumber(ctx context.Context, arg string) (int, error) {
	if arg != "" {
		n, err := strconv.Atoi(arg)
		if err != nil {
			return 0, fmt.Errorf("invalid PR number %q: must be an integer", arg)
		}
		if n <= 0 {
			return 0, fmt.Errorf("invalid PR number %d: must be positive", n)
		}
		return n, nil
	}

	// Auto-detect from current branch
	out, err := c.exec.Run(ctx, "gh", c.ghPRArgs("status", "--json", "number")...)
	if err != nil {
		return 0, fmt.Errorf("auto-detecting PR number: %w\nMake sure you are on a branch with an open PR, or provide a PR number explicitly", err)
	}

	var status struct {
		CurrentBranch struct {
			Number int `json:"number"`
		} `json:"currentBranch"`
	}
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		return 0, fmt.Errorf("parsing gh pr status output: %w", err)
	}

	if status.CurrentBranch.Number == 0 {
		return 0, fmt.Errorf("no open PR found for the current branch\nProvide a PR number explicitly: prr <number>")
	}
	return status.CurrentBranch.Number, nil
}

// GetPRMetadata fetches PR title, body, and branch info.
func (c *Client) GetPRMetadata(ctx context.Context, prNumber int) (*PRMetadata, error) {
	out, err := c.exec.Run(ctx, "gh", c.ghPRArgs("view", strconv.Itoa(prNumber),
		"--json", "number,title,body,baseRefName,headRefName")...)
	if err != nil {
		return nil, fmt.Errorf("fetching PR #%d metadata: %w", prNumber, err)
	}

	var meta PRMetadata
	if err := json.Unmarshal([]byte(out), &meta); err != nil {
		return nil, fmt.Errorf("parsing PR metadata: %w", err)
	}
	return &meta, nil
}

// GetPRDiff fetches the unified diff for a PR.
func (c *Client) GetPRDiff(ctx context.Context, prNumber int) (string, error) {
	out, err := c.exec.Run(ctx, "gh", c.ghPRArgs("diff", strconv.Itoa(prNumber))...)
	if err != nil {
		return "", fmt.Errorf("fetching PR #%d diff: %w", prNumber, err)
	}
	return strings.TrimSpace(out), nil
}

// GetPRComments fetches conversation comments and review summaries for a PR.
func (c *Client) GetPRComments(ctx context.Context, prNumber int) ([]ExistingComment, []ExistingReview, error) {
	out, err := c.exec.Run(ctx, "gh", c.ghPRArgs("view", strconv.Itoa(prNumber),
		"--json", "comments,reviews")...)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching PR #%d comments: %w", prNumber, err)
	}

	var raw struct {
		Comments []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			Body      string `json:"body"`
			CreatedAt string `json:"createdAt"`
		} `json:"comments"`
		Reviews []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			Body        string `json:"body"`
			State       string `json:"state"`
			SubmittedAt string `json:"submittedAt"`
		} `json:"reviews"`
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, nil, fmt.Errorf("parsing PR comments: %w", err)
	}

	var comments []ExistingComment
	for _, c := range raw.Comments {
		comments = append(comments, ExistingComment{
			Author:    c.Author.Login,
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
		})
	}

	var reviews []ExistingReview
	for _, r := range raw.Reviews {
		if r.Body == "" {
			continue // skip reviews with no body (e.g. approvals without comments)
		}
		reviews = append(reviews, ExistingReview{
			Author:      r.Author.Login,
			Body:        r.Body,
			State:       r.State,
			SubmittedAt: r.SubmittedAt,
		})
	}

	return comments, reviews, nil
}

// ListFiles lists files in a directory at a given ref via the GitHub Contents API.
// Returns full paths (e.g. "src/handler.go"), filtering out subdirectories.
func (c *Client) ListFiles(ctx context.Context, ref, dir string) ([]string, error) {
	slug := c.repoSlug
	if slug == "" {
		return nil, fmt.Errorf("repo slug required for remote file listing")
	}

	path := dir
	if path == "." {
		path = ""
	}

	apiPath := fmt.Sprintf("repos/%s/contents/%s?ref=%s", slug, path, ref)
	out, err := c.exec.Run(ctx, "gh", "api", apiPath)
	if err != nil {
		return nil, fmt.Errorf("listing files at %s:%s: %w", ref, dir, err)
	}

	var entries []struct {
		Path string `json:"path"`
		Type string `json:"type"` // "file" or "dir"
	}
	if err := json.Unmarshal([]byte(out), &entries); err != nil {
		return nil, fmt.Errorf("parsing contents response: %w", err)
	}

	var files []string
	for _, e := range entries {
		if e.Type == "file" {
			files = append(files, e.Path)
		}
	}
	return files, nil
}

// ReadFile reads a file's contents at a given ref via the GitHub Contents API.
// Decodes the base64-encoded content returned by the API.
func (c *Client) ReadFile(ctx context.Context, ref, path string) (string, error) {
	slug := c.repoSlug
	if slug == "" {
		return "", fmt.Errorf("repo slug required for remote file reading")
	}

	apiPath := fmt.Sprintf("repos/%s/contents/%s?ref=%s", slug, path, ref)
	out, err := c.exec.Run(ctx, "gh", "api", apiPath)
	if err != nil {
		return "", fmt.Errorf("reading %s at %s: %w", path, ref, err)
	}

	var file struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal([]byte(out), &file); err != nil {
		return "", fmt.Errorf("parsing file response: %w", err)
	}

	if file.Encoding != "base64" {
		return "", fmt.Errorf("unsupported encoding %q for %s", file.Encoding, path)
	}

	// GitHub returns base64 with embedded newlines; strip them before decoding.
	clean := strings.ReplaceAll(file.Content, "\n", "")
	decoded, err := base64.StdEncoding.DecodeString(clean)
	if err != nil {
		return "", fmt.Errorf("decoding base64 content for %s: %w", path, err)
	}

	return string(decoded), nil
}

// GetPRReviewComments fetches line-level review comments for a PR.
func (c *Client) GetPRReviewComments(ctx context.Context, prNumber int) ([]ExistingReviewComment, error) {
	// Use provided slug or auto-detect
	slug := c.repoSlug
	if slug == "" {
		slugOut, err := c.exec.Run(ctx, "gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
		if err != nil {
			return nil, fmt.Errorf("detecting repo: %w", err)
		}
		slug = strings.TrimSpace(slugOut)
		if slug == "" {
			return nil, fmt.Errorf("could not determine repository")
		}
	}

	apiPath := fmt.Sprintf("repos/%s/pulls/%d/comments", slug, prNumber)
	out, err := c.exec.Run(ctx, "gh", "api", apiPath, "--paginate")
	if err != nil {
		return nil, fmt.Errorf("fetching PR #%d review comments: %w", prNumber, err)
	}

	var raw []struct {
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		Body      string `json:"body"`
		Path      string `json:"path"`
		Line      int    `json:"line"`
		DiffHunk  string `json:"diff_hunk"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, fmt.Errorf("parsing review comments: %w", err)
	}

	var comments []ExistingReviewComment
	for _, c := range raw {
		comments = append(comments, ExistingReviewComment{
			Author:    c.User.Login,
			Body:      c.Body,
			Path:      c.Path,
			Line:      c.Line,
			DiffHunk:  c.DiffHunk,
			CreatedAt: c.CreatedAt,
		})
	}

	return comments, nil
}
