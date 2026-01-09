package core

import (
	"github.com/chromy/mylar/internal/cache"
	"os"
	"runtime/debug"
	"sync"
	"unsafe"
)

var mu sync.RWMutex
var blobComputations map[string]BlobComputation = make(map[string]BlobComputation)
var commitComputations map[string]CommitComputation = make(map[string]CommitComputation)
var routes map[string]Route = make(map[string]Route)
var theCache cache.Cache = cache.NewMemoryCache()
var cacheOnce sync.Once

var storagePath string
var storageOnce sync.Once

var version string
var versionOnce sync.Once

func initVersion() {
	version = "dev" // default fallback

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				version = setting.Value
				break
			}
		}
	}
}

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

// GetVersion returns the current version string
func GetVersion() string {
	versionOnce.Do(initVersion)
	return version
}

// Int32SliceToBytes converts []int32 to []byte using unsafe for zero-copy
func Int32SliceToBytes(slice []int32) []byte {
	if len(slice) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&slice[0])), len(slice)*4)
}

// BytesToInt32Slice converts []byte to []int32 using unsafe for zero-copy
func BytesToInt32Slice(data []byte) []int32 {
	if len(data) == 0 {
		return nil
	}
	return unsafe.Slice((*int32)(unsafe.Pointer(&data[0])), len(data)/4)
}
