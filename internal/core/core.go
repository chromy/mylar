package core

import (
	"github.com/chromy/viz/internal/cache"
	"os"
	"sync"
)

var mu sync.RWMutex
var blobComputations map[string]BlobComputation = make(map[string]BlobComputation)
var commitComputations map[string]CommitComputation = make(map[string]CommitComputation)
var routes map[string]Route = make(map[string]Route)
var theCache cache.Cache = cache.NewMemoryCache()
var cacheOnce sync.Once

var storagePath string
var storageOnce sync.Once

func initStorage() {
	if envPath := os.Getenv("MYLAR_STORAGE"); envPath != "" {
		storagePath = envPath
	} else {
		tmpDir, err := os.MkdirTemp("", "viz-storage-")
		if err != nil {
			panic(err)
		}
		storagePath = tmpDir
	}
}

func GetStoragePath() string {
	storageOnce.Do(initStorage)
	return storagePath
}
