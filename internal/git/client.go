package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/dotbrains/prr/internal/exec"
)

// Client wraps raw git commands for local repo operations.
type Client struct {
	exec exec.CommandExecutor
}

// NewClient creates a new git client.
func NewClient(executor exec.CommandExecutor) *Client {
	return &Client{exec: executor}
}

// IsRepo checks whether the given path is inside a git repository.
func (c *Client) IsRepo(ctx context.Context, repoPath string) error {
	_, err := c.exec.Run(ctx, "git", "-C", repoPath, "rev-parse", "--git-dir")
	if err != nil {
		return fmt.Errorf("%s is not a git repository", repoPath)
	}
	return nil
}

// GetCurrentBranch returns the current branch name (or "HEAD" if detached).
func (c *Client) GetCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	out, err := c.exec.Run(ctx, "git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("detecting current branch: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// GetDefaultBranch attempts to detect the default branch of the repo.
// Tries origin/HEAD first, then falls back to checking main/master.
func (c *Client) GetDefaultBranch(ctx context.Context, repoPath string) (string, error) {
	// Try symbolic-ref for origin/HEAD
	out, err := c.exec.Run(ctx, "git", "-C", repoPath, "symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		ref := strings.TrimSpace(out)
		// refs/remotes/origin/main → main
		if parts := strings.SplitN(ref, "refs/remotes/origin/", 2); len(parts) == 2 {
			return parts[1], nil
		}
	}

	// Fallback: check if main or master exists
	for _, branch := range []string{"main", "master"} {
		_, err := c.exec.Run(ctx, "git", "-C", repoPath, "rev-parse", "--verify", branch)
		if err == nil {
			return branch, nil
		}
	}

	return "", fmt.Errorf("could not detect default branch; specify one with --base")
}

// GetDiff returns the unified diff between two refs.
// Uses three-dot diff (base...head) to show changes on head since it diverged from base.
func (c *Client) GetDiff(ctx context.Context, repoPath, base, head string) (string, error) {
	out, err := c.exec.Run(ctx, "git", "-C", repoPath, "diff", base+"..."+head)
	if err != nil {
		return "", fmt.Errorf("getting diff %s...%s: %w", base, head, err)
	}
	return strings.TrimSpace(out), nil
}

// ListFiles returns file paths in a directory at a given git ref.
// Uses git ls-tree to list blob entries (files only, no directories).
func (c *Client) ListFiles(ctx context.Context, repoPath, ref, dir string) ([]string, error) {
	path := dir
	if path == "." || path == "" {
		path = ""
	} else if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	out, err := c.exec.Run(ctx, "git", "-C", repoPath, "ls-tree", "--name-only", ref, path)
	if err != nil {
		return nil, fmt.Errorf("listing files at %s:%s: %w", ref, dir, err)
	}

	raw := strings.TrimSpace(out)
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// ReadFile reads the contents of a file at a given git ref.
// Uses git show ref:path.
func (c *Client) ReadFile(ctx context.Context, repoPath, ref, path string) (string, error) {
	out, err := c.exec.Run(ctx, "git", "-C", repoPath, "show", ref+":"+path)
	if err != nil {
		return "", fmt.Errorf("reading %s at %s: %w", path, ref, err)
	}
	return out, nil
}

// GetCommitCount returns the number of commits in head that are not in base.
func (c *Client) GetCommitCount(ctx context.Context, repoPath, base, head string) (int, error) {
	out, err := c.exec.Run(ctx, "git", "-C", repoPath, "rev-list", "--count", base+".."+head)
	if err != nil {
		return 0, fmt.Errorf("counting commits: %w", err)
	}
	var count int
	if _, err := fmt.Sscanf(strings.TrimSpace(out), "%d", &count); err != nil {
		return 0, fmt.Errorf("parsing commit count: %w", err)
	}
	return count, nil
}
