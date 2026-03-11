package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	flagAgent     string
	flagAll       bool
	flagOutputDir string
	flagRepo      string
	flagBase      string
	flagHead      string
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

	// Subcommands
	root.AddCommand(newAgentsCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newHistoryCmd())
	root.AddCommand(newCleanCmd())

	return root
}

// Execute runs the root command.
func Execute(version string) error {
	return newRootCmd(version).Execute()
}
