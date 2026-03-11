package cmd

import (
	"testing"

	"github.com/dotbrains/prr/internal/agent"
)

func TestFilterComments_NoFilters(t *testing.T) {
	comments := []agent.ReviewComment{
		{Severity: "critical", Body: "bug"},
		{Severity: "praise", Body: "nice"},
	}
	got := filterComments(comments, false, "")
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
}

func TestFilterComments_NoPraise(t *testing.T) {
	comments := []agent.ReviewComment{
		{Severity: "critical", Body: "bug"},
		{Severity: "praise", Body: "nice"},
		{Severity: "suggestion", Body: "refactor"},
	}
	got := filterComments(comments, true, "")
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
	for _, c := range got {
		if c.Severity == "praise" {
			t.Error("praise should be filtered")
		}
	}
}

func TestFilterComments_MinSeverity(t *testing.T) {
	comments := []agent.ReviewComment{
		{Severity: "critical", Body: "a"},
		{Severity: "suggestion", Body: "b"},
		{Severity: "nit", Body: "c"},
		{Severity: "praise", Body: "d"},
	}

	// min=suggestion filters out nit and praise
	got := filterComments(comments, false, "suggestion")
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}

	// min=critical filters out everything except critical
	got = filterComments(comments, false, "critical")
	if len(got) != 1 {
		t.Errorf("expected 1, got %d", len(got))
	}
}

func TestFilterComments_NoPraiseAndMinSeverity(t *testing.T) {
	comments := []agent.ReviewComment{
		{Severity: "critical", Body: "a"},
		{Severity: "nit", Body: "b"},
		{Severity: "praise", Body: "c"},
	}
	// min=nit + no-praise: keeps critical and nit
	got := filterComments(comments, true, "nit")
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
}

func TestFilterComments_UnknownSeverity(t *testing.T) {
	comments := []agent.ReviewComment{
		{Severity: "critical", Body: "a"},
	}
	// Unknown min severity should not filter anything.
	got := filterComments(comments, false, "unknown")
	if len(got) != 1 {
		t.Errorf("expected 1, got %d", len(got))
	}
}

func TestFormatStats(t *testing.T) {
	tests := []struct {
		name  string
		stats map[string]int
		want  string
	}{
		{"empty", map[string]int{}, "no comments"},
		{"single critical", map[string]int{"critical": 1}, "1 critical"},
		{"plural criticals", map[string]int{"critical": 3}, "3 criticals"},
		{"single nit", map[string]int{"nit": 1}, "1 nit"},
		{"plural nits", map[string]int{"nit": 2}, "2 nits"},
		{"mixed", map[string]int{"critical": 1, "suggestion": 2}, "1 critical, 2 suggestions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatStats(tt.stats)
			if got != tt.want {
				t.Errorf("formatStats() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPathToSafeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"main.go", "main-go.md"},
		{"src/internal/handler.go", "src-internal-handler-go.md"},
		{"a/b.c.d", "a-b-c-d.md"},
	}
	for _, tt := range tests {
		got := pathToSafeName(tt.input)
		if got != tt.want {
			t.Errorf("pathToSafeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestReplaceAll(t *testing.T) {
	tests := []struct {
		s, old, new, want string
	}{
		{"hello world", "world", "go", "hello go"},
		{"aaa", "a", "b", "bbb"},
		{"no match", "x", "y", "no match"},
		{"", "a", "b", ""},
		{"abc", "abc", "xyz", "xyz"},
	}
	for _, tt := range tests {
		got := replaceAll(tt.s, tt.old, tt.new)
		if got != tt.want {
			t.Errorf("replaceAll(%q, %q, %q) = %q, want %q", tt.s, tt.old, tt.new, got, tt.want)
		}
	}
}
