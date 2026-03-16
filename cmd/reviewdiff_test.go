package cmd

import (
	"testing"

	"github.com/dotbrains/prr/internal/agent"
)

func TestIndexComments(t *testing.T) {
	comments := []agent.ReviewComment{
		{File: "a.go", StartLine: 10, Severity: "critical", Body: "bug here"},
		{File: "b.go", StartLine: 20, Severity: "nit", Body: "rename this"},
	}

	idx := indexComments(comments)
	if len(idx) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(idx))
	}

	key := commentKey{File: "a.go", StartLine: 10, Severity: "critical"}
	c, ok := idx[key]
	if !ok {
		t.Fatal("expected to find comment for a.go:10:critical")
	}
	if c.Body != "bug here" {
		t.Errorf("expected body 'bug here', got %q", c.Body)
	}
}

func TestIndexComments_Empty(t *testing.T) {
	idx := indexComments(nil)
	if len(idx) != 0 {
		t.Errorf("expected 0 entries, got %d", len(idx))
	}
}

func TestTruncateBody(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly ten", 11, "exactly ten"},
		{"this is a long comment that should be truncated", 20, "this is a long comme..."},
		{"line1\nline2", 20, "line1 line2"},
	}

	for _, tt := range tests {
		got := truncateBody(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateBody(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}
