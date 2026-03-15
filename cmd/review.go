package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"

	"strconv"

	"github.com/dotbrains/prr/internal/agent"
	_ "github.com/dotbrains/prr/internal/agent/anthropic" // register provider
	_ "github.com/dotbrains/prr/internal/agent/claudecli" // register provider
	_ "github.com/dotbrains/prr/internal/agent/codexcli"  // register provider
	_ "github.com/dotbrains/prr/internal/agent/openai"    // register provider
	"github.com/dotbrains/prr/internal/config"
	contextpkg "github.com/dotbrains/prr/internal/context"
	"github.com/dotbrains/prr/internal/diff"
	"github.com/dotbrains/prr/internal/exec"
	"github.com/dotbrains/prr/internal/gh"
	gitpkg "github.com/dotbrains/prr/internal/git"
	"github.com/dotbrains/prr/internal/spinner"
	"github.com/dotbrains/prr/internal/writer"
)

// isLocalMode returns true if --repo or --base was provided.
func isLocalMode() bool {
	return flagRepo != "" || flagBase != ""
}

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

	noPraise, _ := cmd.Flags().GetBool("no-praise")
	minSeverity, _ := cmd.Flags().GetString("min-severity")

	filterOpts := commentFilterOpts{
		allowedSeverities: cfg.Output.Severities,
		noPraise:          noPraise,
		minSeverity:       minSeverity,
	}

	if isLocalMode() {
		return runLocalReview(cmd, ctx, cfg, outputDir, filterOpts)
	}

	// Check if the argument is a PR URL
	if len(args) > 0 && gh.IsPRURL(args[0]) {
		return runURLReview(cmd, ctx, cfg, args[0], outputDir, filterOpts)
	}

	return runPRReview(cmd, ctx, cfg, args, "", outputDir, filterOpts)
}

// runLocalReview handles the --repo/--base local git review path.
func runLocalReview(cmd *cobra.Command, ctx context.Context, cfg *config.Config, outputDir string, filter commentFilterOpts) error {
	executor := exec.NewRealExecutor()
	gitClient := gitpkg.NewClient(executor)

	// Resolve repo path
	repoPath := flagRepo
	if repoPath == "" {
		repoPath, _ = os.Getwd()
	}

	if err := gitClient.IsRepo(ctx, repoPath); err != nil {
		return err
	}

	// Resolve base branch
	baseBranch := flagBase
	if baseBranch == "" {
		var err error
		baseBranch, err = gitClient.GetDefaultBranch(ctx, repoPath)
		if err != nil {
			return err
		}
	}

	// Resolve head branch
	headBranch := flagHead
	if headBranch == "" {
		var err error
		headBranch, err = gitClient.GetCurrentBranch(ctx, repoPath)
		if err != nil {
			return err
		}
	}

	if baseBranch == headBranch {
		return fmt.Errorf("base and head branches are the same (%s); nothing to review", baseBranch)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "→ Local review: %s → %s\n", baseBranch, headBranch)
	fmt.Fprintf(cmd.OutOrStdout(), "→ repo:  %s\n", repoPath)

	// Fetch diff
	rawDiff, err := gitClient.GetDiff(ctx, repoPath, baseBranch, headBranch)
	if err != nil {
		return err
	}

	if rawDiff == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "No diff between branches. Nothing to review.")
		return nil
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

	// Collect codebase context for pattern analysis.
	var codebaseCtx []agent.CodebaseFile
	if cfg.Review.CodebaseContext && !flagNoContext {
		codebaseCtx = contextpkg.CollectContext(ctx, gitClient, repoPath, baseBranch, files, cfg.Review.MaxContextLines)
		if len(codebaseCtx) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "→ context: %d sibling files\n", len(codebaseCtx))
		}
	}

	// Build review input (no PR number, no existing comments)
	input := &agent.ReviewInput{
		PRNumber:        0,
		PRTitle:         fmt.Sprintf("%s → %s", baseBranch, headBranch),
		BaseBranch:      baseBranch,
		HeadBranch:      headBranch,
		Diff:            rawDiff,
		Files:           files,
		CodebaseContext: codebaseCtx,
	}

	if flagAll {
		return runAllAgents(cmd, ctx, cfg, input, outputDir, filter)
	}
	return runSingleAgent(cmd, ctx, cfg, input, outputDir, filter)
}

// runURLReview handles review via a GitHub PR URL.
func runURLReview(cmd *cobra.Command, ctx context.Context, cfg *config.Config, prURL string, outputDir string, filter commentFilterOpts) error {
	owner, repo, prNumber, err := gh.ParsePRURL(prURL)
	if err != nil {
		return err
	}
	repoSlug := owner + "/" + repo

	fmt.Fprintf(cmd.OutOrStdout(), "→ Remote: %s\n", repoSlug)

	return runPRReview(cmd, ctx, cfg, nil, repoSlug, outputDir, filter, strconv.Itoa(prNumber))
}

