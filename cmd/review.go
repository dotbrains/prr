package cmd

import (
	"context"
	"fmt"
	"sync"

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
	"github.com/dotbrains/prr/internal/writer"
)

func runReview(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	outputDir := cfg.Output.Dir
	if flagOutputDir != "" {
		outputDir = flagOutputDir
	}

	// Resolve PR number
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

	// Fetch PR metadata
	meta, err := ghClient.GetPRMetadata(ctx, prNumber)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "→ PR #%d: %s\n", meta.Number, meta.Title)

	// Fetch diff
	rawDiff, err := ghClient.GetPRDiff(ctx, prNumber)
	if err != nil {
		return err
	}

	// Parse and filter diff
	files := diff.Parse(rawDiff)
	files, filtered := diff.Filter(files, cfg.Review.IgnorePatterns)

	fmt.Fprintf(cmd.OutOrStdout(), "→ files:  %d", len(files))
	if filtered > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), " (%d filtered)", filtered)
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// Check diff size
	totalLines := diff.LineCount(rawDiff)
	if totalLines > cfg.Review.MaxDiffLines {
		fmt.Fprintf(cmd.OutOrStdout(), "⚠ Diff is large (%d lines, limit %d). Review may be less thorough.\n",
			totalLines, cfg.Review.MaxDiffLines)
	}

	// Fetch existing comments for context (non-fatal)
	existingComments, existingReviews, err := ghClient.GetPRComments(ctx, prNumber)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Could not fetch existing comments: %v\n", err)
	}

	existingReviewComments, err := ghClient.GetPRReviewComments(ctx, prNumber)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Could not fetch existing review comments: %v\n", err)
	}

	contextCount := len(existingComments) + len(existingReviews) + len(existingReviewComments)
	if contextCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "→ context: %d existing comments\n", contextCount)
	}

	// Build review input
	input := &agent.ReviewInput{
		PRNumber:               meta.Number,
		PRTitle:                meta.Title,
		PRBody:                 meta.Body,
		BaseBranch:             meta.BaseBranch,
		HeadBranch:             meta.HeadBranch,
		Diff:                   rawDiff,
		Files:                  files,
		ExistingComments:       existingComments,
		ExistingReviews:        existingReviews,
		ExistingReviewComments: existingReviewComments,
	}

	noPraise, _ := cmd.Flags().GetBool("no-praise")
	minSeverity, _ := cmd.Flags().GetString("min-severity")

	if flagAll {
		return runAllAgents(cmd, ctx, cfg, input, outputDir, noPraise, minSeverity)
	}

	return runSingleAgent(cmd, ctx, cfg, input, outputDir, noPraise, minSeverity)
}

