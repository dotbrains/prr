package verify

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/dotbrains/prr/internal/agent"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	response string
	err      error
	calls    atomic.Int32
}

func (m *mockAgent) Name() string { return "mock" }

func (m *mockAgent) Review(_ context.Context, _ *agent.ReviewInput) (*agent.ReviewOutput, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockAgent) Generate(_ context.Context, _, _ string) (string, error) {
	m.calls.Add(1)
	return m.response, m.err
}

func TestVerify_Verified(t *testing.T) {
	mock := &mockAgent{response: `{"verdict": "verified", "reason": "Line numbers and claims match the diff."}`}
	v := NewVerifier(mock)

	comment := agent.ReviewComment{
		File: "main.go", StartLine: 10, EndLine: 10, Severity: "critical", Body: "Bug here.",
	}

	result := v.Verify(context.Background(), comment, "some diff", "package main\nfunc main() {}")

	if result.Verdict != "verified" {
		t.Errorf("expected verdict 'verified', got %q", result.Verdict)
	}
	if result.Reason == "" {
		t.Error("expected non-empty reason")
	}
}

func TestVerify_Inaccurate(t *testing.T) {
	mock := &mockAgent{response: `{"verdict": "inaccurate", "reason": "The variable mentioned does not exist."}`}
	v := NewVerifier(mock)

	comment := agent.ReviewComment{
		File: "main.go", StartLine: 5, EndLine: 5, Severity: "nit", Body: "Rename foo.",
	}

	result := v.Verify(context.Background(), comment, "diff", "")
	if result.Verdict != "inaccurate" {
		t.Errorf("expected 'inaccurate', got %q", result.Verdict)
	}
}

func TestVerify_AgentError(t *testing.T) {
	mock := &mockAgent{err: fmt.Errorf("rate limited")}
	v := NewVerifier(mock)

	comment := agent.ReviewComment{
		File: "main.go", StartLine: 1, EndLine: 1, Severity: "nit", Body: "test",
	}

	result := v.Verify(context.Background(), comment, "diff", "")
	if result.Verdict != "uncertain" {
		t.Errorf("expected 'uncertain' on error, got %q", result.Verdict)
	}
}

func TestVerify_MalformedJSON(t *testing.T) {
	mock := &mockAgent{response: "this is not json"}
	v := NewVerifier(mock)

	comment := agent.ReviewComment{
		File: "main.go", StartLine: 1, EndLine: 1, Severity: "nit", Body: "test",
	}

	result := v.Verify(context.Background(), comment, "diff", "")
	if result.Verdict != "uncertain" {
		t.Errorf("expected 'uncertain' on parse failure, got %q", result.Verdict)
	}
}

func TestVerifyAll_Concurrent(t *testing.T) {
	mock := &mockAgent{response: `{"verdict": "verified", "reason": "ok"}`}
	v := NewVerifier(mock)

	comments := make([]agent.ReviewComment, 10)
	for i := range comments {
		comments[i] = agent.ReviewComment{
			File: fmt.Sprintf("file%d.go", i), StartLine: i + 1, EndLine: i + 1,
			Severity: "nit", Body: fmt.Sprintf("comment %d", i),
		}
	}

	diffs := map[string]string{}
	for i := range comments {
		diffs[comments[i].File] = "some diff"
	}

	contents := map[string]string{}
	result := v.VerifyAll(context.Background(), comments, diffs, contents)

	if len(result) != 10 {
		t.Fatalf("expected 10 comments, got %d", len(result))
	}
	for i, c := range result {
		if c.Verification == nil {
			t.Errorf("comment %d has nil verification", i)
		} else if c.Verification.Verdict != "verified" {
			t.Errorf("comment %d: expected 'verified', got %q", i, c.Verification.Verdict)
		}
	}

	if int(mock.calls.Load()) != 10 {
		t.Errorf("expected 10 agent calls, got %d", mock.calls.Load())
	}
}

