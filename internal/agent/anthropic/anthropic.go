package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/dotbrains/prr/internal/agent"
	"github.com/dotbrains/prr/internal/config"
)

const defaultBaseURL = "https://api.anthropic.com"

// Claude implements agent.Agent using the Anthropic Messages API.
type Claude struct {
	name      string
	model     string
	apiKey    string
	maxTokens int
	baseURL   string
	client    *http.Client
}

func init() {
	agent.RegisterProvider("anthropic", New)
}

// New creates a new Claude agent from config.
func New(name string, cfg config.AgentConfig) (agent.Agent, error) {
	apiKey := os.Getenv(cfg.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("environment variable %s is not set", cfg.APIKeyEnv)
	}

	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 8192
	}

	return &Claude{
		name:      name,
		model:     cfg.Model,
		apiKey:    apiKey,
		maxTokens: maxTokens,
		baseURL:   defaultBaseURL,
		client:    &http.Client{},
	}, nil
}

func (c *Claude) Name() string { return c.name }

func (c *Claude) Review(ctx context.Context, input *agent.ReviewInput) (*agent.ReviewOutput, error) {
	systemPrompt := agent.BuildSystemPrompt(input.FocusModes...)
	userPrompt := agent.BuildUserPrompt(input)

	text, err := c.call(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	return agent.ParseReviewJSON(text)
}

func (c *Claude) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return c.call(ctx, systemPrompt, userPrompt)
}

func (c *Claude) call(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	body := messagesRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		System:    systemPrompt,
		Messages: []message{
			{Role: "user", Content: userPrompt},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var messagesResp messagesResponse
	if err := json.Unmarshal(respBody, &messagesResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	text := extractText(messagesResp)
	if text == "" {
		return "", fmt.Errorf("no text content in response")
	}

	return text, nil
}

// SetBaseURL overrides the API base URL (for testing).
func (c *Claude) SetBaseURL(url string) {
	c.baseURL = url
}

// SetClient overrides the HTTP client (for testing).
func (c *Claude) SetClient(client *http.Client) {
	c.client = client
}

// --- Anthropic API types ---

type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesResponse struct {
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func extractText(resp messagesResponse) string {
	for _, block := range resp.Content {
		if block.Type == "text" && block.Text != "" {
			return block.Text
		}
	}
	return ""
}

