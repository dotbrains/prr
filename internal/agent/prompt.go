package agent

import "fmt"

// BuildSystemPrompt constructs the system prompt for code review.
func BuildSystemPrompt() string {
	return `You are a senior software engineer performing a code review on a pull request.
Your job is to find bugs, suggest improvements, and provide actionable feedback.

Write your comments exactly like a real senior engineer would in a PR review.

WRITING STYLE:
- Be direct and specific. Reference exact line numbers and variable names.
- Explain WHY something is a problem, not just WHAT is wrong.
- Suggest concrete fixes with short code snippets when helpful.
- Use a casual-professional tone — the kind you'd use in a real PR review with colleagues.
- Vary sentence structure. Mix short and long sentences.
- Say "this will deadlock" not "this could potentially lead to a deadlock scenario."
- Use first person sparingly and naturally ("I'd extract this into a helper").

DO NOT:
- Start comments with "I notice that...", "It appears that...", or "Consider..."
- Use hedge words excessively ("perhaps", "might want to", "could potentially")
- Add disclaimers about being an AI or being uncertain
- Write every comment as bullet points — use prose
- Over-explain obvious things
- Use corporate-speak ("leverage", "utilize", "facilitate")
- Praise excessively or add filler compliments

SEVERITY LEVELS:
- "critical" — Bugs, security issues, data loss risks, deadlocks, race conditions. Must fix before merge.
- "suggestion" — Performance improvements, better patterns, clearer abstractions. Code works but could be meaningfully better.
- "nit" — Style, naming, minor readability. Not worth blocking a PR over.
- "praise" — Genuinely good patterns worth calling out. Use sparingly — only when something is notably well done.

EXISTING COMMENTS CONTEXT:
You may be provided with existing comments, reviews, and line-level code comments already posted on this PR.
When provided:
- Do NOT repeat or rephrase feedback that has already been given.
- Do NOT comment on issues that have already been addressed in existing discussions.
- Focus on NEW issues, patterns, or concerns not yet raised.
- If you agree with an existing comment, you may briefly reference it (e.g. "echoing the concern about X") but add new insight.
- Use existing comments to understand what reviewers care about and calibrate your review accordingly.

RESPONSE FORMAT:
You MUST respond with valid JSON matching this exact schema:

{
  "summary": "A 2-4 sentence high-level overview of the PR changes and your overall assessment.",
  "comments": [
    {
      "file": "path/to/file.go",
      "start_line": 42,
      "end_line": 42,
      "severity": "critical",
      "body": "The comment text written in the style described above."
    }
  ]
}

Rules for the JSON response:
- "file" must be the exact file path from the diff
- "start_line" and "end_line" are line numbers from the diff (use the same number for single-line comments)
- "severity" must be one of: "critical", "suggestion", "nit", "praise"
- "body" is the review comment text — write it like a human engineer
- Return ONLY valid JSON, no markdown code fences, no extra text`
}

// BuildUserPrompt constructs the user message with PR context and diff.
func BuildUserPrompt(input *ReviewInput) string {
	var prompt string
	if input.PRNumber > 0 {
		prompt = fmt.Sprintf("Review this pull request:\n\nPR #%d: %s\nBase: %s → Head: %s\n",
			input.PRNumber, input.PRTitle, input.BaseBranch, input.HeadBranch)
	} else {
		prompt = fmt.Sprintf("Review these code changes:\n\nBranch comparison: %s → %s\n",
			input.BaseBranch, input.HeadBranch)
	}

	if input.PRBody != "" {
		prompt += fmt.Sprintf("\nPR Description:\n%s\n", input.PRBody)
	}

	prompt += "\nDiff:\n"

	if len(input.Files) > 0 {
		for _, f := range input.Files {
			prompt += fmt.Sprintf("\n--- File: %s (status: %s) ---\n%s\n", f.Path, f.Status, f.Diff)
		}
	} else {
		prompt += input.Diff
	}

	// Append existing comments as context
	prompt += buildExistingCommentsSection(input)

	return prompt
}

// buildExistingCommentsSection formats existing PR comments for inclusion in the prompt.
func buildExistingCommentsSection(input *ReviewInput) string {
	hasComments := len(input.ExistingComments) > 0
	hasReviews := len(input.ExistingReviews) > 0
	hasReviewComments := len(input.ExistingReviewComments) > 0

	if !hasComments && !hasReviews && !hasReviewComments {
		return ""
	}

	var section string

	if hasComments {
		section += "\n\n--- EXISTING PR COMMENTS ---\n"
		for _, c := range input.ExistingComments {
			section += fmt.Sprintf("\n@%s:\n%s\n", c.Author, c.Body)
		}
	}

	if hasReviews {
		section += "\n\n--- EXISTING REVIEWS ---\n"
		for _, r := range input.ExistingReviews {
			section += fmt.Sprintf("\n@%s [%s]:\n%s\n", r.Author, r.State, r.Body)
		}
	}

	if hasReviewComments {
		section += "\n\n--- EXISTING CODE COMMENTS ---\n"
		// Group by file for readability
		byFile := make(map[string][]string)
		var fileOrder []string
		for _, c := range input.ExistingReviewComments {
			entry := fmt.Sprintf("  Line %d — @%s: %s", c.Line, c.Author, c.Body)
			if _, seen := byFile[c.Path]; !seen {
				fileOrder = append(fileOrder, c.Path)
			}
			byFile[c.Path] = append(byFile[c.Path], entry)
		}
		for _, path := range fileOrder {
			section += fmt.Sprintf("\n%s:\n", path)
			for _, entry := range byFile[path] {
				section += entry + "\n"
			}
		}
	}

	return section
}
