package terraform

import (
	"sync"
	"time"
)

// Context represents the current Terraform working context
type Context struct {
	Workspace   string
	Backend     string
	Module      string
	StateFile   string
	RootModule  string
	Variables   map[string]string
	Environment map[string]string
	CheckedAt   time.Time
	Error       error
}

// Detector interface for different context detection strategies
type Detector interface {
	Detect(path string) (*Context, error)
	Name() string
}

// Cache stores context information with TTL
type Cache struct {
	context   *Context
	timestamp time.Time
	ttl       time.Duration
	mu        sync.RWMutex
}

// NewCache creates a new cache with specified TTL
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		ttl: ttl,
	}
}

// Get retrieves cached context if still valid
func (c *Cache) Get() *Context {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.context == nil {
		return nil
	}

	if time.Since(c.timestamp) > c.ttl {
		return nil
	}

	return c.context
}

// Set stores context in cache
func (c *Cache) Set(ctx *Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.context = ctx
	c.timestamp = time.Now()
}

// Clear removes cached context
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.context = nil
	c.timestamp = time.Time{}
}
