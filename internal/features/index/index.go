package index

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromy/viz/internal/cache"
	"github.com/chromy/viz/internal/constants"
	"github.com/chromy/viz/internal/core"
	"github.com/chromy/viz/internal/features/repo"
	"github.com/chromy/viz/internal/schemas"
	"github.com/chromy/viz/internal/utils"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"sort"
	"strconv"
	"time"
	"log"
)

type IndexEntry struct {
	Path       string        `json:"path"`
	LineOffset int64         `json:"lineOffset"`
	LineCount  int64         `json:"lineCount"`
	Hash       plumbing.Hash `json:"hash"`
}

type Index struct {
	Entries []IndexEntry `json:"entries"`
}

// FindFileByLine returns the IndexEntry containing the given line number.
// Uses binary search since entries are sorted by LineOffset.
// Returns nil if the line number is not found in any file.
func (idx *Index) FindFileByLine(lineNumber int64) *IndexEntry {
	if len(idx.Entries) == 0 {
		return nil
	}

	i := sort.Search(len(idx.Entries), func(i int) bool {
		return idx.Entries[i].LineOffset > lineNumber
	})

	if i == 0 {
		return nil
	}

	entry := &idx.Entries[i-1]
	if lineNumber >= entry.LineOffset && lineNumber < entry.LineOffset+entry.LineCount {
		return entry
	}

	return nil
}

func (idx *Index) ToTileLayout() *utils.TileLayout {
	lastEntry := idx.Entries[len(idx.Entries)-1]
	lastLine := lastEntry.LineOffset + lastEntry.LineCount
	layout := utils.TileLayout{LastLine: utils.LinePosition(lastLine)}
	return &layout
}

var (
	// Global cache instance for index calculations
	indexCache cache.Cache
)

var GetBlobIndex = core.RegisterBlobComputation("blobIndex", func(ctx context.Context, repoId string, hash plumbing.Hash) (Index, error) {
	lineCount, err := repo.LineCount(ctx, repoId, hash)
	if err != nil {
		return Index{}, err
	}

	entry := IndexEntry{
		Path:       ".",
		LineOffset: 0,
		LineCount:  lineCount,
		Hash:       hash,
	}

	return Index{Entries: []IndexEntry{entry}}, nil
})

var GetTreeIndex = core.RegisterBlobComputation("treeIndex", func(ctx context.Context, repoId string, hash plumbing.Hash) (Index, error) {

	getIndex, found := core.GetBlobComputation("index")
	if !found {
		return Index{}, fmt.Errorf("index blob computation not found")
	}

	repository, err := repo.Get(ctx, repoId)
	if err != nil {
		return Index{}, err
	}

	treeObj, err := repository.TreeObject(hash)
	if err != nil {
		return Index{}, err
	}

	var allEntries []IndexEntry
	var currentOffset int64

	entries := make([]object.TreeEntry, 0, len(treeObj.Entries))
	entries = append(entries, treeObj.Entries...)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	for _, entry := range entries {
		childIndex, err := getIndex.Execute(ctx, repoId, entry.Hash)
		if err != nil {
			return Index{}, err
		}

		for _, childEntry := range childIndex.(Index).Entries {
			newEntry := IndexEntry{
				Path:       entry.Name,
				LineOffset: currentOffset,
				LineCount:  childEntry.LineCount,
				Hash:       childEntry.Hash,
			}

			if childEntry.Path != "." {
				newEntry.Path = entry.Name + "/" + childEntry.Path
			}

			allEntries = append(allEntries, newEntry)
			currentOffset += childEntry.LineCount
		}
	}

	return Index{Entries: allEntries}, nil
})

var GetIndex = core.RegisterBlobComputation("index", func(ctx context.Context, repoId string, hash plumbing.Hash) (Index, error) {
	objectType, err := repo.GetObjectType(ctx, repoId, hash)
	if err != nil {
		return Index{}, err
	}

	switch objectType {
	case "blob":
		return GetBlobIndex(ctx, repoId, hash)
	case "tree":
		return GetTreeIndex(ctx, repoId, hash)
	default:
		return Index{}, fmt.Errorf("index can't handle object type %s", objectType)
	}
})

func IndexHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	repoName := ps.ByName("repo")
	committish := ps.ByName("committish")

	if repoName == "" {
		http.Error(w, "repo must be set", http.StatusBadRequest)
		return
	}

	if committish == "" {
		http.Error(w, "committish must be set", http.StatusBadRequest)
		return
	}

	repository, err := repo.Get(r.Context(), repoName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	treeHash, err := repo.ResolveCommittishToTreeish(repository, committish)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	index, err := GetTreeIndex(r.Context(), repoName, treeHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(index); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

type LineLength struct {
	Maximum int64 `json:"maximum"`
}

var GetBlobLineLengths = core.RegisterBlobComputation("blobLineLengths", func(ctx context.Context, repoId string, hash plumbing.Hash) ([]int, error) {
	lines, err := repo.Lines(ctx, repoId, hash)
	if err != nil {
		return nil, err
	}

	var lengths = make([]int, 0, len(lines))
	for _, line := range lines {
		lengths = append(lengths, len(line))
	}

	return lengths, nil
})

func TileLineLength(ctx context.Context, repoId string, repository *git.Repository, hash plumbing.Hash, lod int64, x int64, y int64) ([]int64, error) {
	// Try to load from cache first
	cacheKey := core.GenerateCacheKey("tile", hash.String(), fmt.Sprintf("%d", lod), fmt.Sprintf("%d", x), fmt.Sprintf("%d", y))
	if cached, err := indexCache.Get(cacheKey); err == nil {
		var tile []int64
		if err := json.Unmarshal(cached, &tile); err == nil {
			return tile, nil
		}
		// If unmarshaling fails, continue with computation
	}

	if lod != 0 {
		return make([]int64, constants.TileSize*constants.TileSize), nil
	}

	tileSize := utils.LodToSize(int(lod))
	tile := make([]int64, tileSize*tileSize)

	// Get the index for the repository
	index, err := GetIndex(ctx, repoId, hash)
	if err != nil {
		return nil, err
	}

	layout := index.ToTileLayout()

	tilePos := utils.TilePosition{
		Lod:     lod,
		TileX:   x,
		TileY:   y,
		OffsetX: 0,
		OffsetY: 0,
	}


	// Get the world position for the top-left corner of this tile
	tileWorldPos := utils.TileToWorld(tilePos, *layout)

	cornerWorldPosition := utils.WorldPosition{
		X: tileWorldPos.X,
		Y: tileWorldPos.Y,
	}
	cornerLinePosition := utils.WorldToLine(cornerWorldPosition, *layout)
	cornerEntry := index.FindFileByLine(int64(cornerLinePosition))
	log.Printf("%d %d %d %v %v %v %v %v %v", lod, x, y, *layout, tilePos, tileWorldPos, cornerWorldPosition, cornerLinePosition, cornerEntry)

	// For each position in the tile, find the corresponding line and get its length
	for tileY := 0; tileY < tileSize; tileY++ {
		for tileX := 0; tileX < tileSize; tileX++ {
			// Calculate world position for this pixel in the tile
			worldPos := utils.WorldPosition{
				X: tileWorldPos.X + int64(tileX),
				Y: tileWorldPos.Y + int64(tileY),
			}

			// Convert to line position
			linePos := utils.WorldToLine(worldPos, *layout)

			// Find the file entry that contains this line using binary search
			if entry := index.FindFileByLine(int64(linePos)); entry != nil {
				// Get the object directly using the hash from the index entry
				obj, err := repository.Object(plumbing.AnyObject, entry.Hash)
				if err != nil {
					continue // Skip files we can't read
				}

				// Only process blobs (files)
				if blob, ok := obj.(*object.Blob); ok {
					lineLengths, err := GetBlobLineLengths(ctx, repoId, blob.Hash)
					if err != nil {
						continue // Skip files we can't process
					}

					// Get the line index within this file
					lineIdxInFile := int64(linePos) - entry.LineOffset
					if lineIdxInFile >= 0 && lineIdxInFile < int64(len(lineLengths)) {
						tileIdx := tileY*tileSize + tileX
						tile[tileIdx] = int64(lineLengths[lineIdxInFile])
					}
				}
			}
		}
	}

	// Cache the result with 30 minutes expiration (tiles are accessed frequently)
	if tileData, err := json.Marshal(tile); err == nil {
		indexCache.Add(cacheKey, tileData, 30*time.Minute)
	}

	return tile, nil
}

type TileMetadata struct {
	X   int64 `json:"y"`
	Y   int64 `json:"x"`
	Lod int64 `json:"lod"`
}

func TileLineLengthHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	repoName := ps.ByName("repo")
	if repoName == "" {
		http.Error(w, "repo must be set", http.StatusBadRequest)
		return
	}

	committish := ps.ByName("committish")
	if committish == "" {
		http.Error(w, "committish must be set", http.StatusBadRequest)
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

	repository, err := repo.Get(r.Context(), repoName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hash, err := repo.ResolveCommittishToTreeish(repository, committish)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tile, err := TileLineLength(r.Context(), repoName, repository, hash, lod, x, y)
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
	}
	if err := json.NewEncoder(w).Encode(tile); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func init() {
	// Initialize with in-memory cache - can be replaced with other implementations
	indexCache = cache.NewMemoryCache()

	core.RegisterRoute(core.Route{
		Id:      "index.get",
		Method:  http.MethodGet,
		Path:    "/api/repo/:repo/:committish/index/",
		Handler: IndexHandler,
	})

	//core.RegisterRoute(core.Route{
	//	Id:      "index.line_length",
	//	Method:  http.MethodGet,
	//	Path:    "/api/repo/:repo/:committish/line_length/",
	//	Handler: LineLengthHandler,
	//})

	core.RegisterRoute(core.Route{
		Id:      "tile.line_length",
		Method:  http.MethodGet,
		Path:    "/api/repo/:repo/:committish/tile/:lod/:x/:y/length",
		Handler: TileLineLengthHandler,
	})

	schemas.Register("index.IndexEntry", IndexEntry{})
	schemas.Register("index.Index", Index{})
	schemas.Register("index.LineLength", LineLength{})
	schemas.Register("tile.TileMetadata", TileMetadata{})
}
