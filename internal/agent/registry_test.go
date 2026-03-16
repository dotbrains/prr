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

func TestAllAgentsFromConfig(t *testing.T) {
	RegisterProvider("test-all", func(name string, cfg config.AgentConfig) (Agent, error) {
		return &mockAgent{name: name}, nil
	})
	defer delete(providers, "test-all")

	cfg := &config.Config{
		Agents: map[string]config.AgentConfig{
			"a1": {Provider: "test-all"},
			"a2": {Provider: "test-all"},
		},
	}

	agents, err := AllAgentsFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
}

func TestAllAgentsFromConfig_Error(t *testing.T) {
	cfg := &config.Config{
		Agents: map[string]config.AgentConfig{
			"bad": {Provider: "nonexistent-provider"},
		},
	}

	_, err := AllAgentsFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestAvailableProviders_Empty(t *testing.T) {
	// Save and clear providers.
	saved := make(map[string]ProviderFactory)
	for k, v := range providers {
		saved[k] = v
	}
	for k := range providers {
		delete(providers, k)
	}
	defer func() {
		for k, v := range saved {
			providers[k] = v
		}
	}()

	result := availableProviders()
	if result != "none" {
		t.Errorf("expected 'none', got %q", result)
	}
}

func TestAvailableProviders_Multiple(t *testing.T) {
	RegisterProvider("test-p1", func(name string, cfg config.AgentConfig) (Agent, error) {
		return nil, nil
	})
	RegisterProvider("test-p2", func(name string, cfg config.AgentConfig) (Agent, error) {
		return nil, nil
	})
	defer delete(providers, "test-p1")
	defer delete(providers, "test-p2")

	result := availableProviders()
	if result == "none" || result == "" {
		t.Errorf("expected provider names, got %q", result)
	}
}

func TestAvailableAgents_Empty(t *testing.T) {
	cfg := &config.Config{Agents: map[string]config.AgentConfig{}}
	result := availableAgents(cfg)
	if result != "none" {
		t.Errorf("expected 'none', got %q", result)
	}
}

func TestAvailableAgents_Multiple(t *testing.T) {
	cfg := &config.Config{
		Agents: map[string]config.AgentConfig{
			"x": {},
			"y": {},
		},
	}
	result := availableAgents(cfg)
	// Should contain both names.
	if result == "none" || result == "" {
		t.Errorf("expected agent names, got %q", result)
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
func (m *mockAgent) Generate(_ context.Context, _, _ string) (string, error) {
	return "mock generate", nil
}