func runSingleAgent(cmd *cobra.Command, ctx context.Context, cfg *config.Config, input *agent.ReviewInput, outputDir string, noPraise bool, minSeverity string) error {
	agentName := flagAgent
	if agentName == "" {
		agentName = cfg.DefaultAgent
	}

	agentCfg, ok := cfg.Agents[agentName]
	if !ok {
		return fmt.Errorf("agent %q not found in config", agentName)
	}

	a, err := agent.NewAgentFromConfig(agentName, cfg)
	if err != nil {
		return fmt.Errorf("creating agent: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "→ agent:  %s (%s)\n", a.Name(), agentCfg.Model)
	fmt.Fprintln(cmd.OutOrStdout(), "→ Reviewing...")

	output, err := a.Review(ctx, input)
	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	// Apply filters
	output.Comments = filterComments(output.Comments, noPraise, minSeverity)

	// Write output
	opts := writer.WriteOptions{
		BaseDir:    outputDir,
		PRNumber:   input.PRNumber,
		AgentName:  agentName,
		Model:      agentCfg.Model,
		MultiAgent: false,
	}

	reviewDir, err := writer.Write(output, opts)
	if err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	printSummary(cmd, output, reviewDir)
	return nil
}

func runAllAgents(cmd *cobra.Command, ctx context.Context, cfg *config.Config, input *agent.ReviewInput, outputDir string, noPraise bool, minSeverity string) error {
	agents, err := agent.AllAgentsFromConfig(cfg)
	if err != nil {
		return err
	}

	if len(agents) == 0 {
		return fmt.Errorf("no agents configured")
	}

	// Print agent names
	names := ""
	for i, a := range agents {
		if i > 0 {
			names += ", "
		}
		names += a.Name()
	}
	fmt.Fprintf(cmd.OutOrStdout(), "→ agents: %s\n", names)
	fmt.Fprintf(cmd.OutOrStdout(), "→ Reviewing with %d agents...\n", len(agents))

	// Run agents in parallel
	type result struct {
		name   string
		output *agent.ReviewOutput
		model  string
		err    error
	}

	results := make([]result, len(agents))
	var wg sync.WaitGroup

	for i, a := range agents {
		wg.Add(1)
		go func(idx int, ag agent.Agent) {
			defer wg.Done()
			out, err := ag.Review(ctx, input)
			agentCfg := cfg.Agents[ag.Name()]
			results[idx] = result{
				name:   ag.Name(),
				output: out,
				model:  agentCfg.Model,
				err:    err,
			}
		}(i, a)
	}

	wg.Wait()

	// Check for errors
	outputs := make(map[string]*writer.AgentOutput)
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Agent %s failed: %v\n", r.name, r.err)
			continue
		}
		r.output.Comments = filterComments(r.output.Comments, noPraise, minSeverity)
		outputs[r.name] = &writer.AgentOutput{
			Output: r.output,
			Model:  r.model,
		}
	}

	if len(outputs) == 0 {
		return fmt.Errorf("all agents failed")
	}

	reviewDir, err := writer.WriteMulti(outputs, outputDir, input.PRNumber)
	if err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n✓ Review complete.\n")
	fmt.Fprintf(cmd.OutOrStdout(), "→ Output: %s/\n\n", reviewDir)

	for name, ao := range outputs {
		stats := ao.Output.Stats()
		fmt.Fprintf(cmd.OutOrStdout(), "  %s/  %s\n", name, formatStats(stats))
	}

	return nil
}

func filterComments(comments []agent.ReviewComment, noPraise bool, minSeverity string) []agent.ReviewComment {
	if !noPraise && minSeverity == "" {
		return comments
	}

	severityOrder := map[string]int{
		"critical":   0,
		"suggestion": 1,
		"nit":        2,
		"praise":     3,
	}

	minLevel := -1
	if minSeverity != "" {
		if lvl, ok := severityOrder[minSeverity]; ok {
			minLevel = lvl
		}
	}

	var filtered []agent.ReviewComment
	for _, c := range comments {
		if noPraise && c.Severity == "praise" {
			continue
		}
		if minLevel >= 0 {
			if lvl, ok := severityOrder[c.Severity]; ok && lvl > minLevel {
				continue
			}
		}
		filtered = append(filtered, c)
	}
	return filtered
}

func printSummary(cmd *cobra.Command, output *agent.ReviewOutput, reviewDir string) {
	stats := output.Stats()

	fmt.Fprintf(cmd.OutOrStdout(), "\n✓ Review complete.\n")
	fmt.Fprintf(cmd.OutOrStdout(), "→ %s\n", formatStats(stats))
	fmt.Fprintf(cmd.OutOrStdout(), "→ Output: %s/\n", reviewDir)

	byFile := output.CommentsByFile()
	if len(byFile) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\nFiles:")
		fmt.Fprintln(cmd.OutOrStdout(), "  summary.md")
		for path, comments := range byFile {
			safeName := pathToSafeName(path)
			fmt.Fprintf(cmd.OutOrStdout(), "  files/%s  (%d comments)\n", safeName, len(comments))
		}
	}
}

func formatStats(stats map[string]int) string {
	parts := []string{}
	for _, sev := range []string{"critical", "suggestion", "nit", "praise"} {
		if count, ok := stats[sev]; ok && count > 0 {
			suffix := sev
			if count != 1 {
				switch sev {
				case "nit":
					suffix = "nits"
				default:
					suffix = sev + "s"
				}
			}
			parts = append(parts, fmt.Sprintf("%d %s", count, suffix))
		}
	}
	if len(parts) == 0 {
		return "no comments"
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}

func pathToSafeName(path string) string {
	name := path
	for _, c := range []string{"/", "."} {
		name = replaceAll(name, c, "-")
	}
	return name + ".md"
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old)
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}
