package openai

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

const defaultBaseURL = "https://api.openai.com"

// GPT implements agent.Agent using the OpenAI Chat Completions API.
type GPT struct {
	name      string
	model     string
	apiKey    string
	maxTokens int
	baseURL   string
	client    *http.Client
}

func init() {
	agent.RegisterProvider("openai", New)
}

// New creates a new GPT agent from config.
func New(name string, cfg config.AgentConfig) (agent.Agent, error) {
	apiKey := os.Getenv(cfg.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("environment variable %s is not set", cfg.APIKeyEnv)
	}

	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 8192
	}

	return &GPT{
		name:      name,
		model:     cfg.Model,
		apiKey:    apiKey,
		maxTokens: maxTokens,
		baseURL:   defaultBaseURL,
		client:    &http.Client{},
	}, nil
}

func (g *GPT) Name() string { return g.name }

func (g *GPT) Review(ctx context.Context, input *agent.ReviewInput) (*agent.ReviewOutput, error) {
	systemPrompt := agent.BuildSystemPrompt()
	userPrompt := agent.BuildUserPrompt(input)

	body := chatRequest{
		Model: g.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxCompletionTokens: g.maxTokens,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.baseURL+"/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	text := extractText(chatResp)
	if text == "" {
		return nil, fmt.Errorf("no text content in response")
	}

	return agent.ParseReviewJSON(text)
}

// SetBaseURL overrides the API base URL (for testing).
func (g *GPT) SetBaseURL(url string) {
	g.baseURL = url
}

// SetClient overrides the HTTP client (for testing).
func (g *GPT) SetClient(client *http.Client) {
	g.client = client
}

// --- OpenAI API types ---

type chatRequest struct {
	Model               string        `json:"model"`
	Messages            []chatMessage `json:"messages"`
	MaxCompletionTokens int           `json:"max_completion_tokens,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

func extractText(resp chatResponse) string {
	if len(resp.Choices) == 0 {
		return ""
	}
	return resp.Choices[0].Message.Content
}

