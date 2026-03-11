package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dotbrains/prr/internal/config"
)

func newConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage prr configuration",
	}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Create a default config file",
		RunE:  runConfigInit,
	}
	initCmd.Flags().Bool("force", false, "overwrite existing config file")

	configCmd.AddCommand(initCmd)
	return configCmd
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	exists, err := config.Exists()
	if err != nil {
		return fmt.Errorf("checking config: %w", err)
	}

	if exists && !force {
		path, _ := config.ConfigPath()
		return fmt.Errorf("config already exists at %s\nUse --force to overwrite", path)
	}

	cfg := config.DefaultConfig()
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	path, _ := config.ConfigPath()
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Wrote default config to %s\n", path)
	fmt.Fprintln(cmd.OutOrStdout(), "Edit the file to add your API keys and customize agents.")
	return nil
}
