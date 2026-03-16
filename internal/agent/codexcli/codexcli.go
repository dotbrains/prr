package codexcli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dotbrains/prr/internal/agent"
	"github.com/dotbrains/prr/internal/config"
	"github.com/dotbrains/prr/internal/exec"
)

// CodexCLI implements agent.Agent using the OpenAI Codex CLI binary.
type CodexCLI struct {
	name  string
	model string
	exec  exec.CommandExecutor
}

func init() {
	agent.RegisterProvider("codex-cli", func(name string, cfg config.AgentConfig) (agent.Agent, error) {
		return New(name, cfg, exec.NewRealExecutor())
	})
}

// New creates a new Codex CLI agent. Accepts an executor for testability.
func New(name string, cfg config.AgentConfig, executor exec.CommandExecutor) (agent.Agent, error) {
	model := cfg.Model
	if model == "" {
		model = "codex"
	}
	return &CodexCLI{
		name:  name,
		model: model,
		exec:  executor,
	}, nil
}

func (c *CodexCLI) Name() string { return c.name }

func (c *CodexCLI) Review(ctx context.Context, input *agent.ReviewInput) (*agent.ReviewOutput, error) {
	systemPrompt := agent.BuildSystemPrompt(input.FocusModes...)
	userPrompt := agent.BuildUserPrompt(input)

	text, err := c.call(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	return agent.ParseReviewJSON(text)
}

func (c *CodexCLI) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return c.call(ctx, systemPrompt, userPrompt)
}

func (c *CodexCLI) call(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// Codex CLI doesn't have a --system-prompt flag, so we embed it in the user prompt.
	combinedPrompt := fmt.Sprintf("SYSTEM INSTRUCTIONS:\n%s\n\nUSER REQUEST:\n%s", systemPrompt, userPrompt)

	out, err := c.exec.RunWithStdin(ctx, combinedPrompt,
		"codex", "exec",
		"--json",
		"--approval-mode", "suggest",
		"--skip-git-repo-check",
		"-",
	)
	if err != nil {
		return "", fmt.Errorf("codex CLI failed: %w", err)
	}

	text, err := extractCodexResult(out)
	if err != nil {
		return "", err
	}

	return text, nil
}

// extractCodexResult parses JSONL events from codex exec --json and extracts
// the final assistant message text.
func extractCodexResult(output string) (string, error) {
	var lastMessage string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // skip non-JSON lines
		}

		// Look for message events with assistant content
		if eventType, ok := event["type"].(string); ok {
			switch eventType {
			case "message":
				if content, ok := event["content"].(string); ok && content != "" {
					lastMessage = content
				}
				// Handle nested content in role-based messages
				if role, ok := event["role"].(string); ok && role == "assistant" {
					if content, ok := event["content"].(string); ok && content != "" {
						lastMessage = content
					}
				}
			case "result":
				// Final result event
				if result, ok := event["result"].(string); ok && result != "" {
					return result, nil
				}
			}
		}
	}

	if lastMessage != "" {
		return lastMessage, nil
	}

	// Fallback: treat entire output as the response text
	output = strings.TrimSpace(output)
	if output == "" {
		return "", fmt.Errorf("no response from codex CLI")
	}
	return output, nil
}
