package core

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"strings"
	"time"
)

type ObjectFunc func(ctx context.Context, repoId string, hash plumbing.Hash) (interface{}, error)

func wrapObjectFuncWithCaching(id string, execute ObjectFunc) ObjectFunc {
	return func(ctx context.Context, repoId string, hash plumbing.Hash) (interface{}, error) {
		c := GetCache()
		key := GenerateCacheKey(id, hash.String())

		if cached, err := c.Get(key); err == nil {
			var result interface{}
			err := json.Unmarshal(cached, &result)

			if err != nil {
				// TODO: delete key
				return nil, err
			}

			return result, nil
		}

		result, err := execute(ctx, repoId, hash)
		if err != nil {
			return nil, err
		}

		serialized, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}

		c.Add(key, serialized, time.Hour)

		return result, nil
	}
}

// BlobComputation defines a computation that can be performed on a Git blob
type BlobComputation struct {
	Id      string
	Execute ObjectFunc
}

func RegisterBlobComputation(id string, execute ObjectFunc) ObjectFunc {
	mu.Lock()
	defer mu.Unlock()

	if _, found := blobComputations[id]; found {
		panic(fmt.Sprintf("blob computation already registered %s", id))
	}

	wrapped := wrapObjectFuncWithCaching(id, execute)

	blobComputations[id] = BlobComputation{
		Id:      id,
		Execute: wrapped,
	}

	return wrapped
}

func GetBlobComputation(id string) (BlobComputation, bool) {
	mu.RLock()
	defer mu.RUnlock()

	c, found := blobComputations[id]
	return c, found
}

func ResetBlobComputationsForTesting() {
	mu.Lock()
	defer mu.Unlock()
	blobComputations = make(map[string]BlobComputation)
}

func GenerateCacheKey(parts ...string) string {
	combined := strings.Join(parts, ":")
	h := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%x", h)
}
