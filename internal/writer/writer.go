package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dotbrains/prr/internal/agent"
)

// WriteOptions controls output behavior.
type WriteOptions struct {
	BaseDir    string // base output directory (e.g. "reviews")
	PRNumber   int    // 0 for local (non-PR) reviews
	AgentName  string
	Model      string
	MultiAgent bool   // if true, nest under agent name subdirectory
	BaseBranch string // for local reviews
	HeadBranch string // for local reviews
	RepoSlug   string // "owner/repo" for GitHub operations
	HeadSHA    string // commit SHA at time of review
}

// Write writes a ReviewOutput to markdown files on disk.
// Returns the output directory path.
func Write(output *agent.ReviewOutput, opts WriteOptions) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	var dirName string
	if opts.PRNumber > 0 {
		dirName = fmt.Sprintf("pr-%d-%s", opts.PRNumber, timestamp)
	} else {
		dirName = fmt.Sprintf("review-%s-vs-%s-%s", safeBranchName(opts.BaseBranch), safeBranchName(opts.HeadBranch), timestamp)
	}
	reviewDir := filepath.Join(opts.BaseDir, dirName)

	if opts.MultiAgent {
		reviewDir = filepath.Join(reviewDir, opts.AgentName)
	}

	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	// Write summary.md
	if err := writeSummary(reviewDir, output, opts); err != nil {
		return "", err
	}

	// Write per-file comment files organized by severity
	if err := writeCommentsBySeverity(reviewDir, output.Comments); err != nil {
		return "", err
	}

	// Write structured metadata for prr post / ask / diff.
	meta := &ReviewMetadata{
		PRNumber:  opts.PRNumber,
		RepoSlug:  opts.RepoSlug,
		HeadSHA:   opts.HeadSHA,
		AgentName: opts.AgentName,
		Model:     opts.Model,
		CreatedAt: metadataTimestamp(),
		Comments:  output.Comments,
		Summary:   output.Summary,
	}
	if err := WriteMetadata(reviewDir, meta); err != nil {
		return "", err
	}

	return reviewDir, nil
}

// WriteMultiOptions controls multi-agent output behavior.
type WriteMultiOptions struct {
	BaseDir    string
	PRNumber   int
	BaseBranch string
	HeadBranch string
	RepoSlug   string
	HeadSHA    string
}

// WriteMulti writes multiple agent outputs to the same review directory.
// Returns the top-level review directory path.
func WriteMulti(outputs map[string]*agentOutput, opts WriteMultiOptions) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	var dirName string
	if opts.PRNumber > 0 {
		dirName = fmt.Sprintf("pr-%d-%s", opts.PRNumber, timestamp)
	} else {
		dirName = fmt.Sprintf("review-%s-vs-%s-%s", safeBranchName(opts.BaseBranch), safeBranchName(opts.HeadBranch), timestamp)
	}
	reviewDir := filepath.Join(opts.BaseDir, dirName)

	for agentName, ao := range outputs {
		agentDir := filepath.Join(reviewDir, agentName)
		if err := os.MkdirAll(agentDir, 0o755); err != nil {
			return "", fmt.Errorf("creating output directory for %s: %w", agentName, err)
		}

		writeOpts := WriteOptions{
			PRNumber:   opts.PRNumber,
			AgentName:  agentName,
			Model:      ao.Model,
			BaseBranch: opts.BaseBranch,
			HeadBranch: opts.HeadBranch,
		}

		if err := writeSummary(agentDir, ao.Output, writeOpts); err != nil {
			return "", err
		}

		if err := writeCommentsBySeverity(agentDir, ao.Output.Comments); err != nil {
			return "", err
		}

		// Write per-agent metadata.
		meta := &ReviewMetadata{
			PRNumber:  opts.PRNumber,
			RepoSlug:  opts.RepoSlug,
			HeadSHA:   opts.HeadSHA,
			AgentName: agentName,
			Model:     ao.Model,
			CreatedAt: metadataTimestamp(),
			Comments:  ao.Output.Comments,
			Summary:   ao.Output.Summary,
		}
		if err := WriteMetadata(agentDir, meta); err != nil {
			return "", err
		}
	}

	return reviewDir, nil
}

type agentOutput struct {
	Output *agent.ReviewOutput
	Model  string
}

// AgentOutput is the exported version for use by callers.
type AgentOutput = agentOutput

