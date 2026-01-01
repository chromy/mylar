package core

import (
	"github.com/chromy/viz/internal/cache"
)

func InitCache(c cache.Cache) {
	theCache = c
}

func GetCache() cache.Cache {
	return theCache
}
