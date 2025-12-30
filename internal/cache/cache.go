// Package cache provides a simple caching interface with Add and Get methods.
// This package is designed to support multiple backend implementations including
// in-memory, memcached, and GCS bucket storage.
package cache

import (
	"errors"
	"time"
)

// ErrNotFound is returned when a cache key is not found.
var ErrNotFound = errors.New("cache: key not found")

// Cache defines the interface for cache implementations.
type Cache interface {
	// Add stores a value in the cache with the given key and expiration duration.
	// If duration is 0, the item will not expire.
	Add(key string, value []byte, duration time.Duration) error

	// Get retrieves a value from the cache by key.
	// Returns ErrNotFound if the key doesn't exist or has expired.
	Get(key string) ([]byte, error)
}