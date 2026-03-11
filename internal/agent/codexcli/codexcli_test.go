package codexcli

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

func TestCodexCLI_Review_ResultEvent(t *testing.T) {
	reviewJSON := `{"summary":"Good PR","comments":[{"file":"main.go","start_line":1,"end_line":1,"severity":"nit","body":"Rename this"}]}`
	// Codex outputs JSONL with a result event
	jsonl := fmt.Sprintf("{\"type\":\"status\",\"status\":\"running\"}\n{\"type\":\"result\",\"result\":%q}\n", reviewJSON)

	mock := &mockExecutor{output: jsonl}
	a, err := New("test-codex", config.AgentConfig{Model: "codex"}, mock)
	if err != nil {
		t.Fatalf("unexpected error creating agent: %v", err)
	}

	if a.Name() != "test-codex" {
		t.Errorf("expected name 'test-codex', got %q", a.Name())
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

	// Verify system prompt is embedded in stdin
	if !strings.Contains(mock.stdinReceived, "SYSTEM INSTRUCTIONS:") {
		t.Error("expected system prompt to be embedded in stdin")
	}
}

func TestCodexCLI_Review_MessageEvent(t *testing.T) {
	reviewJSON := `{"summary":"test","comments":[]}`
	// Codex outputs a message event instead of result
	jsonl := fmt.Sprintf("{\"type\":\"message\",\"role\":\"assistant\",\"content\":%q}\n", reviewJSON)

	mock := &mockExecutor{output: jsonl}
	a, _ := New("test", config.AgentConfig{}, mock)

	input := &agent.ReviewInput{PRNumber: 1, PRTitle: "Test", BaseBranch: "main", HeadBranch: "test", Diff: "diff"}
	output, err := a.Review(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Summary != "test" {
		t.Errorf("expected summary 'test', got %q", output.Summary)
	}
}

func TestCodexCLI_Review_CLIError(t *testing.T) {
	mock := &mockExecutor{err: fmt.Errorf("codex: command not found")}
	a, _ := New("test", config.AgentConfig{}, mock)

	input := &agent.ReviewInput{PRNumber: 1, PRTitle: "Test", BaseBranch: "main", HeadBranch: "test", Diff: "diff"}
	_, err := a.Review(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when codex CLI fails")
	}
	if !strings.Contains(err.Error(), "codex CLI failed") {
		t.Errorf("expected 'codex CLI failed' in error, got: %v", err)
	}
}

func TestCodexCLI_Review_EmptyOutput(t *testing.T) {
	mock := &mockExecutor{output: ""}
	a, _ := New("test", config.AgentConfig{}, mock)

	input := &agent.ReviewInput{PRNumber: 1, PRTitle: "Test", BaseBranch: "main", HeadBranch: "test", Diff: "diff"}
	_, err := a.Review(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for empty output")
	}
}

func TestCodexCLI_Review_FallbackRawOutput(t *testing.T) {
	// Non-JSONL output falls back to treating entire output as response
	rawJSON := `{"summary":"fallback","comments":[]}`
	mock := &mockExecutor{output: rawJSON}
	a, _ := New("test", config.AgentConfig{}, mock)

	input := &agent.ReviewInput{PRNumber: 1, PRTitle: "Test", BaseBranch: "main", HeadBranch: "test", Diff: "diff"}
	output, err := a.Review(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Summary != "fallback" {
		t.Errorf("expected 'fallback', got %q", output.Summary)
	}
}

func TestCodexCLI_DefaultModel(t *testing.T) {
	mock := &mockExecutor{output: `{"summary":"","comments":[]}`}
	a, _ := New("test", config.AgentConfig{Model: ""}, mock)

	cli := a.(*CodexCLI)
	if cli.model != "codex" {
		t.Errorf("expected default model 'codex', got %q", cli.model)
	}
}

func TestExtractCodexResult_ResultEvent(t *testing.T) {
	input := `{"type":"status","status":"running"}
{"type":"result","result":"hello world"}
`
	result, err := extractCodexResult(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %q", result)
	}
}

func TestExtractCodexResult_MessageEvent(t *testing.T) {
	input := `{"type":"message","content":"response text"}
`
	result, err := extractCodexResult(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "response text" {
		t.Errorf("expected 'response text', got %q", result)
	}
}

func TestExtractCodexResult_Empty(t *testing.T) {
	_, err := extractCodexResult("")
	if err == nil {
		t.Fatal("expected error for empty output")
	}
}