func TestApplyVerification_Annotate(t *testing.T) {
	comments := []agent.ReviewComment{
		{File: "a.go", Severity: "critical", Verification: &agent.VerificationResult{Verdict: "verified"}},
		{File: "b.go", Severity: "nit", Verification: &agent.VerificationResult{Verdict: "inaccurate", Reason: "wrong"}},
		{File: "c.go", Severity: "suggestion", Verification: &agent.VerificationResult{Verdict: "uncertain"}},
	}

	result, stats := ApplyVerification(comments, "annotate")

	if len(result) != 3 {
		t.Fatalf("annotate should keep all comments, got %d", len(result))
	}
	if stats.Verified != 1 || stats.Inaccurate != 1 || stats.Uncertain != 1 {
		t.Errorf("unexpected stats: %+v", stats)
	}
	if stats.Dropped != 0 {
		t.Error("annotate should not drop comments")
	}
}

func TestApplyVerification_Drop(t *testing.T) {
	comments := []agent.ReviewComment{
		{File: "a.go", Severity: "critical", Verification: &agent.VerificationResult{Verdict: "verified"}},
		{File: "b.go", Severity: "nit", Verification: &agent.VerificationResult{Verdict: "inaccurate", Reason: "wrong"}},
		{File: "c.go", Severity: "suggestion", Verification: &agent.VerificationResult{Verdict: "uncertain"}},
	}

	result, stats := ApplyVerification(comments, "drop")

	if len(result) != 2 {
		t.Fatalf("drop should remove inaccurate comments, got %d", len(result))
	}
	if stats.Dropped != 1 {
		t.Errorf("expected 1 dropped, got %d", stats.Dropped)
	}
}

func TestApplyVerification_NilVerification(t *testing.T) {
	comments := []agent.ReviewComment{
		{File: "a.go", Severity: "nit"},
	}

	result, stats := ApplyVerification(comments, "annotate")

	if len(result) != 1 {
		t.Fatal("should pass through comments without verification")
	}
	if stats.Total != 1 {
		t.Errorf("total should be 1, got %d", stats.Total)
	}
}

func TestParseVerificationJSON_Direct(t *testing.T) {
	r, err := parseVerificationJSON(`{"verdict": "verified", "reason": "ok"}`)
	if err != nil {
		t.Fatal(err)
	}
	if r.Verdict != "verified" {
		t.Errorf("expected 'verified', got %q", r.Verdict)
	}
}

func TestParseVerificationJSON_CodeFence(t *testing.T) {
	input := "Here is my analysis:\n```json\n{\"verdict\": \"inaccurate\", \"reason\": \"wrong var\"}\n```"
	r, err := parseVerificationJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	if r.Verdict != "inaccurate" {
		t.Errorf("expected 'inaccurate', got %q", r.Verdict)
	}
}

func TestParseVerificationJSON_ExtraText(t *testing.T) {
	input := "After analysis: {\"verdict\": \"uncertain\", \"reason\": \"not enough context\"} that's my take."
	r, err := parseVerificationJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	if r.Verdict != "uncertain" {
		t.Errorf("expected 'uncertain', got %q", r.Verdict)
	}
}

func TestParseVerificationJSON_Invalid(t *testing.T) {
	_, err := parseVerificationJSON("no json here")
	if err == nil {
		t.Error("expected error for non-JSON input")
	}
}

func TestFileDiffsFromInput(t *testing.T) {
	files := []agent.FileDiff{
		{Path: "a.go", Diff: "diff a"},
		{Path: "b.go", Diff: "diff b"},
	}

	m := FileDiffsFromInput(files)

	if m["a.go"] != "diff a" {
		t.Errorf("expected 'diff a', got %q", m["a.go"])
	}
	if m["b.go"] != "diff b" {
		t.Errorf("expected 'diff b', got %q", m["b.go"])
	}
}

func TestVerifyStats_String(t *testing.T) {
	s := VerifyStats{Total: 10, Verified: 7, Uncertain: 2, Inaccurate: 1, Dropped: 0}
	str := s.String()
	if str != "7/10 verified, 2 uncertain, 1 inaccurate" {
		t.Errorf("unexpected stats string: %q", str)
	}
}

func TestVerifyStats_String_WithDropped(t *testing.T) {
	s := VerifyStats{Total: 5, Verified: 3, Uncertain: 0, Inaccurate: 2, Dropped: 2}
	str := s.String()
	if str != "3/5 verified, 2 inaccurate, 2 dropped" {
		t.Errorf("unexpected stats string: %q", str)
	}
}
