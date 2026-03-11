package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseReviewJSON parses AI response text into a ReviewOutput.
// It handles markdown code fences that models sometimes wrap around JSON.
func ParseReviewJSON(text string) (*ReviewOutput, error) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		// Remove first and last lines (the fences)
		if len(lines) >= 3 {
			lines = lines[1 : len(lines)-1]
		}
		text = strings.Join(lines, "\n")
	}
	text = strings.TrimSpace(text)

	var result ReviewOutput
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("parsing AI response as JSON: %w\n\nRaw response:\n%s", err, Truncate(text, 500))
	}
	return &result, nil
}

// Truncate shortens a string to maxLen characters, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
