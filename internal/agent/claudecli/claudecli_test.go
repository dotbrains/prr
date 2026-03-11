package claudecli

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/dotbrains/prr/internal/agent"
	"github.com/dotbrains/prr/internal/config"
)

type mockExecutor struct {
	stdinReceived string
	output        string
	err           error
}

func (m *mockExecutor) Run(ctx context.Context, name string, args ...string) (string, error) {
	return m.output, m.err
}

func (m *mockExecutor) RunWithStdin(ctx context.Context, stdin string, name string, args ...string) (string, error) {
	m.stdinReceived = stdin
	return m.output, m.err
}

func TestClaudeCLI_Review_Success(t *testing.T) {
	reviewJSON := `{"summary":"Good PR","comments":[{"file":"main.go","start_line":1,"end_line":1,"severity":"nit","body":"Rename this"}]}`
	cliOutput := fmt.Sprintf(`{"type":"result","result":%q,"is_error":false}`, reviewJSON)

	mock := &mockExecutor{output: cliOutput}
	a, err := New("test-claude", config.AgentConfig{Model: "sonnet"}, mock)
	if err != nil {
		t.Fatalf("unexpected error creating agent: %v", err)
	}

	if a.Name() != "test-claude" {
		t.Errorf("expected name 'test-claude', got %q", a.Name())
	}

	input := &agent.ReviewInput{
		PRNumber:   42,
		PRTitle:    "Fix bug",
		BaseBranch: "main",
		HeadBranch: "fix-bug",
		Diff:       "some diff",
	}

	output, err := a.Review(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Summary != "Good PR" {
		t.Errorf("expected summary 'Good PR', got %q", output.Summary)
	}
	if len(output.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(output.Comments))
	}

	// Verify stdin was sent
	if mock.stdinReceived == "" {
		t.Error("expected stdin to be sent to claude CLI")
	}
}

func TestClaudeCLI_Review_CLIError(t *testing.T) {
	mock := &mockExecutor{err: fmt.Errorf("claude: command not found")}
	a, _ := New("test", config.AgentConfig{}, mock)

	input := &agent.ReviewInput{PRNumber: 1, PRTitle: "Test", BaseBranch: "main", HeadBranch: "test", Diff: "diff"}
	_, err := a.Review(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when claude CLI fails")
	}
	if !strings.Contains(err.Error(), "claude CLI failed") {
		t.Errorf("expected 'claude CLI failed' in error, got: %v", err)
	}
}

func TestClaudeCLI_Review_IsError(t *testing.T) {
	cliOutput := `{"type":"result","result":"rate limited","is_error":true}`
	mock := &mockExecutor{output: cliOutput}
	a, _ := New("test", config.AgentConfig{}, mock)

	input := &agent.ReviewInput{PRNumber: 1, PRTitle: "Test", BaseBranch: "main", HeadBranch: "test", Diff: "diff"}
	_, err := a.Review(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when is_error is true")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("expected 'rate limited' in error, got: %v", err)
	}
}

func TestClaudeCLI_Review_RawJSONFallback(t *testing.T) {
	// If the output isn't a JSON wrapper, fall back to parsing as review JSON directly
	rawJSON := `{"summary":"Direct output","comments":[]}`
	mock := &mockExecutor{output: rawJSON}
	a, _ := New("test", config.AgentConfig{}, mock)

	input := &agent.ReviewInput{PRNumber: 1, PRTitle: "Test", BaseBranch: "main", HeadBranch: "test", Diff: "diff"}
	output, err := a.Review(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Summary != "Direct output" {
		t.Errorf("expected 'Direct output', got %q", output.Summary)
	}
}

func TestClaudeCLI_DefaultModel(t *testing.T) {
	mock := &mockExecutor{output: `{"summary":"","comments":[]}`}
	a, _ := New("test", config.AgentConfig{Model: ""}, mock)

	cli := a.(*ClaudeCLI)
	if cli.model != "sonnet" {
		t.Errorf("expected default model 'sonnet', got %q", cli.model)
	}
}
