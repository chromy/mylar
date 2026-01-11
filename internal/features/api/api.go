package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromy/mylar/internal/constants"
	"github.com/chromy/mylar/internal/core"
//	"github.com/chromy/mylar/internal/features/index"
	"github.com/chromy/mylar/internal/schemas"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/sync/errgroup"
	"net/http"
	"strconv"
)

type TileMetadata struct {
	X   int64 `json:"x"`
	Y   int64 `json:"y"`
	Lod int64 `json:"lod"`
}

type AggregationType int

const (
	AggregationMean AggregationType = iota
	AggregationMode
	AggregationMax
	AggregationMin
)

func parseAggregationType(s string) (AggregationType, error) {
	switch s {
	case "mean":
		return AggregationMean, nil
	case "mode":
		return AggregationMode, nil
	case "max":
		return AggregationMax, nil
	case "min":
		return AggregationMin, nil
	default:
		return AggregationMean, fmt.Errorf("invalid aggregation type '%s'. Valid options: mean, mode, max, min", s)
	}
}

func aggregateValues(values []int32, agg AggregationType) int32 {
	if len(values) == 0 {
		return 0
	}

	switch agg {
	case AggregationMean:
		var sum int32
		for _, v := range values {
			sum += v
		}
		return sum / int32(len(values))

	case AggregationMax:
		max := values[0]
		for _, v := range values {
			if v > max {
				max = v
			}
		}
		return max

	case AggregationMin:
		min := values[0]
		for _, v := range values {
			if v < min {
				min = v
			}
		}
		return min

	case AggregationMode:
		// Find the most frequent value
		counts := make(map[int32]int)
		for _, v := range values {
			counts[v]++
		}

		var mode int32
		maxCount := 0
		for val, count := range counts {
			if count > maxCount {
				maxCount = count
				mode = val
			}
		}
		return mode

	default:
		// Default to mean if unknown aggregation type
		var sum int32
		for _, v := range values {
			sum += v
		}
		return sum / int32(len(values))
	}
}

func ComputeHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	repoId := ps.ByName("repoId")
	rawHash := ps.ByName("hash")
	computationId := ps.ByName("computationId")

	computation, found := core.GetBlobComputation(computationId)
	if !found {
		http.Error(w, fmt.Sprintf("Computation '%s' unknown", computationId), http.StatusNotFound)
		return
	}

	if !plumbing.IsHash(rawHash) {
		http.Error(w, fmt.Sprintf("Could not parse hash '%s'", rawHash), http.StatusNotFound)
		return
	}
	hash := plumbing.NewHash(rawHash)

	result, err := computation.Execute(r.Context(), repoId, hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func macroTile(ctx context.Context, computationId string, repoName string, commit plumbing.Hash, lod int64, x int64, y int64, agg AggregationType) ([]int32, error) {
	childLod := lod - 1
	xys := [][2]int64{
		{x * 2, y * 2},     // top-left
		{x*2 + 1, y * 2},   // top-right
		{x * 2, y*2 + 1},   // bottom-left
		{x*2 + 1, y*2 + 1}, // bottom-right
	}

	result := make([]int32, constants.TileSize*constants.TileSize)

	childTiles := make([][]int32, 4)
	g, ctx := errgroup.WithContext(ctx)

	for i, coords := range xys {
		i, coords := i, coords // capture loop variables
		g.Go(func() error {
			childX, childY := coords[0], coords[1]
			childTile, childErr := getTile(ctx, computationId, repoName, commit, childLod, childX, childY, agg)
			if childErr != nil {
				return fmt.Errorf("failed to fetch child tile (%d, %d) at LOD %d: %w", childX, childY, childLod, childErr)
			}
			childTiles[i] = childTile
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return result, err
	}

	for parentY := 0; parentY < constants.TileSize; parentY++ {
		for parentX := 0; parentX < constants.TileSize; parentX++ {
			// Each parent pixel corresponds to a 2x2 block in child tiles
			childBlockX := parentX * 2
			childBlockY := parentY * 2

			childValues := make([]int32, 0, 4)

			// Sample from appropriate child tiles based on position
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					childX := childBlockX + dx
					childY := childBlockY + dy

					// Determine which child tile this pixel belongs to
					var tileIdx int
					if childX >= constants.TileSize {
						// Right side
						if childY >= constants.TileSize {
							tileIdx = 3 // bottom-right
							childX -= constants.TileSize
							childY -= constants.TileSize
						} else {
							tileIdx = 1 // top-right
							childX -= constants.TileSize
						}
					} else {
						// Left side
						if childY >= constants.TileSize {
							tileIdx = 2 // bottom-left
							childY -= constants.TileSize
						} else {
							tileIdx = 0 // top-left
						}
					}

					if tileIdx < len(childTiles) && childTiles[tileIdx] != nil {
						childIdx := childY*constants.TileSize + childX
						if childIdx >= 0 && childIdx < len(childTiles[tileIdx]) {
							childValues = append(childValues, childTiles[tileIdx][childIdx])
						}
					}
				}
			}

			// Apply aggregation function
			var pixel int32
			if len(childValues) > 0 {
				pixel = aggregateValues(childValues, agg)
			}

			resultIdx := parentY*constants.TileSize + parentX
			result[resultIdx] = pixel
		}
	}

	return result, nil
}

func cachingMacroTile(ctx context.Context, computationId string, repoName string, commit plumbing.Hash, lod int64, x int64, y int64, agg AggregationType) ([]int32, error) {
	cacheKey := core.GenerateCacheKey("macroTile", computationId, repoName, commit.String(), fmt.Sprintf("%d", lod), fmt.Sprintf("%d", x), fmt.Sprintf("%d", y), fmt.Sprintf("%d", agg))

	cache := core.GetCache()
	if cached, err := cache.Get(cacheKey); err == nil {
		result := core.BytesToInt32Slice(cached)
		return result, nil
	}

	result, err := macroTile(ctx, computationId, repoName, commit, lod, x, y, agg)
	if err != nil {
		return nil, err
	}

	tileData := core.Int32SliceToBytes(result)
	cache.Add(cacheKey, tileData)

	return result, nil
}

func getTile(ctx context.Context, computationId string, repoName string, commit plumbing.Hash, lod int64, x int64, y int64, agg AggregationType) ([]int32, error) {
	//isBlank, err := index.IsBlankTile(ctx, repoName, commit, lod, x, y)
	//if err != nil {
	//	return []int32{}, err
	//}

	//if isBlank {
	//	return make([]int32, constants.TileSize*constants.TileSize), nil
	//}

	if lod == 0 {
		c, found := core.GetTileComputation(computationId)
		if !found {
			return []int32{}, fmt.Errorf("computation %s not found", computationId)
		}
		return c.Execute(ctx, repoName, commit, lod, x, y)
	} else {
		return cachingMacroTile(ctx, computationId, repoName, commit, lod, x, y, agg)
	}
}

func TileHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	repoName := ps.ByName("repoId")
	if repoName == "" {
		http.Error(w, "repo must be set", http.StatusBadRequest)
		return
	}

	commit := ps.ByName("commit")
	if commit == "" {
		http.Error(w, "commit must be set", http.StatusBadRequest)
		return
	}

	rawX := ps.ByName("x")
	if rawX == "" {
		http.Error(w, "x must be set", http.StatusBadRequest)
		return
	}
	x, err := strconv.ParseInt(rawX, 10, 64)
	if err != nil {
		http.Error(w, "x must be number", http.StatusBadRequest)
		return
	}

	rawY := ps.ByName("y")
	if rawY == "" {
		http.Error(w, "y must be set", http.StatusBadRequest)
		return
	}
	y, err := strconv.ParseInt(rawY, 10, 64)
	if err != nil {
		http.Error(w, "y must be number", http.StatusBadRequest)
		return
	}

	rawLod := ps.ByName("lod")
	if rawLod == "" {
		http.Error(w, "lod must be set", http.StatusBadRequest)
		return
	}
	lod, err := strconv.ParseInt(rawLod, 10, 64)
	if err != nil {
		http.Error(w, "lod must be number", http.StatusBadRequest)
		return
	}

	// Parse optional aggregation parameter
	aggStr := r.URL.Query().Get("agg")
	if aggStr == "" {
		aggStr = "mean" // default to mean
	}

	agg, err := parseAggregationType(aggStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tileComputationId := ps.ByName("tileComputationId")

	tile, err := getTile(r.Context(), tileComputationId, repoName, plumbing.NewHash(commit), lod, x, y, agg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	metadata := TileMetadata{
		X:   x,
		Y:   y,
		Lod: lod,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(tile); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

}

func ListBlobComputationsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	computations := core.ListBlobComputations()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(computations); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func ListTileComputationsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	computations := core.ListTileComputations()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(computations); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func ListCommitComputationsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	computations := core.ListCommitComputations()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(computations); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func CommitComputeHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	repoId := ps.ByName("repoId")
	rawCommitHash := ps.ByName("commit")
	rawHash := ps.ByName("hash")
	computationId := ps.ByName("computationId")

	computation, found := core.GetCommitComputation(computationId)
	if !found {
		http.Error(w, fmt.Sprintf("Computation '%s' unknown", computationId), http.StatusNotFound)
		return
	}

	if !plumbing.IsHash(rawCommitHash) {
		http.Error(w, fmt.Sprintf("Could not parse commit hash '%s'", rawCommitHash), http.StatusNotFound)
		return
	}
	commitHash := plumbing.NewHash(rawCommitHash)

	if !plumbing.IsHash(rawHash) {
		http.Error(w, fmt.Sprintf("Could not parse hash '%s'", rawHash), http.StatusNotFound)
		return
	}
	hash := plumbing.NewHash(rawHash)

	result, err := computation.Execute(r.Context(), repoId, commitHash, hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func TriggerCrashHandler(_ http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	panic("panic triggered")
}

func init() {

	core.RegisterRoute(core.Route{
		Id:      "api.crash",
		Method:  http.MethodPost,
		Path:    "/api/crash",
		Handler: TriggerCrashHandler,
	})

	core.RegisterRoute(core.Route{
		Id:      "api.compute",
		Method:  http.MethodGet,
		Path:    "/api/compute/:computationId/:repoId/:hash",
		Handler: ComputeHandler,
	})

	core.RegisterRoute(core.Route{
		Id:      "api.tile",
		Method:  http.MethodGet,
		Path:    "/api/tile/:tileComputationId/:repoId/:commit/:lod/:x/:y",
		Handler: TileHandler,
	})

	core.RegisterRoute(core.Route{
		Id:      "api.commit",
		Method:  http.MethodGet,
		Path:    "/api/commit/:computationId/:repoId/:commit/:hash",
		Handler: CommitComputeHandler,
	})

	core.RegisterRoute(core.Route{
		Id:      "api.list_blob_computations",
		Method:  http.MethodGet,
		Path:    "/api/blob_computations",
		Handler: ListBlobComputationsHandler,
	})

	core.RegisterRoute(core.Route{
		Id:      "api.list_tile_computations",
		Method:  http.MethodGet,
		Path:    "/api/tile_computations",
		Handler: ListTileComputationsHandler,
	})

	core.RegisterRoute(core.Route{
		Id:      "api.list_commit_computations",
		Method:  http.MethodGet,
		Path:    "/api/commit_computations",
		Handler: ListCommitComputationsHandler,
	})

	schemas.Register("api.TileMetadata", TileMetadata{})
}
