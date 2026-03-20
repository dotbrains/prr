package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/dotbrains/prr/internal/config"
	"github.com/dotbrains/prr/internal/server"
)

func newServeCmd() *cobra.Command {
	var port int
	var open bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start a local web UI to browse reviews",
		Long:  "Starts a local HTTP server that serves a web UI for browsing past reviews. All assets are embedded in the binary — no external dependencies required.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd, port, open)
		},
	}

	cmd.Flags().IntVar(&port, "port", 8600, "port to serve on")
	cmd.Flags().BoolVar(&open, "open", false, "open browser automatically")

	return cmd
}

func runServe(cmd *cobra.Command, port int, open bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	outputDir := cfg.Output.Dir
	if flagOutputDir != "" {
		outputDir = flagOutputDir
	}

	addr := fmt.Sprintf(":%d", port)
	url := fmt.Sprintf("http://localhost:%d", port)

	fmt.Fprintf(cmd.OutOrStdout(), "→ Serving reviews at %s\n", url)
	fmt.Fprintf(cmd.OutOrStdout(), "→ Reviews dir: %s\n", outputDir)
	fmt.Fprintf(cmd.OutOrStdout(), "→ Press Ctrl+C to stop\n")

	if open {
		openBrowser(url)
	}

	srv := server.New(outputDir)
	return srv.ListenAndServe(addr)
}

// openBrowser opens the given URL in the default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	_ = cmd.Start()
}
