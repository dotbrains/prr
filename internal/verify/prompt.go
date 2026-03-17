package verify

import (
	"fmt"
	"strings"

	"github.com/dotbrains/prr/internal/agent"
)

// BuildVerifySystemPrompt returns the system prompt for comment verification.
func BuildVerifySystemPrompt() string {
	return `You are a code review fact-checker. Your job is to verify whether a review comment is accurate given the actual code diff.

You will be given:
1. A review comment (file, line numbers, severity, body)
2. The relevant file diff showing the actual code changes

Your task is to check:
- Do the referenced line numbers exist in the diff?
- Are variable names, function names, and identifiers mentioned in the comment actually present in the referenced code?
- Is the behavioral claim (e.g. "this will deadlock", "nil pointer", "race condition") accurate given the code?
- If a fix is suggested, is it syntactically and logically valid?

IMPORTANT:
- Be strict. If the comment claims something specific about the code, verify it against the actual diff.
- If line numbers are off by a small amount but the comment clearly refers to the right code, verdict is "verified".
- If you cannot determine accuracy because the diff doesn't show enough context, verdict is "uncertain".
- Only mark "inaccurate" when the comment makes a factually wrong claim about the code.

RESPONSE FORMAT:
Respond with valid JSON only:

{
  "verdict": "verified|inaccurate|uncertain",
  "reason": "Brief explanation of your assessment."
}

Rules:
- "verdict" must be exactly one of: "verified", "inaccurate", "uncertain"
- "reason" should be 1-2 sentences max
- Return ONLY valid JSON, no markdown code fences, no extra text`
}

// BuildVerifyUserPrompt constructs the user message for verifying a single comment.
func BuildVerifyUserPrompt(comment agent.ReviewComment, fileDiff string) string {
	var sb strings.Builder

	sb.WriteString("Verify this review comment against the code diff.\n\n")

	sb.WriteString("REVIEW COMMENT:\n")
	fmt.Fprintf(&sb, "  File: %s\n", comment.File)
	if comment.StartLine == comment.EndLine || comment.EndLine == 0 {
		fmt.Fprintf(&sb, "  Line: %d\n", comment.StartLine)
	} else {
		fmt.Fprintf(&sb, "  Lines: %d-%d\n", comment.StartLine, comment.EndLine)
	}
	fmt.Fprintf(&sb, "  Severity: %s\n", comment.Severity)
	fmt.Fprintf(&sb, "  Body: %s\n", comment.Body)

	sb.WriteString("\nFILE DIFF:\n")
	if fileDiff != "" {
		sb.WriteString(fileDiff)
	} else {
		sb.WriteString("(no diff available for this file)")
	}

	return sb.String()
}
