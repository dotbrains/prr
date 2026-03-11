package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/dotbrains/prr/internal/config"
	"github.com/dotbrains/prr/internal/writer"
)

func newCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove old review output",
		RunE:  runClean,
	}
	cmd.Flags().Int("days", 30, "remove reviews older than N days")
	cmd.Flags().Bool("dry-run", false, "show what would be removed without deleting")
	return cmd
}

func runClean(cmd *cobra.Command, args []string) error {
	days, _ := cmd.Flags().GetInt("days")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	outputDir := cfg.Output.Dir
	if flagOutputDir != "" {
		outputDir = flagOutputDir
	}

	maxAge := time.Duration(days) * 24 * time.Hour

	if dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "→ Dry run: showing reviews older than %d days...\n", days)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "→ Removing reviews older than %d days...\n", days)
	}

	removed, err := writer.CleanOlderThan(outputDir, maxAge, dryRun)
	if err != nil {
		return err
	}

	if len(removed) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No reviews to clean up.")
		return nil
	}

	for _, name := range removed {
		if dryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "  would remove: %s\n", name)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  removed: %s\n", name)
		}
	}

	if dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "✓ Would clean up %d review directories.\n", len(removed))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "✓ Cleaned up %d review directories.\n", len(removed))
	}

	return nil
}
