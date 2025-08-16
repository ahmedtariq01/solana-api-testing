package cache

import (
	"sync"
	"time"
)

// CacheEntry represents a cached balance with its timestamp
type CacheEntry struct {
	Balance   float64
	Timestamp time.Time
}

// Cache provides thread-safe caching with TTL support
type Cache struct {
	data   map[string]*CacheEntry
	mutex  sync.RWMutex
	ttl    time.Duration
	stopCh chan struct{}
}

// New creates a new Cache instance with the specified TTL
func New(ttl time.Duration) *Cache {
	c := &Cache{
		data:   make(map[string]*CacheEntry),
		ttl:    ttl,
		stopCh: make(chan struct{}),
	}

	// Start cleanup goroutine
	go c.cleanup()

	return c
}

// Get retrieves a value from the cache if it exists and hasn't expired
func (c *Cache) Get(key string) (float64, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return 0, false
	}

	// Check if entry has expired
	if time.Since(entry.Timestamp) > c.ttl {
		return 0, false
	}

	return entry.Balance, true
}

// Set stores a value in the cache with the current timestamp
func (c *Cache) Set(key string, balance float64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = &CacheEntry{
		Balance:   balance,
		Timestamp: time.Now(),
	}
}

// Delete removes a key from the cache
func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, key)
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]*CacheEntry)
}

// Size returns the number of entries in the cache
func (c *Cache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.data)
}

// cleanup runs periodically to remove expired entries
func (c *Cache) cleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-c.stopCh:
			return
		}
	}
}

// removeExpired removes all expired entries from the cache
func (c *Cache) removeExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, entry := range c.data {
		if now.Sub(entry.Timestamp) > c.ttl {
			delete(c.data, key)
		}
	}
}

// Stop stops the cleanup goroutine
func (c *Cache) Stop() {
	close(c.stopCh)
}
