package index

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromy/viz/internal/constants"
	"github.com/chromy/viz/internal/core"
	"github.com/chromy/viz/internal/features/repo"
	"github.com/chromy/viz/internal/schemas"
	"github.com/chromy/viz/internal/utils"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"sort"
	"strconv"
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

	repository, err := repo.ResolveRepo(ctx, repoId)
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

	repository, err := repo.ResolveRepo(r.Context(), repoName)
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

func ExecuteTileComputation(ctx context.Context, repoId string, hash plumbing.Hash, lod int64, x int64, y int64, pixelFunc func(worldPos utils.WorldPosition, index *Index, layout *utils.TileLayout) int64) ([]int64, error) {
	// For non-zero LOD levels, return empty tiles
	if lod != 0 {
		return make([]int64, constants.TileSize*constants.TileSize), nil
	}

	tileSize := utils.LodToSize(int(lod))
	tile := make([]int64, tileSize*tileSize)

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

	// For each position in the tile, calculate the value using the provided pixel function
	for tileY := 0; tileY < tileSize; tileY++ {
		for tileX := 0; tileX < tileSize; tileX++ {
			// Calculate world position for this pixel in the tile
			worldPos := utils.WorldPosition{
				X: tileWorldPos.X + int64(tileX),
				Y: tileWorldPos.Y + int64(tileY),
			}

			tileIdx := tileY*tileSize + tileX
			tile[tileIdx] = pixelFunc(worldPos, &index, layout)
		}
	}

	return tile, nil
}

var GetTileLineOffset = core.RegisterTileComputation("offset", func(ctx context.Context, repoId string, hash plumbing.Hash, lod int64, x int64, y int64) ([]int64, error) {
	return ExecuteTileComputation(ctx, repoId, hash, lod, x, y, func(worldPos utils.WorldPosition, index *Index, layout *utils.TileLayout) int64 {
		linePos := utils.WorldToLine(worldPos, *layout)
		if entry := index.FindFileByLine(int64(linePos)); entry != nil {
			return int64(linePos) - entry.LineOffset
		}
		return 0
	})
})

var GetTileLineLength = core.RegisterTileComputation("length", func(ctx context.Context, repoId string, hash plumbing.Hash, lod int64, x int64, y int64) ([]int64, error) {
	return ExecuteTileComputation(ctx, repoId, hash, lod, x, y, func(worldPos utils.WorldPosition, index *Index, layout *utils.TileLayout) int64 {
		linePos := utils.WorldToLine(worldPos, *layout)
		if entry := index.FindFileByLine(int64(linePos)); entry != nil {
			lineLengths, err := GetBlobLineLengths(ctx, repoId, entry.Hash)
			if err != nil {
				return 0
			}
			lineIdxInFile := int64(linePos) - entry.LineOffset
			if lineIdxInFile >= 0 && lineIdxInFile < int64(len(lineLengths)) {
				return int64(lineLengths[lineIdxInFile])
			}
		}
		return 0
	})
})

var GetTileFileHash = core.RegisterTileComputation("fileHash", func(ctx context.Context, repoId string, hash plumbing.Hash, lod int64, x int64, y int64) ([]int64, error) {
	return ExecuteTileComputation(ctx, repoId, hash, lod, x, y, func(worldPos utils.WorldPosition, index *Index, layout *utils.TileLayout) int64 {
		linePos := utils.WorldToLine(worldPos, *layout)
		if entry := index.FindFileByLine(int64(linePos)); entry != nil {
			return utils.HashToInt53(entry.Hash)
		}
		return 0
	})
})

var GetTileFileExtension = core.RegisterTileComputation("fileExtension", func(ctx context.Context, repoId string, hash plumbing.Hash, lod int64, x int64, y int64) ([]int64, error) {
	return ExecuteTileComputation(ctx, repoId, hash, lod, x, y, func(worldPos utils.WorldPosition, index *Index, layout *utils.TileLayout) int64 {
		linePos := utils.WorldToLine(worldPos, *layout)
		if entry := index.FindFileByLine(int64(linePos)); entry != nil {
			path := entry.Path
			a := int64(path[len(path)-2])
			b := int64(path[len(path)-1])
			return (a << 8) | b
		}
		return 0
	})
})

type FileByLineResponse struct {
	Entry         IndexEntry          `json:"entry"`
	Content       string              `json:"content"`
	LineOffset    int64               `json:"lineOffset"`
	WorldPosition utils.WorldPosition `json:"worldPosition"`
	TilePosition  utils.TilePosition  `json:"tilePosition"`
}

func FileByLineHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

	rawLine := ps.ByName("line")
	if rawLine == "" {
		http.Error(w, "line must be set", http.StatusBadRequest)
		return
	}
	lineNumber, err := strconv.ParseInt(rawLine, 10, 64)
	if err != nil {
		http.Error(w, "line must be a number", http.StatusBadRequest)
		return
	}

	repository, err := repo.ResolveRepo(r.Context(), repoName)
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

	entry := index.FindFileByLine(lineNumber)
	if entry == nil {
		http.Error(w, fmt.Sprintf("line %d not found in any file", lineNumber), http.StatusNotFound)
		return
	}

	content, err := repo.Content(r.Context(), repoName, entry.Hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	layout := index.ToTileLayout()
	linePos := utils.LinePosition(lineNumber)
	worldPos := utils.LineToWorld(linePos, *layout)
	tilePos := utils.WorldToTile(worldPos, *layout)

	response := FileByLineResponse{
		Entry:         *entry,
		Content:       content,
		LineOffset:    lineNumber - entry.LineOffset,
		WorldPosition: worldPos,
		TilePosition:  tilePos,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func init() {
	core.RegisterRoute(core.Route{
		Id:      "index.get",
		Method:  http.MethodGet,
		Path:    "/api/repo/:repo/:committish/index/",
		Handler: IndexHandler,
	})

	core.RegisterRoute(core.Route{
		Id:      "index.file_by_line",
		Method:  http.MethodGet,
		Path:    "/api/repo/:repo/:committish/index/line/:line",
		Handler: FileByLineHandler,
	})

	schemas.Register("index.IndexEntry", IndexEntry{})
	schemas.Register("index.Index", Index{})
	schemas.Register("index.LineLength", LineLength{})
	schemas.Register("index.FileByLineResponse", FileByLineResponse{})
}
