package orchestrator

import (
	"fmt"
	"sort"
	"strings"
)

// findSimilarTargets finds similar module or instance names to suggest to the user
func findSimilarTargets(target string, config *Config, maxSuggestions int) []string {
	if maxSuggestions == 0 {
		maxSuggestions = 5
	}

	type candidate struct {
		name     string
		module   string
		isModule bool
		score    int
	}

	var candidates []candidate

	// Check all module names
	for moduleName := range config.Modules {
		score := similarityScore(target, moduleName)
		if score > 0 {
			candidates = append(candidates, candidate{
				name:     moduleName,
				module:   moduleName,
				isModule: true,
				score:    score,
			})
		}
	}

	// Check all instance names
	for moduleName, module := range config.Modules {
		for instanceName := range module.Instances {
			score := similarityScore(target, instanceName)
			if score > 0 {
				candidates = append(candidates, candidate{
					name:     instanceName,
					module:   moduleName,
					isModule: false,
					score:    score,
				})
			}
		}
	}

	// Sort by score (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Take top N suggestions
	var suggestions []string
	for i := 0; i < len(candidates) && i < maxSuggestions; i++ {
		c := candidates[i]
		if c.isModule {
			suggestions = append(suggestions, fmt.Sprintf("  - %s (module)", c.name))
		} else {
			suggestions = append(suggestions, fmt.Sprintf("  - %s (instance in module: %s)", c.name, c.module))
		}
	}

	return suggestions
}

// similarityScore calculates a simple similarity score between two strings
// Returns higher scores for better matches
func similarityScore(s1, s2 string) int {
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	// Exact match
	if s1 == s2 {
		return 1000
	}

	score := 0

	// Contains match (substring)
	if strings.Contains(s2, s1) || strings.Contains(s1, s2) {
		score += 500
	}

	// Prefix match
	if strings.HasPrefix(s2, s1) || strings.HasPrefix(s1, s2) {
		score += 300
	}

	// Common prefix length
	commonPrefix := 0
	for i := 0; i < len(s1) && i < len(s2); i++ {
		if s1[i] == s2[i] {
			commonPrefix++
		} else {
			break
		}
	}
	score += commonPrefix * 20

	// Levenshtein-like: count common characters (simple version)
	chars1 := make(map[rune]int)
	chars2 := make(map[rune]int)
	for _, c := range s1 {
		chars1[c]++
	}
	for _, c := range s2 {
		chars2[c]++
	}

	commonChars := 0
	for c, count1 := range chars1 {
		if count2, exists := chars2[c]; exists {
			commonChars += min(count1, count2)
		}
	}
	score += commonChars * 5

	// Penalty for length difference
	lengthDiff := abs(len(s1) - len(s2))
	score -= lengthDiff * 3

	// Word boundary matches (hyphens, underscores)
	parts1 := strings.FieldsFunc(s1, func(r rune) bool { return r == '-' || r == '_' })
	parts2 := strings.FieldsFunc(s2, func(r rune) bool { return r == '-' || r == '_' })

	for _, p1 := range parts1 {
		for _, p2 := range parts2 {
			if p1 == p2 {
				score += 50
			}
		}
	}

	return score
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// formatTargetNotFoundError creates a user-friendly error message with suggestions
func formatTargetNotFoundError(target string, config *Config) error {
	suggestions := findSimilarTargets(target, config, 5)

	var errMsg strings.Builder
	errMsg.WriteString(fmt.Sprintf("target '%s' not found", target))

	if len(suggestions) > 0 {
		errMsg.WriteString("\n\nDid you mean one of these?")
		for _, suggestion := range suggestions {
			errMsg.WriteString("\n")
			errMsg.WriteString(suggestion)
		}
	} else {
		errMsg.WriteString("\n\nNo similar targets found. Use --list-modules to see available modules and instances.")
	}

	return fmt.Errorf("%s", errMsg.String())
}
