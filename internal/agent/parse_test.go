package agent

import (
	"testing"
)

func TestParseReviewJSON_ValidJSON(t *testing.T) {
	input := `{"summary":"Good PR","comments":[{"file":"main.go","start_line":1,"end_line":1,"severity":"nit","body":"Rename this"}]}`
	output, err := ParseReviewJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Summary != "Good PR" {
		t.Errorf("expected summary 'Good PR', got %q", output.Summary)
	}
	if len(output.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(output.Comments))
	}
	if output.Comments[0].File != "main.go" {
		t.Errorf("expected file 'main.go', got %q", output.Comments[0].File)
	}
}

func TestParseReviewJSON_WithCodeFences(t *testing.T) {
	input := "```json\n{\"summary\":\"test\",\"comments\":[]}\n```"
	output, err := ParseReviewJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Summary != "test" {
		t.Errorf("expected summary 'test', got %q", output.Summary)
	}
}

func TestParseReviewJSON_WithWhitespace(t *testing.T) {
	input := "  \n  {\"summary\":\"test\",\"comments\":[]}  \n  "
	output, err := ParseReviewJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Summary != "test" {
		t.Errorf("expected summary 'test', got %q", output.Summary)
	}
}

func TestParseReviewJSON_InvalidJSON(t *testing.T) {
	_, err := ParseReviewJSON("not json at all")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseReviewJSON_EmptyString(t *testing.T) {
	_, err := ParseReviewJSON("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("hello", 10); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
	if got := Truncate("hello world", 5); got != "hello..." {
		t.Errorf("expected 'hello...', got %q", got)
	}
}