func writeSummary(dir string, output *agent.ReviewOutput, opts WriteOptions) error {
	stats := output.Stats()

	var sb strings.Builder
	if opts.PRNumber > 0 {
		fmt.Fprintf(&sb, "# PR #%d\n\n", opts.PRNumber)
	} else {
		fmt.Fprintf(&sb, "# Review: %s → %s\n\n", opts.BaseBranch, opts.HeadBranch)
	}
	fmt.Fprintf(&sb, "**Agent:** %s", opts.AgentName)
	if opts.Model != "" {
		fmt.Fprintf(&sb, " (%s)", opts.Model)
	}
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "**Date:** %s\n", time.Now().Format("2006-01-02 15:04:05"))

	sb.WriteString("\n## Overview\n\n")
	sb.WriteString(output.Summary)
	sb.WriteString("\n")

	sb.WriteString("\n## Stats\n\n")
	for _, sev := range []string{"critical", "suggestion", "nit", "praise"} {
		if count, ok := stats[sev]; ok && count > 0 {
			fmt.Fprintf(&sb, "- %d %s\n", count, sev)
		}
	}

	path := filepath.Join(dir, "summary.md")
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("writing summary: %w", err)
	}
	return nil
}

// severityOrder defines the display order for severity directories.
var severityOrder = []string{"critical", "suggestion", "nit", "praise"}

// writeCommentsBySeverity organizes comments into severity-based subdirectories,
// with one markdown file per source file inside each severity directory.
func writeCommentsBySeverity(reviewDir string, comments []agent.ReviewComment) error {
	// Group: severity → file → comments
	grouped := make(map[string]map[string][]agent.ReviewComment)
	for _, c := range comments {
		if grouped[c.Severity] == nil {
			grouped[c.Severity] = make(map[string][]agent.ReviewComment)
		}
		grouped[c.Severity][c.File] = append(grouped[c.Severity][c.File], c)
	}

	for _, sev := range severityOrder {
		byFile, ok := grouped[sev]
		if !ok || len(byFile) == 0 {
			continue
		}

		sevDir := filepath.Join(reviewDir, sev)
		if err := os.MkdirAll(sevDir, 0o755); err != nil {
			return fmt.Errorf("creating %s directory: %w", sev, err)
		}

		for filePath, fileComments := range byFile {
			if err := writeFileComments(sevDir, filePath, fileComments); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeFileComments(sevDir string, filePath string, comments []agent.ReviewComment) error {
	// Sort comments by start line.
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].StartLine < comments[j].StartLine
	})

	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s\n", filePath)

	for i, c := range comments {
		sb.WriteString("\n")
		if c.StartLine == c.EndLine || c.EndLine == 0 {
			fmt.Fprintf(&sb, "## Line %d\n", c.StartLine)
		} else {
			fmt.Fprintf(&sb, "## Lines %d-%d\n", c.StartLine, c.EndLine)
		}
		sb.WriteString("\n")
		sb.WriteString(c.Body)
		sb.WriteString("\n")

		if i < len(comments)-1 {
			sb.WriteString("\n---\n")
		}
	}

	safeName := pathToFilename(filePath)
	outPath := filepath.Join(sevDir, safeName)
	if err := os.WriteFile(outPath, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("writing comments for %s: %w", filePath, err)
	}
	return nil
}

// safeBranchName sanitizes a branch name for use in directory names.
func safeBranchName(name string) string {
	result := strings.ReplaceAll(name, "/", "-")
	result = strings.ReplaceAll(result, "\\", "-")
	result = strings.ReplaceAll(result, " ", "-")
	return result
}

// pathToFilename converts a file path to a safe filename for the output.
func pathToFilename(path string) string {
	// Replace / and . with -
	name := strings.ReplaceAll(path, "/", "-")
	name = strings.ReplaceAll(name, ".", "-")
	return name + ".md"
}

// ListReviewDirs returns existing review directories, sorted newest first.
func ListReviewDirs(baseDir string) ([]ReviewEntry, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading reviews directory: %w", err)
	}

	var reviews []ReviewEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "pr-") && !strings.HasPrefix(name, "review-") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		reviews = append(reviews, ReviewEntry{
			Name:    name,
			Path:    filepath.Join(baseDir, name),
			ModTime: info.ModTime(),
		})
	}

	// Sort newest first
	sort.Slice(reviews, func(i, j int) bool {
		return reviews[i].ModTime.After(reviews[j].ModTime)
	})

	return reviews, nil
}

// ReviewEntry represents a single review output directory.
type ReviewEntry struct {
	Name    string
	Path    string
	ModTime time.Time
}

// CleanOlderThan removes review directories older than the given duration.
func CleanOlderThan(baseDir string, maxAge time.Duration, dryRun bool) ([]string, error) {
	entries, err := ListReviewDirs(baseDir)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-maxAge)
	var removed []string

	for _, e := range entries {
		if e.ModTime.Before(cutoff) {
			if !dryRun {
				if err := os.RemoveAll(e.Path); err != nil {
					return removed, fmt.Errorf("removing %s: %w", e.Name, err)
				}
			}
			removed = append(removed, e.Name)
		}
	}

	return removed, nil
}
