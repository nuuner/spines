package cache

import (
	"sync"
	"time"
)

// CacheEntry holds cached data with expiration time
type CacheEntry struct {
	Data      any
	ExpiresAt time.Time
}

// Cache is a thread-safe in-memory cache with TTL support
type Cache struct {
	mu    sync.RWMutex
	items map[string]CacheEntry
	ttl   time.Duration
}

// New creates a new cache with the specified TTL
func New(ttl time.Duration) *Cache {
	return &Cache{
		items: make(map[string]CacheEntry),
		ttl:   ttl,
	}
}

// Get retrieves an item from the cache. Returns the data and true if found and not expired,
// otherwise returns nil and false.
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	entry, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		// Entry has expired, remove it
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}

	return entry.Data, true
}

// Set stores an item in the cache with the default TTL
func (c *Cache) Set(key string, data any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = CacheEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}
