package context

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/dotbrains/prr/internal/agent"
)

// FileReader abstracts file listing and reading so that CollectContext
// can work with both a local git repo and the GitHub API.
type FileReader interface {
	ListFiles(ctx context.Context, ref, dir string) ([]string, error)
	ReadFile(ctx context.Context, ref, path string) (string, error)
}

// skipExtensions are file extensions that are unlikely to contain useful patterns.
var skipExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true,
	".svg": true, ".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
	".pdf": true, ".zip": true, ".tar": true, ".gz": true, ".bin": true,
	".lock": true, ".sum": true, ".mod": true,
}

// skipDirs are directory prefixes to skip entirely.
var skipDirs = []string{
	"vendor/", "node_modules/", ".git/", "dist/", "build/",
}

// CollectContext gathers sibling files from the base branch for each changed file.
// It returns CodebaseFile entries capped at maxLines total lines.
func CollectContext(ctx context.Context, reader FileReader, baseRef string, changedFiles []agent.FileDiff, maxLines int) []agent.CodebaseFile {
	if maxLines <= 0 {
		return nil
	}

	// Collect unique directories from changed files.
	dirs := uniqueDirs(changedFiles)

	// Track which files are in the diff so we don't include them as context.
	changedSet := make(map[string]bool, len(changedFiles))
	for _, f := range changedFiles {
		changedSet[f.Path] = true
	}

	// Determine if any changed file is a test file.
	hasTestFile := false
	for _, f := range changedFiles {
		if isTestFile(f.Path) {
			hasTestFile = true
			break
		}
	}

	var result []agent.CodebaseFile
	totalLines := 0

	for _, dir := range dirs {
		siblings, err := reader.ListFiles(ctx, baseRef, dir)
		if err != nil {
			continue // non-fatal
		}

		for _, sibling := range siblings {
			if totalLines >= maxLines {
				return result
			}

			// Skip files that are in the diff.
			if changedSet[sibling] {
				continue
			}

			// Skip binary/non-code extensions.
			ext := strings.ToLower(filepath.Ext(sibling))
			if skipExtensions[ext] {
				continue
			}

			// Skip vendored paths.
			if shouldSkipPath(sibling) {
				continue
			}

			// Skip test files unless the PR itself modifies tests.
			if !hasTestFile && isTestFile(sibling) {
				continue
			}

			content, err := reader.ReadFile(ctx, baseRef, sibling)
			if err != nil {
				continue // non-fatal
			}

			lines := strings.Count(content, "\n") + 1
			if totalLines+lines > maxLines {
				// Include a truncated version if it fits partially.
				remaining := maxLines - totalLines
				if remaining > 10 {
					truncated := truncateToLines(content, remaining)
					result = append(result, agent.CodebaseFile{
						Path:    sibling,
						Content: truncated,
					})
				}
				return result
			}

			result = append(result, agent.CodebaseFile{
				Path:    sibling,
				Content: content,
			})
			totalLines += lines
		}
	}

	return result
}

// uniqueDirs returns deduplicated directory paths from the changed files.
func uniqueDirs(files []agent.FileDiff) []string {
	seen := make(map[string]bool)
	var dirs []string
	for _, f := range files {
		dir := filepath.Dir(f.Path)
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

func isTestFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".test.js") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.HasSuffix(base, ".spec.js") ||
		strings.HasPrefix(base, "test_") ||
		strings.Contains(path, "__tests__/")
}

func shouldSkipPath(path string) bool {
	for _, dir := range skipDirs {
		if strings.HasPrefix(path, dir) {
			return true
		}
	}
	return false
}

func truncateToLines(content string, maxLines int) string {
	lines := strings.SplitN(content, "\n", maxLines+1)
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n") + "\n// ... truncated"
}
