package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	flagAgent        string
	flagAll          bool
	flagOutputDir    string
	flagRepo         string
	flagBase         string
	flagHead         string
	flagNoContext    bool
	flagFocus        string
	flagSince        string
	flagVerify       bool
	flagNoVerify     bool
	flagVerifyAgent  string
	flagVerifyAction string
)

func newRootCmd(version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "prr [PR_NUMBER]",
		Short: "AI-powered PR code review CLI",
		Long:  "Run AI-powered code reviews on GitHub pull requests and output structured, human-readable markdown comments.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runReview,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		Version: version,
	}

	root.SetVersionTemplate(fmt.Sprintf("prr version %s\n", version))

	// Global flags
	root.PersistentFlags().StringVar(&flagAgent, "agent", "", "use a specific configured agent")
	root.PersistentFlags().BoolVar(&flagAll, "all", false, "run review with all configured agents")
	root.PersistentFlags().StringVar(&flagOutputDir, "output-dir", "", "override output directory")

	// Local repo flags
	root.PersistentFlags().StringVar(&flagRepo, "repo", "", "path to a local git repo (enables local mode)")
	root.PersistentFlags().StringVar(&flagBase, "base", "", "base branch to diff against (enables local mode)")
	root.PersistentFlags().StringVar(&flagHead, "head", "", "head branch (defaults to current branch)")

	// Review-specific flags
	root.Flags().Bool("no-praise", false, "skip positive/praise comments")
	root.Flags().String("min-severity", "", "minimum severity to include (critical, suggestion, nit)")
	root.Flags().BoolVar(&flagNoContext, "no-context", false, "disable codebase pattern context")
	root.Flags().StringVar(&flagFocus, "focus", "", "focus review on specific areas (security,performance,testing)")
	root.Flags().StringVar(&flagSince, "since", "", "incremental review: only changes since last review or a commit SHA")
	root.Flags().BoolVar(&flagVerify, "verify", false, "verify each comment's accuracy (enabled by default)")
	root.Flags().BoolVar(&flagNoVerify, "no-verify", false, "disable comment verification")
	root.Flags().StringVar(&flagVerifyAgent, "verify-agent", "", "agent to use for verification (defaults to review agent)")
	root.Flags().StringVar(&flagVerifyAction, "verify-action", "", "action for inaccurate comments: drop (default) or annotate")

	// Subcommands
	root.AddCommand(newAgentsCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newHistoryCmd())
	root.AddCommand(newCleanCmd())
	root.AddCommand(newPostCmd())
	root.AddCommand(newDescribeCmd())
	root.AddCommand(newAskCmd())
	root.AddCommand(newReviewDiffCmd())
	root.AddCommand(newServeCmd())

	return root
}

// Execute runs the root command.
func Execute(version string) error {
	return newRootCmd(version).Execute()
}
