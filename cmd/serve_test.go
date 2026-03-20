package cmd

import (
	"testing"
)

func TestNewServeCmd(t *testing.T) {
	cmd := newServeCmd()

	if cmd.Use != "serve" {
		t.Errorf("expected Use 'serve', got %q", cmd.Use)
	}

	// Check flags exist with correct defaults.
	portFlag := cmd.Flags().Lookup("port")
	if portFlag == nil {
		t.Fatal("expected --port flag")
	}
	if portFlag.DefValue != "8600" {
		t.Errorf("expected default port 8600, got %s", portFlag.DefValue)
	}

	openFlag := cmd.Flags().Lookup("open")
	if openFlag == nil {
		t.Fatal("expected --open flag")
	}
	if openFlag.DefValue != "false" {
		t.Errorf("expected default open false, got %s", openFlag.DefValue)
	}
}

func TestServeCmd_RegisteredInRoot(t *testing.T) {
	root := newRootCmd("test")

	found := false
	for _, sub := range root.Commands() {
		if sub.Use == "serve" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'serve' subcommand to be registered on root")
	}
}
