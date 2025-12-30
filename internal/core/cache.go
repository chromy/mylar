package core

import (
	"github.com/chromy/viz/internal/cache"
)

func GetCacheLocked() cache.Cache {
	return theCache
}

func GetCache() cache.Cache {
	mu.RLock()
	defer mu.RUnlock()
	return theCache
}
