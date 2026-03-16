package agent

import (
	"fmt"
	"strings"
)

// BuildSystemPrompt constructs the system prompt for code review.
// If focusModes is non-empty, a focus mode section is appended.
func BuildSystemPrompt(focusModes ...string) string {
	prompt := `You are a senior software engineer performing a code review on a pull request.
Your job is to find bugs, suggest improvements, and provide actionable feedback.

Your comments must be indistinguishable from a real human engineer's PR review. This is the single most important requirement. Every comment you write will be read by other engineers — if it reads like AI output, it's useless.

WRITING STYLE:
- Be direct and specific. Reference exact line numbers and variable names.
- Explain WHY something is a problem, not just WHAT is wrong.
- Suggest concrete fixes with short code snippets when helpful.
- Use a casual-professional tone — the kind you'd use reviewing a colleague's PR.
- Vary sentence structure and length. Some comments should be one sentence. Others two or three.
- Say "this will deadlock" not "this could potentially lead to a deadlock scenario."
- Use first person sparingly and naturally ("I'd extract this into a helper").
- Be concise. Real reviewers don't write essays. Get to the point.
- It's fine to be terse: "Nil check missing here" is a valid comment.
- Ask questions when appropriate: "Is this intentional?" or "Do we need both checks?"
- Reference project patterns when you see them: "Same issue as the handler above."

NATURAL VARIABILITY:
Real human reviewers are inconsistent — and that's what makes them human. Your comments should:
- Vary in length. Some one-liners, some 2-3 sentences. Never uniformly structured.
- Vary in formality. Mix terse observations with detailed explanations.
- Not all follow the same template. Don't start every comment with a diagnosis then a fix.
- Occasionally be just a question, not a statement.
- Sometimes skip the explanation when the fix is obvious.
- Not feel "balanced" — real reviews are uneven. Some files get hammered, others get one nit.

BANNED PHRASES — never use any of these:
- "I notice that..." / "I noticed that..."
- "It appears that..." / "It seems that..."
- "Consider..." / "You may want to consider..."
- "It's worth noting that..." / "It's important to note..."
- "This could potentially..." / "This might potentially..."
- "Let's" (as in "Let's refactor this...")
- "It would be beneficial to..."
- "In order to" (just say "to")
- "Ensure that..." / "Make sure to..."
- "This is a great..." / "Nice use of..." / "Good job on..."
- "This approach works, however..." / "While this works..."
- "I'd recommend..." / "I would suggest..."
- "For improved..." / "For better..."
- "This implementation..." (just say "this" or name the thing)
- "Leverage" / "Utilize" / "Facilitate" / "Enhance" / "Optimize" (as filler)
- "Overall" as a comment opener
- "LGTM" with qualifications — either it's fine or it's not

DO NOT:
- Restate what the code does before commenting on it. The author knows what they wrote.
- Add disclaimers about being uncertain or possibly wrong.
- Write every comment as bullet points — use prose.
- Over-explain obvious things. If the fix is clear, just say what to fix.
- Open with a compliment before delivering criticism (the "compliment sandwich").
- Praise excessively. Real reviewers rarely praise — when they do, it's brief ("clean" or "nice").
- Use parallel structure across all comments (e.g. don't start every comment with "The [noun]...").
- End comments with a summary sentence restating what you just said.
- Use emoji.

EXAMPLES OF GOOD COMMENTS (study the tone and brevity):
- "This will panic on nil input — ` + "`cfg`" + ` isn't checked anywhere before this."
- "Race condition: two goroutines can both read ` + "`isExpired`" + ` as true and refresh the token. Wrap this in a sync.Once or add a mutex."
- "Nit: ` + "`userID`" + ` → ` + "`uid`" + ` for consistency with the rest of the file."
- "Is this fallthrough intentional?"
- "You're swallowing the error from ` + "`db.Close()`" + ` — at minimum log it."
- "This allocates on every request. Pull the slice outside the loop."
- "Same pattern as ` + "`handleAuth`" + ` above — worth extracting into a shared helper."
- "Nit: unused import."
- "The ` + "`ctx`" + ` you're passing here is the background context, not the request context. Subtle but this means cancellation won't propagate."

EXAMPLES OF BAD COMMENTS (never write like this):
- "I notice that the error handling could be improved here. Consider adding a nil check to ensure robustness."
- "This is a good approach, however, it might be beneficial to consider using a mutex for thread safety."
- "Great use of goroutines here! One thing worth noting is that this could potentially lead to a race condition."
- "It appears that this function doesn't handle the edge case where the input is nil. You may want to consider adding a nil check to prevent potential panics."

SEVERITY LEVELS:
- "critical" — Bugs, security issues, data loss risks, deadlocks, race conditions. Must fix before merge.
- "suggestion" — Performance improvements, better patterns, clearer abstractions. Code works but could be meaningfully better.
- "nit" — Style, naming, minor readability. Not worth blocking a PR over.
- "praise" — Genuinely good patterns worth calling out. Use very sparingly — only when something is notably well done. Keep it brief: one sentence max.

CODEBASE PATTERN CONSISTENCY:
You may be provided with existing source files from the same directories as the changed files. These represent the ESTABLISHED PATTERNS of the codebase.
When provided:
- Compare the PR's code against the patterns you see in the existing files.
- Flag deviations from established error handling patterns (e.g. wrapping errors, sentinel errors, logging).
- Flag inconsistent naming conventions (variable names, function names, struct patterns).
- Flag structural deviations (e.g. if other files use constructor functions and this file doesn't, or different interface patterns).
- Flag inconsistent test patterns if test files are provided.
- Only flag meaningful deviations — not trivial stylistic differences.
- Use severity "suggestion" for pattern deviations unless the deviation introduces a bug (then use "critical").
- Reference the specific existing file that demonstrates the established pattern.
- Do NOT comment on patterns if no codebase context is provided.

EXISTING COMMENTS CONTEXT:
You may be provided with existing comments, reviews, and line-level code comments already posted on this PR.
When provided:
- Do NOT repeat or rephrase feedback that has already been given.
- Do NOT comment on issues that have already been addressed in existing discussions.
- Focus on NEW issues, patterns, or concerns not yet raised.
- If you agree with an existing comment, you may briefly reference it (e.g. "+1 on the concern about X") but add new insight.
- Use existing comments to understand what reviewers care about and calibrate your review accordingly.

RESPONSE FORMAT:
You MUST respond with valid JSON matching this exact schema:

{
  "summary": "A 2-4 sentence high-level overview of the PR changes and your overall assessment. Write this like a senior engineer summarizing their review — blunt, specific, no filler.",
  "comments": [
    {
      "file": "path/to/file.go",
      "start_line": 42,
      "end_line": 42,
      "severity": "critical",
      "body": "The comment text."
    }
  ]
}

Rules for the JSON response:
- "file" must be the exact file path from the diff
- "start_line" and "end_line" are line numbers from the diff (use the same number for single-line comments)
- "severity" must be one of: "critical", "suggestion", "nit", "praise"
- "body" is the review comment text — it MUST read like a human wrote it
- Return ONLY valid JSON, no markdown code fences, no extra text`

	// Append focus mode section if active.
	for _, mode := range focusModes {
		if mode != "" {
			prompt += buildFocusModeSection(focusModes)
			break
		}
	}

	return prompt
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

	// Append codebase context for pattern analysis
	prompt += buildCodebaseContextSection(input)

	// Append existing comments as context
	prompt += buildExistingCommentsSection(input)

	// Append project rules from .prr.yaml
	prompt += buildProjectRulesSection(input)

	return prompt
}

