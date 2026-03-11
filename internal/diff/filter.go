package diff

import (
	"path/filepath"

	"github.com/dotbrains/prr/internal/agent"
)

// Filter removes files matching any of the ignore patterns.
// Patterns use filepath.Match syntax (e.g. "*.lock", "vendor/**").
func Filter(files []agent.FileDiff, patterns []string) (kept []agent.FileDiff, filtered int) {
	for _, f := range files {
		if matchesAny(f.Path, patterns) {
			filtered++
			continue
		}
		kept = append(kept, f)
	}
	return kept, filtered
}

// matchesAny returns true if the path matches any of the glob patterns.
func matchesAny(path string, patterns []string) bool {
	for _, pattern := range patterns {
		// Try matching the full path
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		// Try matching just the filename
		base := filepath.Base(path)
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
		// For ** patterns, try matching each path segment
		if containsDoubleStar(pattern) {
			if matchDoubleStar(path, pattern) {
				return true
			}
		}
	}
	return false
}

// containsDoubleStar checks if a pattern uses ** glob syntax.
func containsDoubleStar(pattern string) bool {
	return len(pattern) >= 2 && (pattern[:2] == "**" || pattern[len(pattern)-2:] == "**" || contains(pattern, "/**/") || contains(pattern, "**"))
}

// matchDoubleStar handles ** patterns by checking if the path starts with the prefix.
func matchDoubleStar(path, pattern string) bool {
	// Handle "dir/**" pattern
	if len(pattern) > 3 && pattern[len(pattern)-3:] == "/**" {
		prefix := pattern[:len(pattern)-3]
		if len(path) > len(prefix) && path[:len(prefix)] == prefix && path[len(prefix)] == '/' {
			return true
		}
		return path == prefix
	}
	// Handle "**/*.ext" pattern
	if len(pattern) > 3 && pattern[:3] == "**/" {
		suffix := pattern[3:]
		base := filepath.Base(path)
		if matched, _ := filepath.Match(suffix, base); matched {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
