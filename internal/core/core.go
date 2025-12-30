package core

import (
	"sync"
	"github.com/chromy/viz/internal/cache"
)


var mu sync.RWMutex
var blobComputations map[string]BlobComputation = make(map[string]BlobComputation)
var routes map[string]Route =  make(map[string]Route)
var theCache = cache.NewMemoryCache()
