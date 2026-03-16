package writer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dotbrains/prr/internal/agent"
)

// ReviewMetadata is the structured metadata stored alongside review output.
type ReviewMetadata struct {
	PRNumber  int                    `json:"pr_number"`
	RepoSlug  string                 `json:"repo_slug,omitempty"`
	HeadSHA   string                 `json:"head_sha,omitempty"`
	AgentName string                 `json:"agent_name"`
	Model     string                 `json:"model,omitempty"`
	CreatedAt string                 `json:"created_at"`
	Comments  []agent.ReviewComment  `json:"comments"`
	Summary   string                 `json:"summary"`
}

// WriteMetadata writes a metadata.json file to the given directory.
func WriteMetadata(dir string, meta *ReviewMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}
	path := filepath.Join(dir, "metadata.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing metadata: %w", err)
	}
	return nil
}

// ReadMetadata reads a metadata.json file from the given directory.
func ReadMetadata(dir string) (*ReviewMetadata, error) {
	path := filepath.Join(dir, "metadata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading metadata: %w", err)
	}
	var meta ReviewMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing metadata: %w", err)
	}
	return &meta, nil
}

// FindLatestReviewForPR finds the most recent review directory for a given PR number.
// Returns the directory path, or "" if none found.
func FindLatestReviewForPR(baseDir string, prNumber int) (string, error) {
	entries, err := ListReviewDirs(baseDir)
	if err != nil {
		return "", err
	}

	prefix := fmt.Sprintf("pr-%d-", prNumber)
	for _, e := range entries {
		// entries are sorted newest-first
		if strings.HasPrefix(e.Name, prefix) {
			meta, err := ReadMetadata(e.Path)
			if err != nil {
				continue // skip dirs without valid metadata
			}
			if meta.PRNumber == prNumber {
				return e.Path, nil
			}
		}
	}
	return "", nil
}

// FindLatestPRReview finds the most recent review directory for any PR (pr_number > 0).
// Returns the directory path and metadata, or "", nil if none found.
func FindLatestPRReview(baseDir string) (string, *ReviewMetadata, error) {
	entries, err := ListReviewDirs(baseDir)
	if err != nil {
		return "", nil, err
	}

	for _, e := range entries {
		if !strings.HasPrefix(e.Name, "pr-") {
			continue
		}
		meta, err := ReadMetadata(e.Path)
		if err != nil {
			continue
		}
		if meta.PRNumber > 0 {
			return e.Path, meta, nil
		}
	}
	return "", nil, nil
}

// ReadReviewContext reads a review directory and returns a human-readable context
// string containing the summary and all comments. Used by `prr ask`.
func ReadReviewContext(dir string) (string, error) {
	meta, err := ReadMetadata(dir)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	if meta.PRNumber > 0 {
		fmt.Fprintf(&sb, "PR #%d review by %s", meta.PRNumber, meta.AgentName)
	} else {
		fmt.Fprintf(&sb, "Review by %s", meta.AgentName)
	}
	if meta.CreatedAt != "" {
		fmt.Fprintf(&sb, " on %s", meta.CreatedAt)
	}
	sb.WriteString("\n\n")

	// Summary
	if meta.Summary != "" {
		sb.WriteString("## Summary\n")
		sb.WriteString(meta.Summary)
		sb.WriteString("\n\n")
	}

	// Comments
	if len(meta.Comments) > 0 {
		sb.WriteString("## Comments\n")
		for _, c := range meta.Comments {
			fmt.Fprintf(&sb, "\n### %s — %s L%d", c.Severity, c.File, c.StartLine)
			if c.EndLine > 0 && c.EndLine != c.StartLine {
				fmt.Fprintf(&sb, "-%d", c.EndLine)
			}
			sb.WriteString("\n")
			sb.WriteString(c.Body)
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

// metadataTimestamp returns a consistent timestamp for metadata.
func metadataTimestamp() string {
	return time.Now().Format(time.RFC3339)
}
