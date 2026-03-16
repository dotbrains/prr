package claudecli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dotbrains/prr/internal/agent"
	"github.com/dotbrains/prr/internal/config"
	"github.com/dotbrains/prr/internal/exec"
)

// ClaudeCLI implements agent.Agent using the claude CLI binary.
type ClaudeCLI struct {
	name  string
	model string
	exec  exec.CommandExecutor
}

func init() {
	agent.RegisterProvider("claude-cli", func(name string, cfg config.AgentConfig) (agent.Agent, error) {
		return New(name, cfg, exec.NewRealExecutor())
	})
}

// New creates a new Claude CLI agent. Accepts an executor for testability.
func New(name string, cfg config.AgentConfig, executor exec.CommandExecutor) (agent.Agent, error) {
	model := cfg.Model
	if model == "" {
		model = "opus"
	}
	return &ClaudeCLI{
		name:  name,
		model: model,
		exec:  executor,
	}, nil
}

func (c *ClaudeCLI) Name() string { return c.name }

func (c *ClaudeCLI) Review(ctx context.Context, input *agent.ReviewInput) (*agent.ReviewOutput, error) {
	systemPrompt := agent.BuildSystemPrompt(input.FocusModes...)
	userPrompt := agent.BuildUserPrompt(input)

	text, err := c.call(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	return agent.ParseReviewJSON(text)
}

func (c *ClaudeCLI) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return c.call(ctx, systemPrompt, userPrompt)
}

func (c *ClaudeCLI) call(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	out, err := c.exec.RunWithStdin(ctx, userPrompt,
		"claude", "-p",
		"--output-format", "json",
		"--system-prompt", systemPrompt,
		"--model", c.model,
	)
	if err != nil {
		return "", fmt.Errorf("claude CLI failed: %w", err)
	}

	// claude -p --output-format json returns a JSON wrapper:
	// {"type":"result","result":"...","is_error":false,...}
	var cliResp struct {
		Type    string `json:"type"`
		Result  string `json:"result"`
		IsError bool   `json:"is_error"`
	}
	if err := json.Unmarshal([]byte(out), &cliResp); err != nil || cliResp.Type == "" {
		// Not a claude JSON wrapper — return raw output
		return out, nil
	}

	if cliResp.IsError {
		return "", fmt.Errorf("claude CLI error: %s", agent.Truncate(cliResp.Result, 500))
	}

	return cliResp.Result, nil
}
