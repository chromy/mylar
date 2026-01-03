// Package cache provides a simple caching interface with Add and Get methods.
// This package is designed to support multiple backend implementations including
// in-memory, memcached, and GCS bucket storage.
package cache

import (
	"errors"
)

// ErrNotFound is returned when a cache key is not found.
var ErrNotFound = errors.New("cache: key not found")

// Cache defines the interface for cache implementations.
type Cache interface {
	// Add stores a value in the cache with the given key.
	// Items will not expire.
	Add(key string, value []byte) error

	// Get retrieves a value from the cache by key.
	// Returns ErrNotFound if the key doesn't exist.
	Get(key string) ([]byte, error)
}
