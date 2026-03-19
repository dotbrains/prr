package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/dotbrains/prr/internal/agent"
)

// maxConcurrency limits parallel verification calls to avoid rate limits.
const maxConcurrency = 5

// Verifier fact-checks review comments using a secondary AI call.
type Verifier struct {
	agent agent.Agent
}

// NewVerifier creates a Verifier backed by the given agent.
func NewVerifier(a agent.Agent) *Verifier {
	return &Verifier{agent: a}
}

// Verify checks a single review comment against its file diff and optional full source.
func (v *Verifier) Verify(ctx context.Context, comment agent.ReviewComment, fileDiff, fileContent string) *agent.VerificationResult {
	systemPrompt := BuildVerifySystemPrompt()
	userPrompt := BuildVerifyUserPrompt(comment, fileDiff, fileContent)

	text, err := v.agent.Generate(ctx, systemPrompt, userPrompt)
	if err != nil {
		return &agent.VerificationResult{
			Verdict: "uncertain",
			Reason:  fmt.Sprintf("verification call failed: %v", err),
		}
	}

	result, err := parseVerificationJSON(text)
	if err != nil {
		return &agent.VerificationResult{
			Verdict: "uncertain",
			Reason:  fmt.Sprintf("could not parse verification response: %v", err),
		}
	}

	return result
}

// VerifyAll checks all comments concurrently, returning them with verification results attached.
// fileDiffs maps file paths to their diff content. fileContents maps file paths to full source.
func (v *Verifier) VerifyAll(ctx context.Context, comments []agent.ReviewComment, fileDiffs, fileContents map[string]string) []agent.ReviewComment {
	verified := make([]agent.ReviewComment, len(comments))
	copy(verified, comments)

	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i := range verified {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fileDiff := fileDiffs[verified[idx].File]
			fileContent := fileContents[verified[idx].File]
			result := v.Verify(ctx, verified[idx], fileDiff, fileContent)
			verified[idx].Verification = result
		}(i)
	}

	wg.Wait()
	return verified
}

// ApplyVerification processes verified comments according to the action policy.
// action is "drop" (default) or "annotate".
// Returns the processed comments and stats.
func ApplyVerification(comments []agent.ReviewComment, action string) ([]agent.ReviewComment, VerifyStats) {
	var stats VerifyStats
	var result []agent.ReviewComment

	for _, c := range comments {
		if c.Verification == nil {
			result = append(result, c)
			continue
		}

		switch c.Verification.Verdict {
		case "verified":
			stats.Verified++
			result = append(result, c)
		case "inaccurate":
			stats.Inaccurate++
			if action == "drop" {
				stats.Dropped++
				continue
			}
			result = append(result, c)
		case "uncertain":
			stats.Uncertain++
			result = append(result, c)
		default:
			stats.Uncertain++
			result = append(result, c)
		}
	}

	stats.Total = len(comments)
	return result, stats
}

// VerifyStats holds verification outcome counts.
type VerifyStats struct {
	Total      int
	Verified   int
	Inaccurate int
	Uncertain  int
	Dropped    int
}

// String returns a human-readable summary of verification stats.
func (s VerifyStats) String() string {
	parts := []string{fmt.Sprintf("%d/%d verified", s.Verified, s.Total)}
	if s.Uncertain > 0 {
		parts = append(parts, fmt.Sprintf("%d uncertain", s.Uncertain))
	}
	if s.Inaccurate > 0 {
		parts = append(parts, fmt.Sprintf("%d inaccurate", s.Inaccurate))
	}
	if s.Dropped > 0 {
		parts = append(parts, fmt.Sprintf("%d dropped", s.Dropped))
	}
	return strings.Join(parts, ", ")
}

// FileDiffsFromInput builds a file path → diff content map from ReviewInput files.
func FileDiffsFromInput(files []agent.FileDiff) map[string]string {
	m := make(map[string]string, len(files))
	for _, f := range files {
		m[f.Path] = f.Diff
	}
	return m
}

// parseVerificationJSON parses the AI's verification response.
func parseVerificationJSON(text string) (*agent.VerificationResult, error) {
	text = strings.TrimSpace(text)

	// Try direct parse.
	var result agent.VerificationResult
	if err := json.Unmarshal([]byte(text), &result); err == nil {
		if isValidVerdict(result.Verdict) {
			return &result, nil
		}
	}

	// Strip markdown code fences if present.
	if idx := strings.Index(text, "```"); idx != -1 {
		inner := text[idx:]
		lines := strings.Split(inner, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
		}
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.TrimSpace(lines[i]) == "```" {
				lines = lines[:i]
				break
			}
		}
		stripped := strings.TrimSpace(strings.Join(lines, "\n"))
		if err := json.Unmarshal([]byte(stripped), &result); err == nil {
			if isValidVerdict(result.Verdict) {
				return &result, nil
			}
		}
	}

	// Extract JSON object as last resort.
	start := strings.IndexByte(text, '{')
	end := strings.LastIndexByte(text, '}')
	if start != -1 && end > start {
		candidate := text[start : end+1]
		if err := json.Unmarshal([]byte(candidate), &result); err == nil {
			if isValidVerdict(result.Verdict) {
				return &result, nil
			}
		}
	}

	return nil, fmt.Errorf("could not parse verification JSON from response")
}

func isValidVerdict(v string) bool {
	return v == "verified" || v == "inaccurate" || v == "uncertain"
}