// runPRReview handles the original GitHub PR review path.
// prNumberOverride is used when the PR number is already known (e.g. from URL parsing).
func runPRReview(cmd *cobra.Command, ctx context.Context, cfg *config.Config, args []string, repoSlug string, outputDir string, filter commentFilterOpts, prNumberOverride ...string) error {
	executor := exec.NewRealExecutor()
	var ghClient *gh.Client
	if repoSlug != "" {
		ghClient = gh.NewClientWithRepo(executor, repoSlug)
	} else {
		ghClient = gh.NewClient(executor)
	}

	prArg := ""
	if len(prNumberOverride) > 0 && prNumberOverride[0] != "" {
		prArg = prNumberOverride[0]
	} else if len(args) > 0 {
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

	// Collect codebase context if running from a local repo.
	var codebaseCtx []agent.CodebaseFile
	if cfg.Review.CodebaseContext && !flagNoContext {
		localExecutor := exec.NewRealExecutor()
		localGit := gitpkg.NewClient(localExecutor)
		cwd, _ := os.Getwd()
		if err := localGit.IsRepo(ctx, cwd); err == nil {
			codebaseCtx = contextpkg.CollectContext(ctx, localGit, cwd, meta.BaseBranch, files, cfg.Review.MaxContextLines)
			if len(codebaseCtx) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "→ context: %d sibling files\n", len(codebaseCtx))
			}
		}
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
		CodebaseContext:        codebaseCtx,
		ExistingComments:       existingComments,
		ExistingReviews:        existingReviews,
		ExistingReviewComments: existingReviewComments,
	}

	if flagAll {
		return runAllAgents(cmd, ctx, cfg, input, outputDir, filter)
	}
	return runSingleAgent(cmd, ctx, cfg, input, outputDir, filter)
}

func runSingleAgent(cmd *cobra.Command, ctx context.Context, cfg *config.Config, input *agent.ReviewInput, outputDir string, filter commentFilterOpts) error {
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

	sp := spinner.New(cmd.OutOrStdout(), "→ Reviewing...")
	sp.Start()
	output, err := a.Review(ctx, input)
	sp.Stop()
	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	// Apply filters
	output.Comments = filterComments(output.Comments, filter)

	// Write output
	opts := writer.WriteOptions{
		BaseDir:    outputDir,
		PRNumber:   input.PRNumber,
		AgentName:  agentName,
		Model:      agentCfg.Model,
		MultiAgent: false,
		BaseBranch: input.BaseBranch,
		HeadBranch: input.HeadBranch,
	}

	reviewDir, err := writer.Write(output, opts)
	if err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	printSummary(cmd, output, reviewDir)
	return nil
}

func runAllAgents(cmd *cobra.Command, ctx context.Context, cfg *config.Config, input *agent.ReviewInput, outputDir string, filter commentFilterOpts) error {
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

	sp := spinner.New(cmd.OutOrStdout(), fmt.Sprintf("→ Reviewing with %d agents...", len(agents)))
	sp.Start()

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
	sp.Stop()

	// Check for errors
	outputs := make(map[string]*writer.AgentOutput)
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Agent %s failed: %v\n", r.name, r.err)
			continue
		}
		r.output.Comments = filterComments(r.output.Comments, filter)
		outputs[r.name] = &writer.AgentOutput{
			Output: r.output,
			Model:  r.model,
		}
	}

	if len(outputs) == 0 {
		return fmt.Errorf("all agents failed")
	}

	reviewDir, err := writer.WriteMulti(outputs, writer.WriteMultiOptions{
		BaseDir:    outputDir,
		PRNumber:   input.PRNumber,
		BaseBranch: input.BaseBranch,
		HeadBranch: input.HeadBranch,
	})
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

// commentFilterOpts bundles all comment filtering criteria.
type commentFilterOpts struct {
	allowedSeverities []string // from config output.severities
	noPraise          bool     // --no-praise flag
	minSeverity       string   // --min-severity flag
}

func filterComments(comments []agent.ReviewComment, opts commentFilterOpts) []agent.ReviewComment {
	// Build allowed set from config severities.
	allowed := make(map[string]bool, len(opts.allowedSeverities))
	for _, s := range opts.allowedSeverities {
		allowed[s] = true
	}

	severityOrder := map[string]int{
		"critical":   0,
		"suggestion": 1,
		"nit":        2,
		"praise":     3,
	}

	minLevel := -1
	if opts.minSeverity != "" {
		if lvl, ok := severityOrder[opts.minSeverity]; ok {
			minLevel = lvl
		}
	}

	var filtered []agent.ReviewComment
	for _, c := range comments {
		// Config-level severity filter.
		if len(allowed) > 0 && !allowed[c.Severity] {
			continue
		}
		if opts.noPraise && c.Severity == "praise" {
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
