package auth

import "time"

// Status represents authentication status for a provider
type Status struct {
	Provider      string            `json:"provider"`       // Provider name (e.g., "azure", "github")
	Authenticated bool              `json:"authenticated"`  // Whether authenticated
	User          string            `json:"user,omitempty"` // Username/email
	Details       map[string]string `json:"details"`        // Additional details
	Error         error             `json:"error,omitempty"`
	CheckedAt     time.Time         `json:"checked_at"`
}

// Checker interface for authentication checkers
type Checker interface {
	// Check performs authentication check
	Check() (*Status, error)

	// Name returns the provider name
	Name() string
}

// Cache stores authentication check results with TTL
type Cache struct {
	status    *Status
	timestamp time.Time
	ttl       time.Duration
}

// NewCache creates a new cache with given TTL
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		ttl: ttl,
	}
}

// Get returns cached status if not expired
func (c *Cache) Get() *Status {
	if c.status == nil {
		return nil
	}

	if time.Since(c.timestamp) > c.ttl {
		return nil // Expired
	}

	return c.status
}

// Set stores status in cache
func (c *Cache) Set(status *Status) {
	c.status = status
	c.timestamp = time.Now()
}

// Clear clears the cache
func (c *Cache) Clear() {
	c.status = nil
}
