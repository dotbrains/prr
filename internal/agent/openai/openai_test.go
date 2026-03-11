package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dotbrains/prr/internal/agent"
	"github.com/dotbrains/prr/internal/config"
)

func newTestGPT(t *testing.T, handler http.HandlerFunc) *GPT {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	t.Setenv("TEST_OPENAI_KEY", "test-key")
	a, err := New("test-gpt", config.AgentConfig{
		Provider:  "openai",
		Model:     "test-model",
		APIKeyEnv: "TEST_OPENAI_KEY",
		MaxTokens: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	g := a.(*GPT)
	g.SetBaseURL(srv.URL)
	return g
}

func TestNew_MissingAPIKey(t *testing.T) {
	t.Setenv("MISSING_KEY", "")
	_, err := New("test", config.AgentConfig{
		Provider:  "openai",
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
		Provider:  "openai",
		Model:     "test",
		APIKeyEnv: "TEST_KEY",
		MaxTokens: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	g := a.(*GPT)
	if g.maxTokens != 8192 {
		t.Errorf("expected default 8192, got %d", g.maxTokens)
	}
}

func TestGPT_Name(t *testing.T) {
	t.Setenv("TEST_KEY", "key")
	a, _ := New("my-gpt", config.AgentConfig{
		Provider:  "openai",
		Model:     "test",
		APIKeyEnv: "TEST_KEY",
	})
	if a.Name() != "my-gpt" {
		t.Errorf("expected my-gpt, got %q", a.Name())
	}
}

func TestGPT_Review_Success(t *testing.T) {
	reviewJSON := `{"summary":"lgtm","comments":[{"path":"app.go","line":5,"severity":"nit","body":"rename var"}]}`
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing or wrong auth header")
		}
		resp := chatResponse{
			Choices: []chatChoice{
				{Message: chatMessage{Role: "assistant", Content: reviewJSON}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}

	g := newTestGPT(t, handler)
	input := &agent.ReviewInput{
		PRNumber: 42,
		PRTitle:  "test pr",
		Diff:     "some diff",
	}

	out, err := g.Review(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if out.Summary != "lgtm" {
		t.Errorf("expected summary 'lgtm', got %q", out.Summary)
	}
	if len(out.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(out.Comments))
	}
	if out.Comments[0].Severity != "nit" {
		t.Errorf("expected severity nit, got %q", out.Comments[0].Severity)
	}
}

func TestGPT_Review_APIError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"server error"}`))
	}

	g := newTestGPT(t, handler)
	_, err := g.Review(context.Background(), &agent.ReviewInput{PRNumber: 1, Diff: "d"})
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestGPT_Review_EmptyChoices(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{Choices: []chatChoice{}}
		_ = json.NewEncoder(w).Encode(resp)
	}

	g := newTestGPT(t, handler)
	_, err := g.Review(context.Background(), &agent.ReviewInput{PRNumber: 1, Diff: "d"})
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestGPT_Review_InvalidJSON(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`broken`))
	}

	g := newTestGPT(t, handler)
	_, err := g.Review(context.Background(), &agent.ReviewInput{PRNumber: 1, Diff: "d"})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestGPT_SetClient(t *testing.T) {
	t.Setenv("TEST_KEY", "key")
	a, _ := New("test", config.AgentConfig{
		Provider:  "openai",
		Model:     "test",
		APIKeyEnv: "TEST_KEY",
	})
	g := a.(*GPT)
	custom := &http.Client{}
	g.SetClient(custom)
	if g.client != custom {
		t.Error("SetClient did not set the client")
	}
}

func TestExtractText(t *testing.T) {
	tests := []struct {
		name string
		resp chatResponse
		want string
	}{
		{"single choice", chatResponse{Choices: []chatChoice{{Message: chatMessage{Content: "hi"}}}}, "hi"},
		{"empty choices", chatResponse{Choices: []chatChoice{}}, ""},
		{"empty content", chatResponse{Choices: []chatChoice{{Message: chatMessage{Content: ""}}}}, ""},
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
