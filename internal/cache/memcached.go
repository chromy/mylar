package cache

import (
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

// MemcachedCache is a memcached implementation of the Cache interface.
type MemcachedCache struct {
	client *memcache.Client
}

// NewMemcachedCache creates a new memcached cache client.
// The serverList should contain memcached server addresses like ["localhost:11211"].
func NewMemcachedCache(serverList ...string) *MemcachedCache {
	return &MemcachedCache{
		client: memcache.New(serverList...),
	}
}

// Add stores a value in memcached with the given key and expiration duration.
func (c *MemcachedCache) Add(key string, value []byte, duration time.Duration) error {
	item := &memcache.Item{
		Key:   key,
		Value: value,
	}

	if duration > 0 {
		item.Expiration = int32(duration.Seconds())
	}

	return c.client.Set(item)
}

// Get retrieves a value from memcached by key.
func (c *MemcachedCache) Get(key string) ([]byte, error) {
	item, err := c.client.Get(key)
	if err != nil {
		if err == memcache.ErrCacheMiss {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return item.Value, nil
}