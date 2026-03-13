package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseReviewJSON parses AI response text into a ReviewOutput.
// It handles markdown code fences and prose surrounding the JSON that
// models sometimes include in their response.
func ParseReviewJSON(text string) (*ReviewOutput, error) {
	text = strings.TrimSpace(text)

	// Try direct parse first (fastest path).
	var result ReviewOutput
	if err := json.Unmarshal([]byte(text), &result); err == nil {
		return &result, nil
	}

	// Strip markdown code fences if present.
	if idx := strings.Index(text, "```"); idx != -1 {
		inner := text[idx:]
		lines := strings.Split(inner, "\n")
		// Drop opening fence line.
		if len(lines) >= 2 {
			lines = lines[1:]
		}
		// Drop closing fence if present.
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.TrimSpace(lines[i]) == "```" {
				lines = lines[:i]
				break
			}
		}
		stripped := strings.TrimSpace(strings.Join(lines, "\n"))
		if err := json.Unmarshal([]byte(stripped), &result); err == nil {
			return &result, nil
		}
	}

	// Last resort: extract the outermost { ... } JSON object.
	if jsonStr, ok := extractJSONObject(text); ok {
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			return &result, nil
		}
	}

	return nil, fmt.Errorf("parsing AI response as JSON: could not find valid JSON in response\n\nRaw response:\n%s", Truncate(text, 500))
}

// extractJSONObject finds the outermost balanced { ... } substring in s.
func extractJSONObject(s string) (string, bool) {
	start := strings.IndexByte(s, '{')
	if start == -1 {
		return "", false
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1], true
			}
		}
	}
	return "", false
}

// Truncate shortens a string to maxLen characters, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
