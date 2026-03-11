package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	_ "github.com/dotbrains/prr/internal/agent/claudecli" // register provider
	_ "github.com/dotbrains/prr/internal/agent/codexcli"  // register provider
	"github.com/dotbrains/prr/internal/config"
)

func newAgentsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agents",
		Short: "List configured AI agents",
		RunE:  runAgents,
	}
}

func runAgents(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Agents) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No agents configured.")
		fmt.Fprintln(cmd.OutOrStdout(), "Run 'prr config init' to create a config file.")
		return nil
	}

	for name, agentCfg := range cfg.Agents {
		var keyStatus string
		if config.CLIProviders[agentCfg.Provider] {
			keyStatus = "✓ (cli)"
		} else if agentCfg.APIKeyEnv != "" && os.Getenv(agentCfg.APIKeyEnv) != "" {
			keyStatus = "✓"
		} else {
			keyStatus = "✗ (not set)"
		}

		defaultMark := ""
		if name == cfg.DefaultAgent {
			defaultMark = " (default)"
		}

		keyEnv := agentCfg.APIKeyEnv
		if keyEnv == "" {
			keyEnv = "-"
		}

		fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %-12s %-30s %-20s %s%s\n",
			name, agentCfg.Provider, agentCfg.Model, keyEnv, keyStatus, defaultMark)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nDefault: %s\n", cfg.DefaultAgent)
	return nil
}
