package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dotbrains/prr/internal/agent"
	_ "github.com/dotbrains/prr/internal/agent/anthropic" // register provider
	_ "github.com/dotbrains/prr/internal/agent/claudecli" // register provider
	_ "github.com/dotbrains/prr/internal/agent/codexcli"  // register provider
	_ "github.com/dotbrains/prr/internal/agent/openai"    // register provider
	"github.com/dotbrains/prr/internal/config"
	"github.com/dotbrains/prr/internal/spinner"
	"github.com/dotbrains/prr/internal/writer"
)

func newAskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ask <question> [review-dir]",
		Short: "Ask a follow-up question about a review",
		Long:  "Loads review context and asks the AI a follow-up question. Uses the latest review if no directory is given.",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  runAsk,
	}
}

func runAsk(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	question := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	outputDir := cfg.Output.Dir
	if flagOutputDir != "" {
		outputDir = flagOutputDir
	}

	// Resolve review directory.
	var reviewDir string
	if len(args) > 1 {
		reviewDir = args[1]
	} else {
		dir, _, err := writer.FindLatestPRReview(outputDir)
		if err != nil {
			return fmt.Errorf("finding latest review: %w", err)
		}
		if dir == "" {
			// Try any review (including local).
			entries, err := writer.ListReviewDirs(outputDir)
			if err != nil || len(entries) == 0 {
				return fmt.Errorf("no reviews found in %s; run a review first", outputDir)
			}
			reviewDir = entries[0].Path
		} else {
			reviewDir = dir
		}
	}

	// Load review context.
	reviewContext, err := writer.ReadReviewContext(reviewDir)
	if err != nil {
		return fmt.Errorf("loading review context: %w", err)
	}

	// Create agent.
	agentName := flagAgent
	if agentName == "" {
		agentName = cfg.DefaultAgent
	}

	a, err := agent.NewAgentFromConfig(agentName, cfg)
	if err != nil {
		return fmt.Errorf("creating agent: %w", err)
	}

	systemPrompt := agent.BuildAskSystemPrompt()
	userPrompt := agent.BuildAskUserPrompt(reviewContext, question)

	sp := spinner.New(cmd.OutOrStdout(), "→ Thinking...")
	sp.Start()
	answer, err := a.Generate(ctx, systemPrompt, userPrompt)
	sp.Stop()
	if err != nil {
		return fmt.Errorf("asking question: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), answer)
	return nil
}
