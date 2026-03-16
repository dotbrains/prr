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

func TestParseReviewJSON_ProseBeforeJSON(t *testing.T) {
	input := "Based on the diff analysis, here's my review:\n\n{\"summary\":\"Looks good\",\"comments\":[]}"
	output, err := ParseReviewJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Summary != "Looks good" {
		t.Errorf("expected summary 'Looks good', got %q", output.Summary)
	}
}

func TestParseReviewJSON_ProseBeforeCodeFence(t *testing.T) {
	input := "Here is the review:\n\n```json\n{\"summary\":\"test\",\"comments\":[]}\n```\n\nLet me know if you need more."
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

func TestParseReviewJSON_TruncatedBetweenComments(t *testing.T) {
	// Simulates token-limit truncation: first comment is complete, second is cut off.
	input := `{"summary":"overview","comments":[{"file":"a.go","start_line":1,"end_line":1,"severity":"nit","body":"fix"},{"file":"b.go","start_li`
	output, err := ParseReviewJSON(input)
	if err != nil {
		t.Fatalf("expected repair to succeed, got: %v", err)
	}
	if output.Summary != "overview" {
		t.Errorf("expected summary 'overview', got %q", output.Summary)
	}
	if len(output.Comments) != 1 {
		t.Errorf("expected 1 salvaged comment, got %d", len(output.Comments))
	}
	if !output.Truncated {
		t.Error("expected Truncated to be true")
	}
}

func TestParseReviewJSON_TruncatedInSummary(t *testing.T) {
	// Truncated inside the summary string — no complete '}' at all.
	input := `{"summary":"this is a very long summ`
	_, err := ParseReviewJSON(input)
	if err == nil {
		t.Fatal("expected error when truncated in summary with no complete object")
	}
}

func TestParseReviewJSON_TruncatedNoCompleteObject(t *testing.T) {
	// Truncated inside the first comment — no '}' exists to anchor repair.
	input := `{"summary":"overview","comments":[{"file":"a.go","start_`
	_, err := ParseReviewJSON(input)
	if err == nil {
		t.Fatal("expected error when no complete object exists to anchor repair")
	}
}

func TestParseReviewJSON_TruncatedWithOneCompleteComment(t *testing.T) {
	// First comment is complete, truncation happens inside the second.
	input := `{"summary":"overview","comments":[{"file":"a.go","start_line":1,"end_line":1,"severity":"nit","body":"ok"}, {"file":"b.go","start_`
	output, err := ParseReviewJSON(input)
	if err != nil {
		t.Fatalf("expected repair to succeed, got: %v", err)
	}
	if output.Summary != "overview" {
		t.Errorf("expected summary 'overview', got %q", output.Summary)
	}
	if len(output.Comments) != 1 {
		t.Errorf("expected 1 salvaged comment, got %d", len(output.Comments))
	}
	if !output.Truncated {
		t.Error("expected Truncated to be true")
	}
}

func TestParseReviewJSON_TruncatedWithBraceInString(t *testing.T) {
	// Ensure a '}' inside a quoted string doesn't fool the repair logic.
	input := `{"summary":"has } brace","comments":[{"file":"a.go","start_line":1,"end_line":1,"severity":"nit","body":"fix {this}"},  {"file":"b`
	output, err := ParseReviewJSON(input)
	if err != nil {
		t.Fatalf("expected repair to succeed, got: %v", err)
	}
	if len(output.Comments) != 1 {
		t.Errorf("expected 1 salvaged comment, got %d", len(output.Comments))
	}
	if output.Comments[0].Body != "fix {this}" {
		t.Errorf("expected body 'fix {this}', got %q", output.Comments[0].Body)
	}
}

func TestParseReviewJSON_TruncatedCodeFence(t *testing.T) {
	// Truncated response wrapped in markdown code fences (no closing fence).
	input := "```json\n" + `{"summary":"ok","comments":[{"file":"x.go","start_line":1,"end_line":1,"severity":"nit","body":"a"},{"file":"y.go","st`
	output, err := ParseReviewJSON(input)
	if err != nil {
		t.Fatalf("expected repair to succeed, got: %v", err)
	}
	if output.Summary != "ok" {
		t.Errorf("expected summary 'ok', got %q", output.Summary)
	}
	if len(output.Comments) != 1 {
		t.Errorf("expected 1 salvaged comment, got %d", len(output.Comments))
	}
	if !output.Truncated {
		t.Error("expected Truncated to be true")
	}
}

func TestRepairTruncatedJSON_NoJSON(t *testing.T) {
	_, ok := repairTruncatedJSON("no json here")
	if ok {
		t.Error("expected false for input with no JSON")
	}
}

func TestRepairTruncatedJSON_AlreadyValid(t *testing.T) {
	input := `{"summary":"ok","comments":[]}`
	repaired, ok := repairTruncatedJSON(input)
	if !ok {
		t.Fatal("expected true for valid JSON")
	}
	if repaired != input {
		t.Errorf("expected unchanged input, got %q", repaired)
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
