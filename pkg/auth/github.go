package auth

import (
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// GitHubChecker checks GitHub CLI authentication status
type GitHubChecker struct {
	cache *Cache
}

// NewGitHubChecker creates a new GitHub authentication checker
func NewGitHubChecker() *GitHubChecker {
	return &GitHubChecker{
		cache: NewCache(60 * time.Second), // 60 second TTL
	}
}

// Name returns the provider name
func (g *GitHubChecker) Name() string {
	return "github"
}

// Check performs GitHub CLI authentication check
func (g *GitHubChecker) Check() (*Status, error) {
	// Try to get from cache first
	if cached := g.cache.Get(); cached != nil {
		return cached, nil
	}

	// Perform actual check
	status := &Status{
		Provider:  "github",
		CheckedAt: time.Now(),
		Details:   make(map[string]string),
	}

	// Check if gh CLI is installed
	if !isCommandAvailable("gh") {
		status.Authenticated = false
		status.Details["error"] = "GitHub CLI not installed"
		g.cache.Set(status)
		return status, nil
	}

	// Execute: gh auth status
	cmd := exec.Command("gh", "auth", "status")
	output, err := cmd.CombinedOutput() // gh auth status outputs to stderr
	outputStr := string(output)

	if err != nil {
		// Not authenticated
		status.Authenticated = false
		if strings.Contains(outputStr, "not logged into") ||
			strings.Contains(outputStr, "You are not logged into any GitHub hosts") {
			status.Details["error"] = "Not logged in"
		} else {
			status.Details["error"] = "Check failed"
		}
		g.cache.Set(status)
		return status, nil
	}

	// Parse output for username and host
	// Output format example:
	// github.com
	//   ✓ Logged in to github.com as username (keyring)
	//   ✓ Git operations for github.com configured to use https protocol.
	//   ✓ Token: *******************

	status.Authenticated = true

	// Extract username using regex - try multiple formats
	// New format: "Logged in to github.com account username (keyring)"
	// Old format: "Logged in to github.com as username (keyring)"
	usernameRegex := regexp.MustCompile(`Logged in to [\w\.-]+ (?:account|as) ([\w\-]+)`)
	if matches := usernameRegex.FindStringSubmatch(outputStr); len(matches) > 1 {
		status.User = matches[1]
	}

	// Extract host
	hostRegex := regexp.MustCompile(`Logged in to ([\w\.-]+) (?:account|as)`)
	if matches := hostRegex.FindStringSubmatch(outputStr); len(matches) > 1 {
		status.Details["host"] = matches[1]
	} else {
		status.Details["host"] = "github.com" // Default
	}

	// Extract auth method (keyring, oauth token, etc.)
	authMethodRegex := regexp.MustCompile(`(?:account|as) [\w\-]+ \(([\w\s]+)\)`)
	if matches := authMethodRegex.FindStringSubmatch(outputStr); len(matches) > 1 {
		status.Details["auth_method"] = matches[1]
	}

	g.cache.Set(status)
	return status, nil
}