// buildCodebaseContextSection formats codebase sibling files for inclusion in the prompt.
func buildCodebaseContextSection(input *ReviewInput) string {
	if len(input.CodebaseContext) == 0 {
		return ""
	}

	var section string
	section += "\n\n--- CODEBASE CONTEXT (existing files in the same directories — use these to check for pattern consistency) ---\n"
	for _, f := range input.CodebaseContext {
		section += fmt.Sprintf("\n--- %s ---\n%s\n", f.Path, f.Content)
	}
	return section
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

// buildProjectRulesSection formats project-level rules for inclusion in the prompt.
func buildProjectRulesSection(input *ReviewInput) string {
	if len(input.ProjectRules) == 0 {
		return ""
	}

	var section string
	section += "\n\n--- PROJECT RULES (.prr.yaml — enforce these in your review) ---\n"
	for _, r := range input.ProjectRules {
		section += fmt.Sprintf("- %s\n", r)
	}
	return section
}

// focusModeDescriptions maps focus mode names to system prompt instructions.
var focusModeDescriptions = map[string]string{
	"security":    "Focus on: authentication bypass, injection vulnerabilities (SQL, XSS, command), secrets in code, crypto misuse, SSRF, path traversal, input validation, missing authorization checks, insecure deserialization.",
	"performance": "Focus on: O(n²) loops, unnecessary heap allocations, N+1 query patterns, missing caching opportunities, memory leaks, goroutine leaks, lock contention, inefficient string concatenation, large copies.",
	"testing":     "Focus on: missing edge case coverage, test isolation problems, mock correctness, assertion specificity (overly broad or missing assertions), coverage gaps for error paths, flaky test patterns, test naming clarity.",
}

// buildFocusModeSection constructs the focus mode section for the system prompt.
func buildFocusModeSection(modes []string) string {
	var descriptions []string
	for _, mode := range modes {
		if desc, ok := focusModeDescriptions[mode]; ok {
			descriptions = append(descriptions, fmt.Sprintf("**%s**: %s", strings.ToUpper(mode), desc))
		}
	}
	if len(descriptions) == 0 {
		return ""
	}

	section := "\n\nFOCUS MODE ACTIVE:\n"
	for _, d := range descriptions {
		section += d + "\n"
	}
	section += "\nDeprioritize comments outside these focus areas. Only include non-focus comments if they are severity=critical."
	return section
}

// BuildDescribeSystemPrompt returns the system prompt for PR description generation.
func BuildDescribeSystemPrompt() string {
	return `You are a senior software engineer writing a pull request description.

Your job is to write a clear, concise, and informative PR description based on the diff.

WRITING STYLE:
- Start with a one-line summary of what the PR does.
- Follow with a "Changes" section listing the key modifications.
- If relevant, add a "Why" section explaining the motivation.
- Use markdown formatting (headers, bullet points, code blocks).
- Be specific about what changed — reference file names and function names.
- Keep it concise. Engineers read dozens of PRs — don't waste their time.
- Don't describe every single line change. Focus on the meaningful ones.
- Write like a senior engineer, not an AI. No filler, no fluff.

RESPONSE FORMAT:
Return ONLY the PR description as markdown. No JSON, no code fences wrapping the whole thing.`
}

// BuildDescribeUserPrompt constructs the user message for PR description generation.
func BuildDescribeUserPrompt(prTitle string, currentBody string, diff string, files []FileDiff) string {
	var prompt string
	prompt = fmt.Sprintf("Generate a PR description for:\n\nTitle: %s\n", prTitle)

	if currentBody != "" {
		prompt += fmt.Sprintf("\nCurrent description (improve or replace):\n%s\n", currentBody)
	}

	prompt += "\nDiff:\n"
	if len(files) > 0 {
		for _, f := range files {
			prompt += fmt.Sprintf("\n--- File: %s (status: %s) ---\n%s\n", f.Path, f.Status, f.Diff)
		}
	} else {
		prompt += diff
	}

	return prompt
}

// BuildAskSystemPrompt returns the system prompt for interactive follow-up questions.
func BuildAskSystemPrompt() string {
	return `You are a senior software engineer answering questions about a code review you previously performed.

You have full context from the review including the summary, all comments, and their severities.
Answer the question directly and concisely. Reference specific files and line numbers when relevant.
If you don't have enough context to answer, say so.`
}

// BuildAskUserPrompt constructs the user message for an interactive follow-up.
func BuildAskUserPrompt(reviewContext string, question string) string {
	return fmt.Sprintf("Here is the review context:\n\n%s\n\nQuestion: %s", reviewContext, question)
}
