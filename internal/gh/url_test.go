package gh

import (
	"testing"
)

func TestParsePRURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		owner    string
		repo     string
		prNumber int
		wantErr  bool
	}{
		{
			name:     "https URL",
			input:    "https://github.com/dotbrains/prr/pull/42",
			owner:    "dotbrains",
			repo:     "prr",
			prNumber: 42,
		},
		{
			name:     "http URL",
			input:    "http://github.com/dotbrains/prr/pull/7",
			owner:    "dotbrains",
			repo:     "prr",
			prNumber: 7,
		},
		{
			name:     "no scheme",
			input:    "github.com/dotbrains/prr/pull/123",
			owner:    "dotbrains",
			repo:     "prr",
			prNumber: 123,
		},
		{
			name:     "URL with trailing slash",
			input:    "https://github.com/owner/repo/pull/99/",
			owner:    "owner",
			repo:     "repo",
			prNumber: 99,
		},
		{
			name:     "URL with extra path segments",
			input:    "https://github.com/owner/repo/pull/5/files",
			owner:    "owner",
			repo:     "repo",
			prNumber: 5,
		},
		{
			name:    "not github.com",
			input:   "https://gitlab.com/owner/repo/pull/1",
			wantErr: true,
		},
		{
			name:    "missing pull segment",
			input:   "https://github.com/owner/repo/issues/1",
			wantErr: true,
		},
		{
			name:    "missing PR number",
			input:   "https://github.com/owner/repo/pull/",
			wantErr: true,
		},
		{
			name:    "non-numeric PR number",
			input:   "https://github.com/owner/repo/pull/abc",
			wantErr: true,
		},
		{
			name:    "negative PR number",
			input:   "https://github.com/owner/repo/pull/-1",
			wantErr: true,
		},
		{
			name:    "zero PR number",
			input:   "https://github.com/owner/repo/pull/0",
			wantErr: true,
		},
		{
			name:    "too few path segments",
			input:   "https://github.com/owner",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, prNumber, err := ParsePRURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got owner=%q repo=%q pr=%d", owner, repo, prNumber)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if owner != tt.owner {
				t.Errorf("owner = %q, want %q", owner, tt.owner)
			}
			if repo != tt.repo {
				t.Errorf("repo = %q, want %q", repo, tt.repo)
			}
			if prNumber != tt.prNumber {
				t.Errorf("prNumber = %d, want %d", prNumber, tt.prNumber)
			}
		})
	}
}

func TestIsPRURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"https://github.com/owner/repo/pull/1", true},
		{"github.com/owner/repo/pull/42", true},
		{"42", false},
		{"", false},
		{"https://github.com/owner/repo/issues/1", false},
		{"github.com/owner/repo", false},
	}

	for _, tt := range tests {
		got := IsPRURL(tt.input)
		if got != tt.want {
			t.Errorf("IsPRURL(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
