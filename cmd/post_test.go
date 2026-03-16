package cmd

import (
	"testing"

	"github.com/dotbrains/prr/internal/agent"
	"github.com/dotbrains/prr/internal/writer"
)

func TestBuildReviewPayload_AutoEvent(t *testing.T) {
	meta := &writer.ReviewMetadata{
		PRNumber:  42,
		HeadSHA:   "abc123",
		AgentName: "claude",
		Summary:   "Test",
		Comments: []agent.ReviewComment{
			{File: "main.go", StartLine: 10, EndLine: 10, Severity: "critical", Body: "bug"},
			{File: "main.go", StartLine: 20, EndLine: 20, Severity: "suggestion", Body: "refactor"},
			{File: "main.go", StartLine: 30, EndLine: 30, Severity: "praise", Body: "nice"},
		},
	}

	payload := buildReviewPayload(meta, "")

	// Should auto-detect REQUEST_CHANGES due to critical.
	if payload.Event != "REQUEST_CHANGES" {
		t.Errorf("expected REQUEST_CHANGES, got %q", payload.Event)
	}

	// Praise should be filtered out.
	if len(payload.Comments) != 2 {
		t.Errorf("expected 2 comments (praise filtered), got %d", len(payload.Comments))
	}

	if payload.CommitID != "abc123" {
		t.Errorf("expected commit_id abc123, got %q", payload.CommitID)
	}
}

func TestBuildReviewPayload_NoCritical(t *testing.T) {
	meta := &writer.ReviewMetadata{
		PRNumber:  10,
		AgentName: "gpt",
		Summary:   "Clean",
		Comments: []agent.ReviewComment{
			{File: "main.go", StartLine: 5, EndLine: 5, Severity: "nit", Body: "rename"},
		},
	}

	payload := buildReviewPayload(meta, "")
	if payload.Event != "COMMENT" {
		t.Errorf("expected COMMENT, got %q", payload.Event)
	}
}

func TestBuildReviewPayload_EventOverride(t *testing.T) {
	meta := &writer.ReviewMetadata{
		PRNumber:  10,
		AgentName: "claude",
		Summary:   "ok",
		Comments: []agent.ReviewComment{
			{File: "main.go", StartLine: 5, EndLine: 5, Severity: "critical", Body: "bug"},
		},
	}

	payload := buildReviewPayload(meta, "comment")
	if payload.Event != "COMMENT" {
		t.Errorf("expected COMMENT override, got %q", payload.Event)
	}
}

func TestBuildReviewPayload_MultiLineComment(t *testing.T) {
	meta := &writer.ReviewMetadata{
		PRNumber:  10,
		AgentName: "claude",
		Summary:   "ok",
		Comments: []agent.ReviewComment{
			{File: "main.go", StartLine: 10, EndLine: 20, Severity: "suggestion", Body: "refactor this block"},
		},
	}

	payload := buildReviewPayload(meta, "")
	if len(payload.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(payload.Comments))
	}

	c := payload.Comments[0]
	if c.StartLine != 10 {
		t.Errorf("start_line = %d, want 10", c.StartLine)
	}
	if c.Line != 20 {
		t.Errorf("line = %d, want 20", c.Line)
	}
	if c.StartSide != "RIGHT" {
		t.Errorf("start_side = %q, want RIGHT", c.StartSide)
	}
}
