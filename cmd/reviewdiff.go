package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dotbrains/prr/internal/agent"
	"github.com/dotbrains/prr/internal/writer"
)

func newReviewDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <review-dir-1> <review-dir-2>",
		Short: "Compare two review outputs",
		Long:  "Compares comments between two review directories and shows new, resolved, and changed comments.",
		Args:  cobra.ExactArgs(2),
		RunE:  runReviewDiff,
	}
}

// commentKey uniquely identifies a comment by location.
type commentKey struct {
	File      string
	StartLine int
	Severity  string
}

func runReviewDiff(cmd *cobra.Command, args []string) error {
	meta1, err := writer.ReadMetadata(args[0])
	if err != nil {
		return fmt.Errorf("reading first review: %w", err)
	}
	meta2, err := writer.ReadMetadata(args[1])
	if err != nil {
		return fmt.Errorf("reading second review: %w", err)
	}

	// Index comments from both reviews.
	oldComments := indexComments(meta1.Comments)
	newComments := indexComments(meta2.Comments)

	// Compute diffs.
	var newItems, resolvedItems, changedItems []string

	// Collect all files for ordered output.
	fileSet := make(map[string]bool)
	for k := range oldComments {
		fileSet[k.File] = true
	}
	for k := range newComments {
		fileSet[k.File] = true
	}
	var files []string
	for f := range fileSet {
		files = append(files, f)
	}
	sort.Strings(files)

	for _, file := range files {
		// Find new comments (in new but not old).
		for k, c := range newComments {
			if k.File != file {
				continue
			}
			if _, exists := oldComments[k]; !exists {
				newItems = append(newItems, fmt.Sprintf("  + [%s L%d] %s: %s",
					c.Severity, c.StartLine, file, truncateBody(c.Body, 80)))
			} else if oldComments[k].Body != c.Body {
				changedItems = append(changedItems, fmt.Sprintf("  ~ [%s L%d] %s: %q → %q",
					c.Severity, c.StartLine, file, truncateBody(oldComments[k].Body, 40), truncateBody(c.Body, 40)))
			}
		}

		// Find resolved comments (in old but not new).
		for k, c := range oldComments {
			if k.File != file {
				continue
			}
			if _, exists := newComments[k]; !exists {
				resolvedItems = append(resolvedItems, fmt.Sprintf("  - [%s L%d] %s: %s",
					c.Severity, c.StartLine, file, truncateBody(c.Body, 80)))
			}
		}
	}

	// Print summary.
	fmt.Fprintf(cmd.OutOrStdout(), "Review diff: %d → %d comments\n\n", len(meta1.Comments), len(meta2.Comments))

	if len(newItems) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "New comments:")
		for _, item := range newItems {
			fmt.Fprintln(cmd.OutOrStdout(), item)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	if len(resolvedItems) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Resolved comments:")
		for _, item := range resolvedItems {
			fmt.Fprintln(cmd.OutOrStdout(), item)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	if len(changedItems) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Changed comments:")
		for _, item := range changedItems {
			fmt.Fprintln(cmd.OutOrStdout(), item)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	if len(newItems) == 0 && len(resolvedItems) == 0 && len(changedItems) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No differences between reviews.")
	}

	return nil
}

func indexComments(comments []agent.ReviewComment) map[commentKey]agent.ReviewComment {
	idx := make(map[commentKey]agent.ReviewComment, len(comments))
	for _, c := range comments {
		key := commentKey{
			File:      c.File,
			StartLine: c.StartLine,
			Severity:  c.Severity,
		}
		idx[key] = c
	}
	return idx
}

func truncateBody(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
