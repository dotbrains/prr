package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dotbrains/prr/internal/agent"
	"github.com/dotbrains/prr/internal/config"
)

func newTestClaude(t *testing.T, handler http.HandlerFunc) *Claude {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	t.Setenv("TEST_ANTHROPIC_KEY", "test-key")
	a, err := New("test-claude", config.AgentConfig{
		Provider:  "anthropic",
		Model:     "test-model",
		APIKeyEnv: "TEST_ANTHROPIC_KEY",
		MaxTokens: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	c := a.(*Claude)
	c.SetBaseURL(srv.URL)
	return c
}

func TestNew_MissingAPIKey(t *testing.T) {
	t.Setenv("MISSING_KEY", "")
	_, err := New("test", config.AgentConfig{
		Provider:  "anthropic",
		Model:     "test",
		APIKeyEnv: "MISSING_KEY",
	})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestNew_DefaultMaxTokens(t *testing.T) {
	t.Setenv("TEST_KEY", "key")
	a, err := New("test", config.AgentConfig{
		Provider:  "anthropic",
		Model:     "test",
		APIKeyEnv: "TEST_KEY",
		MaxTokens: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	c := a.(*Claude)
	if c.maxTokens != 8192 {
		t.Errorf("expected default 8192, got %d", c.maxTokens)
	}
}

func TestClaude_Name(t *testing.T) {
	t.Setenv("TEST_KEY", "key")
	a, _ := New("my-claude", config.AgentConfig{
		Provider:  "anthropic",
		Model:     "test",
		APIKeyEnv: "TEST_KEY",
	})
	if a.Name() != "my-claude" {
		t.Errorf("expected my-claude, got %q", a.Name())
	}
}

func TestClaude_Review_Success(t *testing.T) {
	reviewJSON := `{"summary":"looks good","comments":[{"path":"main.go","line":10,"severity":"suggestion","body":"add error handling"}]}`
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("missing api key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("missing anthropic-version header")
		}
		resp := messagesResponse{
			Content: []contentBlock{
				{Type: "text", Text: reviewJSON},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}

	c := newTestClaude(t, handler)
	input := &agent.ReviewInput{
		PRNumber: 1,
		PRTitle:  "test",
		Diff:     "diff content",
	}

	out, err := c.Review(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if out.Summary != "looks good" {
		t.Errorf("expected summary 'looks good', got %q", out.Summary)
	}
	if len(out.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(out.Comments))
	}
	if out.Comments[0].Severity != "suggestion" {
		t.Errorf("expected severity suggestion, got %q", out.Comments[0].Severity)
	}
}

func TestClaude_Review_APIError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}

	c := newTestClaude(t, handler)
	_, err := c.Review(context.Background(), &agent.ReviewInput{PRNumber: 1, Diff: "d"})
	if err == nil {
		t.Fatal("expected error for 429")
	}
}

func TestClaude_Review_EmptyResponse(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := messagesResponse{Content: []contentBlock{}}
		_ = json.NewEncoder(w).Encode(resp)
	}

	c := newTestClaude(t, handler)
	_, err := c.Review(context.Background(), &agent.ReviewInput{PRNumber: 1, Diff: "d"})
	if err == nil {
		t.Fatal("expected error for empty response")
	}
}

func TestClaude_Review_InvalidJSON(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}

	c := newTestClaude(t, handler)
	_, err := c.Review(context.Background(), &agent.ReviewInput{PRNumber: 1, Diff: "d"})
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestClaude_SetClient(t *testing.T) {
	t.Setenv("TEST_KEY", "key")
	a, _ := New("test", config.AgentConfig{
		Provider:  "anthropic",
		Model:     "test",
		APIKeyEnv: "TEST_KEY",
	})
	c := a.(*Claude)
	custom := &http.Client{}
	c.SetClient(custom)
	if c.client != custom {
		t.Error("SetClient did not set the client")
	}
}

func TestExtractText(t *testing.T) {
	tests := []struct {
		name string
		resp messagesResponse
		want string
	}{
		{"text block", messagesResponse{Content: []contentBlock{{Type: "text", Text: "hello"}}}, "hello"},
		{"empty content", messagesResponse{Content: []contentBlock{}}, ""},
		{"non-text block", messagesResponse{Content: []contentBlock{{Type: "image", Text: ""}}}, ""},
		{"multiple blocks", messagesResponse{Content: []contentBlock{{Type: "image"}, {Type: "text", Text: "found"}}}, "found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractText(tt.resp)
			if got != tt.want {
				t.Errorf("extractText() = %q, want %q", got, tt.want)
			}
		})
	}
}
