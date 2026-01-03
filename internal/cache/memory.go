package cache

import (
	"sync"
	"time"
)

// cacheItem represents an item stored in the memory cache.
type cacheItem struct {
	data   []byte
	expiry time.Time
}

// isExpired checks if the cache item has expired.
func (item *cacheItem) isExpired() bool {
	return !item.expiry.IsZero() && time.Now().After(item.expiry)
}

// MemoryCache is an in-memory implementation of the Cache interface.
// It is thread-safe and supports expiration of cached items.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		items: make(map[string]*cacheItem),
	}
}

// Add stores a value in the cache with the given key.
func (c *MemoryCache) Add(key string, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Make a copy of the value to avoid external modifications
	data := make([]byte, len(value))
	copy(data, value)

	c.items[key] = &cacheItem{
		data:   data,
		expiry: time.Time{}, // Zero time means no expiration
	}

	return nil
}

// Get retrieves a value from the cache by key.
func (c *MemoryCache) Get(key string) ([]byte, error) {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		return nil, ErrNotFound
	}

	if item.isExpired() {
		// Clean up expired item
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, ErrNotFound
	}

	// Return a copy to prevent external modifications
	data := make([]byte, len(item.data))
	copy(data, item.data)
	return data, nil
}

// Size returns the number of items currently in the cache.
// This method is useful for debugging and monitoring.
func (c *MemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear removes all items from the cache.
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*cacheItem)
}
