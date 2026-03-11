package agent

import (
	"context"
	"testing"

	"github.com/dotbrains/prr/internal/config"
)

func TestRegisterAndNewAgent(t *testing.T) {
	// Register a test provider
	RegisterProvider("test-provider", func(name string, cfg config.AgentConfig) (Agent, error) {
		return &mockAgent{name: name}, nil
	})
	defer delete(providers, "test-provider")

	cfg := config.AgentConfig{Provider: "test-provider", Model: "test-model"}
	agent, err := NewAgent("test", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.Name() != "test" {
		t.Errorf("expected name test, got %q", agent.Name())
	}
}

func TestNewAgent_UnknownProvider(t *testing.T) {
	cfg := config.AgentConfig{Provider: "nonexistent"}
	_, err := NewAgent("test", cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestNewAgentFromConfig(t *testing.T) {
	RegisterProvider("test-provider", func(name string, cfg config.AgentConfig) (Agent, error) {
		return &mockAgent{name: name}, nil
	})
	defer delete(providers, "test-provider")

	cfg := &config.Config{
		DefaultAgent: "myagent",
		Agents: map[string]config.AgentConfig{
			"myagent": {Provider: "test-provider", Model: "m1"},
		},
	}

	// Use default
	agent, err := NewAgentFromConfig("", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.Name() != "myagent" {
		t.Errorf("expected myagent, got %q", agent.Name())
	}

	// Use explicit name
	agent, err = NewAgentFromConfig("myagent", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.Name() != "myagent" {
		t.Errorf("expected myagent, got %q", agent.Name())
	}
}

func TestNewAgentFromConfig_NotFound(t *testing.T) {
	cfg := &config.Config{
		DefaultAgent: "missing",
		Agents:       map[string]config.AgentConfig{},
	}

	_, err := NewAgentFromConfig("", cfg)
	if err == nil {
		t.Fatal("expected error for missing agent")
	}
}

func TestAvailableProviders(t *testing.T) {
	RegisterProvider("test-a", func(name string, cfg config.AgentConfig) (Agent, error) {
		return nil, nil
	})
	defer delete(providers, "test-a")

	names := AvailableProviders()
	found := false
	for _, n := range names {
		if n == "test-a" {
			found = true
		}
	}
	if !found {
		t.Error("expected test-a in available providers")
	}
}

// mockAgent is a simple test implementation of Agent.
type mockAgent struct {
	name string
}

func (m *mockAgent) Name() string { return m.name }
func (m *mockAgent) Review(_ context.Context, _ *ReviewInput) (*ReviewOutput, error) {
	return &ReviewOutput{Summary: "mock review"}, nil
}
