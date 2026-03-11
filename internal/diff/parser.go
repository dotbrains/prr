package diff

import (
	"strings"

	"github.com/dotbrains/prr/internal/agent"
)

// Parse splits a unified diff into per-file FileDiff structs.
func Parse(raw string) []agent.FileDiff {
	var files []agent.FileDiff
	var current *agent.FileDiff
	var currentLines []string

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			// Flush previous file
			if current != nil {
				current.Diff = strings.Join(currentLines, "\n")
				files = append(files, *current)
			}
			path := parseDiffPath(line)
			current = &agent.FileDiff{
				Path:   path,
				Status: "modified",
			}
			currentLines = []string{line}
			continue
		}

		if current == nil {
			continue
		}

		// Detect file status from diff headers
		if strings.HasPrefix(line, "new file mode") {
			current.Status = "added"
		} else if strings.HasPrefix(line, "deleted file mode") {
			current.Status = "deleted"
		} else if strings.HasPrefix(line, "rename from") || strings.HasPrefix(line, "rename to") {
			current.Status = "renamed"
		}

		currentLines = append(currentLines, line)
	}

	// Flush last file
	if current != nil {
		current.Diff = strings.Join(currentLines, "\n")
		files = append(files, *current)
	}

	return files
}

// parseDiffPath extracts the file path from a "diff --git a/path b/path" line.
func parseDiffPath(line string) string {
	// Format: "diff --git a/path/to/file b/path/to/file"
	parts := strings.SplitN(line, " b/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	// Fallback: try to extract from a/ prefix
	parts = strings.SplitN(line, " a/", 2)
	if len(parts) == 2 {
		// Take everything up to the next space
		rest := parts[1]
		if idx := strings.Index(rest, " "); idx != -1 {
			return rest[:idx]
		}
		return rest
	}
	return line
}

// LineCount returns the total number of lines in a diff string.
func LineCount(diff string) int {
	if diff == "" {
		return 0
	}
	return strings.Count(diff, "\n") + 1
}
