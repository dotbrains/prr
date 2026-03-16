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
	"github.com/dotbrains/prr/internal/diff"
	"github.com/dotbrains/prr/internal/exec"
	"github.com/dotbrains/prr/internal/gh"
	"github.com/dotbrains/prr/internal/spinner"
)

func newDescribeCmd() *cobra.Command {
	var flagUpdate bool

	cmd := &cobra.Command{
		Use:   "describe [PR_NUMBER]",
		Short: "Generate or improve a PR description",
		Long:  "Uses AI to generate a PR description from the diff. Optionally updates the PR description on GitHub.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDescribe(cmd, args, flagUpdate)
		},
	}

	cmd.Flags().BoolVar(&flagUpdate, "update", false, "update the PR description on GitHub")

	return cmd
}

func runDescribe(cmd *cobra.Command, args []string, update bool) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	executor := exec.NewRealExecutor()
	ghClient := gh.NewClient(executor)

	prArg := ""
	if len(args) > 0 {
		prArg = args[0]
	}

	prNumber, err := ghClient.ResolvePRNumber(ctx, prArg)
	if err != nil {
		return err
	}

	meta, err := ghClient.GetPRMetadata(ctx, prNumber)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "→ PR #%d: %s\n", meta.Number, meta.Title)

	rawDiff, err := ghClient.GetPRDiff(ctx, prNumber)
	if err != nil {
		return err
	}

	files := diff.Parse(rawDiff)
	files, _ = diff.Filter(files, cfg.Review.IgnorePatterns)

	// Create the agent.
	agentName := flagAgent
	if agentName == "" {
		agentName = cfg.DefaultAgent
	}

	a, err := agent.NewAgentFromConfig(agentName, cfg)
	if err != nil {
		return fmt.Errorf("creating agent: %w", err)
	}

	agentCfg := cfg.Agents[agentName]
	fmt.Fprintf(cmd.OutOrStdout(), "→ agent:  %s (%s)\n", a.Name(), agentCfg.Model)

	systemPrompt := agent.BuildDescribeSystemPrompt()
	userPrompt := agent.BuildDescribeUserPrompt(meta.Title, meta.Body, rawDiff, files)

	sp := spinner.New(cmd.OutOrStdout(), "→ Generating description...")
	sp.Start()
	description, err := a.Generate(ctx, systemPrompt, userPrompt)
	sp.Stop()
	if err != nil {
		return fmt.Errorf("generating description: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), description)

	if update {
		fmt.Fprintln(cmd.OutOrStdout(), "\n→ Updating PR description...")
		if err := ghClient.UpdatePRBody(ctx, prNumber, description); err != nil {
			return fmt.Errorf("updating PR: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "✓ PR #%d description updated.\n", prNumber)
	}

	return nil
}
