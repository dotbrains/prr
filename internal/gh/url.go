package gh

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParsePRURL parses a GitHub PR URL and extracts owner, repo, and PR number.
// Accepted formats:
//   - https://github.com/owner/repo/pull/123
//   - http://github.com/owner/repo/pull/123
//   - github.com/owner/repo/pull/123
func ParsePRURL(rawURL string) (owner, repo string, prNumber int, err error) {
	// Normalize: add scheme if missing so url.Parse works correctly.
	normalized := rawURL
	if !strings.Contains(normalized, "://") {
		normalized = "https://" + normalized
	}

	u, err := url.Parse(normalized)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	if u.Host != "github.com" {
		return "", "", 0, fmt.Errorf("unsupported host %q: only github.com is supported", u.Host)
	}

	// Expected path: /owner/repo/pull/123
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 || parts[2] != "pull" {
		return "", "", 0, fmt.Errorf("invalid PR URL %q: expected github.com/owner/repo/pull/NUMBER", rawURL)
	}

	owner = parts[0]
	repo = parts[1]

	prNumber, err = strconv.Atoi(parts[3])
	if err != nil || prNumber <= 0 {
		return "", "", 0, fmt.Errorf("invalid PR number in URL %q: must be a positive integer", rawURL)
	}

	return owner, repo, prNumber, nil
}

// IsPRURL returns true if the string looks like a GitHub PR URL.
func IsPRURL(s string) bool {
	return strings.Contains(s, "github.com/") && strings.Contains(s, "/pull/")
}
