package agent

import (
	"fmt"

	"github.com/dotbrains/prr/internal/config"
)

// ProviderFactory creates an Agent from a name and config.
type ProviderFactory func(name string, cfg config.AgentConfig) (Agent, error)

var providers = map[string]ProviderFactory{}

// RegisterProvider registers a new provider factory by name.
func RegisterProvider(name string, factory ProviderFactory) {
	providers[name] = factory
}

// NewAgent creates an agent from the config, looking up the provider factory.
func NewAgent(name string, cfg config.AgentConfig) (Agent, error) {
	factory, ok := providers[cfg.Provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q for agent %q (available: %s)", cfg.Provider, name, availableProviders())
	}
	return factory(name, cfg)
}

// NewAgentFromConfig creates the default (or named) agent from the full config.
func NewAgentFromConfig(agentName string, cfg *config.Config) (Agent, error) {
	if agentName == "" {
		agentName = cfg.DefaultAgent
	}
	agentCfg, ok := cfg.Agents[agentName]
	if !ok {
		return nil, fmt.Errorf("agent %q not found in config (available: %s)", agentName, availableAgents(cfg))
	}
	return NewAgent(agentName, agentCfg)
}

// AllAgentsFromConfig creates all configured agents.
func AllAgentsFromConfig(cfg *config.Config) ([]Agent, error) {
	var agents []Agent
	for name, agentCfg := range cfg.Agents {
		a, err := NewAgent(name, agentCfg)
		if err != nil {
			return nil, fmt.Errorf("creating agent %q: %w", name, err)
		}
		agents = append(agents, a)
	}
	return agents, nil
}

// AvailableProviders returns the list of registered provider names.
func AvailableProviders() []string {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}

func availableProviders() string {
	names := AvailableProviders()
	if len(names) == 0 {
		return "none"
	}
	result := ""
	for i, n := range names {
		if i > 0 {
			result += ", "
		}
		result += n
	}
	return result
}

func availableAgents(cfg *config.Config) string {
	result := ""
	i := 0
	for name := range cfg.Agents {
		if i > 0 {
			result += ", "
		}
		result += name
		i++
	}
	if result == "" {
		return "none"
	}
	return result
}
