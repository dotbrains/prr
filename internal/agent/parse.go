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

	// Final fallback: attempt to repair truncated JSON (e.g. from token limit).
	if repaired, ok := repairTruncatedJSON(text); ok {
		if err := json.Unmarshal([]byte(repaired), &result); err == nil {
			result.Truncated = true
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

// repairTruncatedJSON attempts to salvage a truncated JSON response by
// finding the last safely-closed '}' (outside any quoted string) and
// appending whatever closing delimiters are still needed.
func repairTruncatedJSON(s string) (string, bool) {
	start := strings.IndexByte(s, '{')
	if start == -1 {
		return "", false
	}
	text := s[start:]

	// Walk the JSON tracking a stack of open delimiters and snapshot
	// the state every time we close a '}' outside a quoted string.
	type snapshot struct {
		end   int    // exclusive index after the '}'
		stack []byte // copy of open-delimiter stack at that point
	}

	var best snapshot
	var stack []byte
	inStr := false
	esc := false

	for i := 0; i < len(text); i++ {
		c := text[i]
		if esc {
			esc = false
			continue
		}
		if inStr {
			switch c {
			case '\\':
				esc = true
			case '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{':
			stack = append(stack, '}')
		case '[':
			stack = append(stack, ']')
		case '}':
			if len(stack) > 0 && stack[len(stack)-1] == '}' {
				stack = stack[:len(stack)-1]
			}
			cp := make([]byte, len(stack))
			copy(cp, stack)
			best = snapshot{end: i + 1, stack: cp}
		case ']':
			if len(stack) > 0 && stack[len(stack)-1] == ']' {
				stack = stack[:len(stack)-1]
			}
		}
	}

	// If nothing is left open the JSON was already balanced — the caller
	// would have parsed it directly, so this shouldn't normally happen.
	if len(stack) == 0 && !inStr {
		return text, true
	}

	// If we never saw a complete '}', there's nothing to salvage.
	if best.end == 0 {
		return "", false
	}

	// Truncate to the last safe '}' and strip any trailing comma.
	repaired := strings.TrimRight(text[:best.end], " \t\n\r,")

	// Close remaining open delimiters in reverse (innermost first).
	for i := len(best.stack) - 1; i >= 0; i-- {
		repaired += string(best.stack[i])
	}

	return repaired, true
}

// Truncate shortens a string to maxLen characters, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
