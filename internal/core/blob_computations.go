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

type ObjectFunc[T any] func(ctx context.Context, repoId string, hash plumbing.Hash) (T, error)

func wrapObjectFuncWithCaching[T any](id string, execute ObjectFunc[T]) ObjectFunc[T] {
	return func(ctx context.Context, repoId string, hash plumbing.Hash) (T, error) {
		c := GetCache()
		key := GenerateCacheKey(id, hash.String())

		if cached, err := c.Get(key); err == nil {
			var result T
			err := json.Unmarshal(cached, &result)

			if err != nil {
				// TODO: delete key
				var zero T
				return zero, err
			}

			return result, nil
		}

		result, err := execute(ctx, repoId, hash)
		if err != nil {
			var zero T
			return zero, err
		}

		serialized, err := json.Marshal(result)
		if err != nil {
			var zero T
			return zero, err
		}

		c.Add(key, serialized, time.Hour)

		return result, nil
	}
}

// BlobComputation defines a computation that can be performed on a Git blob
type BlobComputation struct {
	Id      string
	Execute ObjectFunc[interface{}]
}

func RegisterBlobComputation[T any](id string, execute ObjectFunc[T]) ObjectFunc[T] {
	mu.Lock()
	defer mu.Unlock()

	if _, found := blobComputations[id]; found {
		panic(fmt.Sprintf("blob computation already registered %s", id))
	}

	wrapped := wrapObjectFuncWithCaching(id, execute)

	// Store as interface{} type for backward compatibility
	interfaceWrapped := func(ctx context.Context, repoId string, hash plumbing.Hash) (interface{}, error) {
		return wrapped(ctx, repoId, hash)
	}

	blobComputations[id] = BlobComputation{
		Id:      id,
		Execute: interfaceWrapped,
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
