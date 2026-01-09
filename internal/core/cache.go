package core

import (
	"github.com/chromy/mylar/internal/cache"
)

func InitCache(c cache.Cache) {
	theCache = c
}

func GetCache() cache.Cache {
	return theCache
}
