package core

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
)

type TileFunc func(ctx context.Context, repoId string, commit plumbing.Hash, lod int64, x int64, y int64) ([]int32, error)

type TileComputation struct {
	Id      string
	Execute TileFunc
}

var tileComputations map[string]TileComputation = make(map[string]TileComputation)

func wrapTileFuncWithCaching(id string, execute TileFunc) TileFunc {
	return func(ctx context.Context, repoId string, commit plumbing.Hash, lod int64, x int64, y int64) ([]int32, error) {
		cacheKey := GenerateCacheKey(id, commit.String(), fmt.Sprintf("%d", lod), fmt.Sprintf("%d", x), fmt.Sprintf("%d", y))

		if cached, err := theCache.Get(cacheKey); err == nil {
			tile := BytesToInt32Slice(cached)
			return tile, nil
		}

		result, err := execute(ctx, repoId, commit, lod, x, y)
		if err != nil {
			return nil, err
		}

		tileData := Int32SliceToBytes(result)
		theCache.Add(cacheKey, tileData)

		return result, nil
	}
}

func RegisterTileComputation(id string, execute TileFunc) TileFunc {
	mu.Lock()
	defer mu.Unlock()

	if _, found := tileComputations[id]; found {
		panic(fmt.Sprintf("tile computation already registered %s", id))
	}

	wrapped := wrapTileFuncWithCaching(id, execute)

	tileComputations[id] = TileComputation{
		Id:      id,
		Execute: wrapped,
	}

	return wrapped
}

func GetTileComputation(id string) (TileComputation, bool) {
	mu.RLock()
	defer mu.RUnlock()

	c, found := tileComputations[id]
	return c, found
}

func ListTileComputations() []string {
	mu.RLock()
	defer mu.RUnlock()

	ids := make([]string, 0, len(tileComputations))
	for id := range tileComputations {
		ids = append(ids, id)
	}
	return ids
}

func ResetTileComputationsForTesting() {
	mu.Lock()
	defer mu.Unlock()
	tileComputations = make(map[string]TileComputation)
}
