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
	BaseDir   string // base output directory (e.g. "reviews")
	PRNumber  int
	AgentName string
	Model     string
	MultiAgent bool // if true, nest under agent name subdirectory
}

// Write writes a ReviewOutput to markdown files on disk.
// Returns the output directory path.
func Write(output *agent.ReviewOutput, opts WriteOptions) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	reviewDir := filepath.Join(opts.BaseDir, fmt.Sprintf("pr-%d-%s", opts.PRNumber, timestamp))

	if opts.MultiAgent {
		reviewDir = filepath.Join(reviewDir, opts.AgentName)
	}

	filesDir := filepath.Join(reviewDir, "files")
	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	// Write summary.md
	if err := writeSummary(reviewDir, output, opts); err != nil {
		return "", err
	}

	// Write per-file comment files
	byFile := output.CommentsByFile()
	for filePath, comments := range byFile {
		if err := writeFileComments(filesDir, filePath, comments); err != nil {
			return "", err
		}
	}

	return reviewDir, nil
}

// WriteMulti writes multiple agent outputs to the same review directory.
// Returns the top-level review directory path.
func WriteMulti(outputs map[string]*agentOutput, baseDir string, prNumber int) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	reviewDir := filepath.Join(baseDir, fmt.Sprintf("pr-%d-%s", prNumber, timestamp))

	for agentName, ao := range outputs {
		agentDir := filepath.Join(reviewDir, agentName)
		filesDir := filepath.Join(agentDir, "files")
		if err := os.MkdirAll(filesDir, 0o755); err != nil {
			return "", fmt.Errorf("creating output directory for %s: %w", agentName, err)
		}

		opts := WriteOptions{
			PRNumber:  prNumber,
			AgentName: agentName,
			Model:     ao.Model,
		}

		if err := writeSummary(agentDir, ao.Output, opts); err != nil {
			return "", err
		}

		byFile := ao.Output.CommentsByFile()
		for filePath, comments := range byFile {
			if err := writeFileComments(filesDir, filePath, comments); err != nil {
				return "", err
			}
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
	sb.WriteString(fmt.Sprintf("# PR #%d\n\n", opts.PRNumber))
	sb.WriteString(fmt.Sprintf("**Agent:** %s", opts.AgentName))
	if opts.Model != "" {
		sb.WriteString(fmt.Sprintf(" (%s)", opts.Model))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("**Date:** %s\n", time.Now().Format("2006-01-02 15:04:05")))

	sb.WriteString("\n## Overview\n\n")
	sb.WriteString(output.Summary)
	sb.WriteString("\n")

	sb.WriteString("\n## Stats\n\n")
	for _, sev := range []string{"critical", "suggestion", "nit", "praise"} {
		if count, ok := stats[sev]; ok && count > 0 {
			sb.WriteString(fmt.Sprintf("- %d %s\n", count, sev))
		}
	}

	path := filepath.Join(dir, "summary.md")
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("writing summary: %w", err)
	}
	return nil
}

func writeFileComments(filesDir string, filePath string, comments []agent.ReviewComment) error {
	// Sort comments by start line
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].StartLine < comments[j].StartLine
	})

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", filePath))

	for i, c := range comments {
		sb.WriteString("\n")
		if c.StartLine == c.EndLine || c.EndLine == 0 {
			sb.WriteString(fmt.Sprintf("## Line %d — %s\n", c.StartLine, c.Severity))
		} else {
			sb.WriteString(fmt.Sprintf("## Lines %d-%d — %s\n", c.StartLine, c.EndLine, c.Severity))
		}
		sb.WriteString("\n")
		sb.WriteString(c.Body)
		sb.WriteString("\n")

		if i < len(comments)-1 {
			sb.WriteString("\n---\n")
		}
	}

	// Convert file path to a safe filename: src/auth/handler.go → src-auth-handler-go.md
	safeName := pathToFilename(filePath)
	outPath := filepath.Join(filesDir, safeName)
	if err := os.WriteFile(outPath, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("writing comments for %s: %w", filePath, err)
	}
	return nil
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
		if !strings.HasPrefix(name, "pr-") {
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
