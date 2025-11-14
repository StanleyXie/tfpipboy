package auth

import (
	"sync"
)

// Manager manages multiple authentication checkers
type Manager struct {
	checkers []Checker
	mu       sync.RWMutex
}

// NewManager creates a new authentication manager
func NewManager() *Manager {
	return &Manager{
		checkers: []Checker{
			NewAzureChecker(),
			NewGitHubChecker(),
			// Add more checkers here as they're implemented
		},
	}
}

// CheckAll checks all authentication providers concurrently
func (m *Manager) CheckAll() map[string]*Status {
	results := make(map[string]*Status)
	resultChan := make(chan *Status, len(m.checkers))
	var wg sync.WaitGroup

	// Check all providers concurrently
	for _, checker := range m.checkers {
		wg.Add(1)
		go func(c Checker) {
			defer wg.Done()
			status, _ := c.Check()
			if status != nil {
				resultChan <- status
			}
		}(checker)
	}

	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for status := range resultChan {
		results[status.Provider] = status
	}

	return results
}

// Check checks a specific provider
func (m *Manager) Check(providerName string) (*Status, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, checker := range m.checkers {
		if checker.Name() == providerName {
			return checker.Check()
		}
	}

	return nil, nil // Provider not found
}

// GetProviders returns list of available provider names
func (m *Manager) GetProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]string, len(m.checkers))
	for i, checker := range m.checkers {
		providers[i] = checker.Name()
	}
	return providers
}

// AddChecker adds a new authentication checker to the manager
func (m *Manager) AddChecker(checker Checker) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.checkers = append(m.checkers, checker)
}

// ClearCache clears all authentication caches
// Call this after commands that may affect authentication status
func (m *Manager) ClearCache() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, checker := range m.checkers {
		// Type assert to access cache - only works for our builtin checkers
		switch c := checker.(type) {
		case *AzureChecker:
			c.cache.Clear()
		case *GitHubChecker:
			c.cache.Clear()
		}
	}
}
