package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dotbrains/prr/internal/config"
	"github.com/dotbrains/prr/internal/exec"
	"github.com/dotbrains/prr/internal/gh"
	"github.com/dotbrains/prr/internal/writer"
)

func newPostCmd() *cobra.Command {
	var dryRun bool
	var eventOverride string

	cmd := &cobra.Command{
		Use:   "post [review-dir]",
		Short: "Post review comments to GitHub as a PR review",
		Long:  "Posts the review from a previous prr run directly to GitHub as inline PR review comments. If no review directory is given, uses the latest PR review.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPost(cmd, args, dryRun, eventOverride)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview the review payload without posting")
	cmd.Flags().StringVar(&eventOverride, "event", "", "override review event (comment, request_changes)")

	return cmd
}

func runPost(cmd *cobra.Command, args []string, dryRun bool, eventOverride string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	outputDir := cfg.Output.Dir
	if flagOutputDir != "" {
		outputDir = flagOutputDir
	}

	// Resolve the review directory.
	var reviewDir string
	if len(args) > 0 {
		reviewDir = args[0]
	} else {
		// Find the latest PR review.
		dir, _, err := writer.FindLatestPRReview(outputDir)
		if err != nil {
			return fmt.Errorf("finding latest review: %w", err)
		}
		if dir == "" {
			return fmt.Errorf("no PR reviews found in %s\nRun a review first: prr <PR_NUMBER>", outputDir)
		}
		reviewDir = dir
	}

	// Read metadata.
	meta, err := writer.ReadMetadata(reviewDir)
	if err != nil {
		return fmt.Errorf("reading review metadata from %s: %w (re-run the review to generate metadata)", reviewDir, err)
	}

	if meta.PRNumber <= 0 {
		return fmt.Errorf("review at %s is not a PR review (no PR number); only PR reviews can be posted", reviewDir)
	}

	// Build the review payload.
	payload := buildReviewPayload(meta, eventOverride)

	fmt.Fprintf(cmd.OutOrStdout(), "→ Posting review for PR #%d (%d comments)\n", meta.PRNumber, len(payload.Comments))

	if dryRun {
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling payload: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		fmt.Fprintln(cmd.OutOrStdout(), "\n(dry run — not posted)")
		return nil
	}

	// Post via gh API.
	executor := exec.NewRealExecutor()
	var ghClient *gh.Client
	if meta.RepoSlug != "" {
		ghClient = gh.NewClientWithRepo(executor, meta.RepoSlug)
	} else {
		ghClient = gh.NewClient(executor)
	}

	result, err := ghClient.CreateReview(ctx, meta.PRNumber, payload)
	if err != nil {
		return fmt.Errorf("posting review: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Review posted to PR #%d\n", meta.PRNumber)
	if result.HTMLURL != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "→ %s\n", result.HTMLURL)
	}

	return nil
}

// buildReviewPayload converts review metadata into a GitHub review API payload.
func buildReviewPayload(meta *writer.ReviewMetadata, eventOverride string) *gh.ReviewPayload {
	// Determine event type.
	event := "COMMENT"
	if eventOverride != "" {
		switch eventOverride {
		case "comment":
			event = "COMMENT"
		case "request_changes":
			event = "REQUEST_CHANGES"
		case "approve":
			event = "APPROVE"
		default:
			event = "COMMENT"
		}
	} else {
		// Auto-detect: if any critical comments, request changes.
		for _, c := range meta.Comments {
			if c.Severity == "critical" {
				event = "REQUEST_CHANGES"
				break
			}
		}
	}

	// Build summary body.
	body := fmt.Sprintf("**prr** review by `%s`\n\n%s", meta.AgentName, meta.Summary)

	// Convert comments to GitHub review line comments.
	var comments []gh.ReviewLineComment
	for _, c := range meta.Comments {
		if c.Severity == "praise" {
			continue // skip praise — not useful as inline review comments
		}

		commentBody := fmt.Sprintf("**[%s]** %s", c.Severity, c.Body)

		rc := gh.ReviewLineComment{
			Path: c.File,
			Line: c.EndLine,
			Side: "RIGHT",
			Body: commentBody,
		}

		// Use start_line for multi-line comments.
		if c.StartLine > 0 && c.StartLine < c.EndLine {
			rc.StartLine = c.StartLine
			rc.StartSide = "RIGHT"
		} else if c.StartLine > 0 {
			rc.Line = c.StartLine
		}

		comments = append(comments, rc)
	}

	return &gh.ReviewPayload{
		CommitID: meta.HeadSHA,
		Body:     body,
		Event:    event,
		Comments: comments,
	}
}
