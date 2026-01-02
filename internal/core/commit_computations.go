package core

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"time"
)

type CommitFunc[T any] func(ctx context.Context, repoId string, commit plumbing.Hash, hash plumbing.Hash) (T, error)

func wrapCommitFuncWithCaching[T any](id string, execute CommitFunc[T]) CommitFunc[T] {
	return func(ctx context.Context, repoId string, commit plumbing.Hash, hash plumbing.Hash) (T, error) {
		c := GetCache()
		key := GenerateCacheKey(id, commit.String(), hash.String())

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

		result, err := execute(ctx, repoId, commit, hash)
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

type CommitComputation struct {
	Id      string
	Execute CommitFunc[interface{}]
}

func RegisterCommitComputation[T any](id string, execute CommitFunc[T]) CommitFunc[T] {
	mu.Lock()
	defer mu.Unlock()

	if _, found := commitComputations[id]; found {
		panic(fmt.Sprintf("commit computation already registered %s", id))
	}

	wrapped := wrapCommitFuncWithCaching(id, execute)

	interfaceWrapped := func(ctx context.Context, repoId string, commit plumbing.Hash, hash plumbing.Hash) (interface{}, error) {
		return wrapped(ctx, repoId, commit, hash)
	}

	commitComputations[id] = CommitComputation{
		Id:      id,
		Execute: interfaceWrapped,
	}

	return wrapped
}

func GetCommitComputation(id string) (CommitComputation, bool) {
	mu.RLock()
	defer mu.RUnlock()

	c, found := commitComputations[id]
	return c, found
}

func ListCommitComputations() []string {
	mu.RLock()
	defer mu.RUnlock()
	
	ids := make([]string, 0, len(commitComputations))
	for id := range commitComputations {
		ids = append(ids, id)
	}
	return ids
}
