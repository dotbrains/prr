package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dotbrains/prr/internal/config"
	"github.com/dotbrains/prr/internal/writer"
)

func newHistoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "List past reviews",
		RunE:  runHistory,
	}
}

func runHistory(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	outputDir := cfg.Output.Dir
	if flagOutputDir != "" {
		outputDir = flagOutputDir
	}

	entries, err := writer.ListReviewDirs(outputDir)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No reviews found.")
		return nil
	}

	for _, e := range entries {
		fmt.Fprintf(cmd.OutOrStdout(), "  %-40s  %s\n", e.Name, e.ModTime.Format("2006-01-02 15:04"))
	}

	return nil
}
