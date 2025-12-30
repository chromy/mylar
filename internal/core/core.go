package core

import (
	"github.com/chromy/viz/internal/cache"
	"sync"
)

var mu sync.RWMutex
var blobComputations map[string]BlobComputation = make(map[string]BlobComputation)
var routes map[string]Route = make(map[string]Route)
var theCache = cache.NewMemoryCache()
