package exec

import (
	"context"
	"testing"
)

func TestRealExecutor_Run(t *testing.T) {
	e := NewRealExecutor()
	out, err := e.Run(context.Background(), "echo", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello\n" {
		t.Errorf("expected 'hello\\n', got %q", out)
	}
}

func TestRealExecutor_Run_Error(t *testing.T) {
	e := NewRealExecutor()
	_, err := e.Run(context.Background(), "false")
	if err == nil {
		t.Fatal("expected error from 'false' command")
	}
}

func TestRealExecutor_RunWithStdin(t *testing.T) {
	e := NewRealExecutor()
	out, err := e.RunWithStdin(context.Background(), "hello from stdin", "cat")
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello from stdin" {
		t.Errorf("expected input echoed back, got %q", out)
	}
}

func TestRealExecutor_RunWithStdin_Error(t *testing.T) {
	e := NewRealExecutor()
	_, err := e.RunWithStdin(context.Background(), "", "false")
	if err == nil {
		t.Fatal("expected error from 'false' command")
	}
}
