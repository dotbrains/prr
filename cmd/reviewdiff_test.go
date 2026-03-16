package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dotbrains/prr/internal/agent"
	"github.com/dotbrains/prr/internal/writer"
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

func TestRunReviewDiff(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	meta1 := writer.ReviewMetadata{
		PRNumber:  42,
		AgentName: "claude",
		Comments: []agent.ReviewComment{
			{File: "a.go", StartLine: 10, EndLine: 10, Severity: "critical", Body: "bug here"},
			{File: "b.go", StartLine: 20, EndLine: 20, Severity: "nit", Body: "rename this"},
		},
	}
	meta2 := writer.ReviewMetadata{
		PRNumber:  42,
		AgentName: "claude",
		Comments: []agent.ReviewComment{
			{File: "a.go", StartLine: 10, EndLine: 10, Severity: "critical", Body: "bug here fixed"},
			{File: "c.go", StartLine: 5, EndLine: 5, Severity: "suggestion", Body: "new issue"},
		},
	}

	writeMeta := func(dir string, meta writer.ReviewMetadata) {
		data, err := json.Marshal(meta)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "metadata.json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeMeta(dir1, meta1)
	writeMeta(dir2, meta2)

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"diff", dir1, dir2})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	// b.go:20 nit was removed
	if !contains(out, "Resolved comments") {
		t.Errorf("expected 'Resolved comments' in output, got: %s", out)
	}
	// c.go:5 suggestion is new
	if !contains(out, "New comments") {
		t.Errorf("expected 'New comments' in output, got: %s", out)
	}
	// a.go:10 critical changed body
	if !contains(out, "Changed comments") {
		t.Errorf("expected 'Changed comments' in output, got: %s", out)
	}
}

func TestRunReviewDiff_NoDifferences(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	meta := writer.ReviewMetadata{
		PRNumber:  42,
		AgentName: "claude",
		Comments: []agent.ReviewComment{
			{File: "a.go", StartLine: 10, EndLine: 10, Severity: "critical", Body: "same"},
		},
	}

	for _, dir := range []string{dir1, dir2} {
		data, _ := json.Marshal(meta)
		if err := os.WriteFile(filepath.Join(dir, "metadata.json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"diff", dir1, dir2})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !contains(buf.String(), "No differences") {
		t.Errorf("expected 'No differences' in output, got: %s", buf.String())
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
