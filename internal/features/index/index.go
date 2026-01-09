package index

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromy/mylar/internal/constants"
	"github.com/chromy/mylar/internal/core"
	"github.com/chromy/mylar/internal/features/repo"
	"github.com/chromy/mylar/internal/schemas"
	"github.com/chromy/mylar/internal/utils"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
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

func (idx *Index) ToTileLayout() utils.TileLayout {
	lastEntry := idx.Entries[len(idx.Entries)-1]
	lineCount := lastEntry.LineOffset + lastEntry.LineCount
	layout := utils.TileLayout{LineCount: utils.LinePosition(lineCount)}
	return layout
}

var mu sync.RWMutex
var indexCache map[string]*Index = make(map[string]*Index)

func IsBlankTile(ctx context.Context, repoId string, commit plumbing.Hash, lod int64, x int64, y int64) (bool, error) {
	tree, err := repo.CommitToTree(ctx, repoId, commit)
	if err != nil {
		return false, err
	}

	index, err := GetIndex(ctx, repoId, tree)
	if err != nil {
		return false, err
	}

	layout := index.ToTileLayout()

	tile := utils.TilePosition{
		Lod:     lod,
		TileX:   x,
		TileY:   y,
		OffsetX: 0,
		OffsetY: 0,
	}
	world := utils.TileToWorld(tile, layout)
	line := utils.WorldToLine(world, layout)
	return line >= layout.LineCount, nil
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
		return Index{}, fmt.Errorf("getting tree object %s: %s", hash, err)
	}

	var allEntries []IndexEntry
	var currentOffset int64

	entries := make([]object.TreeEntry, 0, len(treeObj.Entries))
	entries = append(entries, treeObj.Entries...)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	for _, entry := range entries {
		if entry.Mode == filemode.Submodule {
			continue
		}
		childIndex, err := getIndex.Execute(ctx, repoId, entry.Hash)
		if err != nil {
			return Index{}, fmt.Errorf("getting child index %s: %s", hash, err)
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

var GetIndexInternal = core.RegisterBlobComputation("index", func(ctx context.Context, repoId string, hash plumbing.Hash) (Index, error) {
	objectType, err := repo.GetObjectType(ctx, repoId, hash)
	if err != nil {
		return Index{}, fmt.Errorf("getting object type %s: %s", hash, err)
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

func GetIndex(ctx context.Context, repoId string, hash plumbing.Hash) (*Index, error) {
	cacheKey := repoId + ":" + hash.String()

	// Try to get from cache first
	mu.RLock()
	if cached, found := indexCache[cacheKey]; found {
		mu.RUnlock()
		return cached, nil
	}
	mu.RUnlock()

	// Not in cache, compute it
	index, err := GetIndexInternal(ctx, repoId, hash)
	if err != nil {
		return nil, err
	}

	// Cache the result
	mu.Lock()
	indexCache[cacheKey] = &index
	mu.Unlock()

	return &index, nil
}

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

type BlameLine struct {
	Author     string    `json:"author"`
	AuthorName string    `json:"authorName"`
	Hash       string    `json:"hash"`
	Date       time.Time `json:"date"`
	Text       string    `json:"text"`
}

type BlameResult struct {
	Path  string      `json:"path"`
	Lines []BlameLine `json:"lines"`
}

var GetBlame = core.RegisterCommitComputation("blame", func(ctx context.Context, repoId string, commit plumbing.Hash, hash plumbing.Hash) (BlameResult, error) {
	repository, err := repo.ResolveRepo(ctx, repoId)
	if err != nil {
		return BlameResult{}, err
	}

	ptr, err := repository.CommitObject(commit)
	if err != nil {
		return BlameResult{}, err
	}

	// TODO
	path := "README.md"

	blameResult, err := git.Blame(ptr, path)
	if err != nil {
		return BlameResult{}, fmt.Errorf("blame failed: %w", err)
	}

	lines := make([]BlameLine, len(blameResult.Lines))
	for i, line := range blameResult.Lines {
		lines[i] = BlameLine{
			Author:     line.Author,
			AuthorName: line.AuthorName,
			Hash:       line.Hash.String(),
			Date:       line.Date,
			Text:       line.Text,
		}
	}

	return BlameResult{
		Path:  blameResult.Path,
		Lines: lines,
	}, nil
})

func ExecuteTileComputation(ctx context.Context, repoId string, commit plumbing.Hash, lod int64, x int64, y int64, pixelFunc func(worldPos utils.WorldPosition, index *Index, layout utils.TileLayout) int32) ([]int32, error) {
	if lod != 0 {
		return make([]int32, constants.TileSize*constants.TileSize), nil
	}

	tileSize := utils.LodToSize(int(lod))
	tile := make([]int32, constants.TileSize*constants.TileSize)

	tree, err := repo.CommitToTree(ctx, repoId, commit)
	if err != nil {
		return nil, err
	}

	index, err := GetIndex(ctx, repoId, tree)
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
	tileWorldPos := utils.TileToWorld(tilePos, layout)

	// For each position in the tile, calculate the value using the provided pixel function
	for tileY := 0; tileY < tileSize; tileY++ {
		for tileX := 0; tileX < tileSize; tileX++ {
			// Calculate world position for this pixel in the tile
			worldPos := utils.WorldPosition{
				X: tileWorldPos.X + int64(tileX),
				Y: tileWorldPos.Y + int64(tileY),
			}

			tileIdx := tileY*tileSize + tileX
			tile[tileIdx] = pixelFunc(worldPos, index, layout)
		}
	}

	return tile, nil
}

//type Shader interface {
//	Pixel(world *utils.WorldPosition, index *Index) int32
//}

var GetTileLineOffset = core.RegisterTileComputation("offset", func(ctx context.Context, repoId string, commit plumbing.Hash, lod int64, x int64, y int64) ([]int32, error) {
	return ExecuteTileComputation(ctx, repoId, commit, lod, x, y, func(worldPos utils.WorldPosition, index *Index, layout utils.TileLayout) int32 {
		linePos := utils.WorldToLine(worldPos, layout)
		if entry := index.FindFileByLine(int64(linePos)); entry != nil {
			return int32(int64(linePos) - entry.LineOffset)
		}
		return 0
	})
})

//var GetTileLineLength = core.RegisterTileComputation("length", func(ctx context.Context, repoId string, commit plumbing.Hash, lod int64, x int64, y int64) ([]int32, error) {
//	return ExecuteTileComputation(ctx, repoId, commit, lod, x, y, func(worldPos utils.WorldPosition, index *Index, layout *utils.TileLayout) int32 {
//		linePos := utils.WorldToLine(worldPos, *layout)
//		if entry := index.FindFileByLine(int64(linePos)); entry != nil {
//			lineLengths, err := GetBlobLineLengths(ctx, repoId, entry.Hash)
//			if err != nil {
//				return 0
//			}
//			lineIdxInFile := int64(linePos) - entry.LineOffset
//			if lineIdxInFile >= 0 && lineIdxInFile < int64(len(lineLengths)) {
//				return int32(lineLengths[lineIdxInFile])
//			}
//		}
//		return 0
//	})
//})

var GetTileLineLength = core.RegisterTileComputation("length", func(ctx context.Context, repoId string, commit plumbing.Hash, lod int64, x int64, y int64) ([]int32, error) {
	if lod != 0 {
		return make([]int32, constants.TileSize*constants.TileSize), nil
	}

	tileSize := utils.LodToSize(int(lod))
	tile := make([]int32, constants.TileSize*constants.TileSize)

	tree, err := repo.CommitToTree(ctx, repoId, commit)
	if err != nil {
		return nil, err
	}

	index, err := GetIndex(ctx, repoId, tree)
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
	tileWorldPos := utils.TileToWorld(tilePos, layout)

	m := make(map[plumbing.Hash][]int)

	// For each position in the tile, calculate the value using the provided pixel function
	for tileY := 0; tileY < tileSize; tileY++ {
		for tileX := 0; tileX < tileSize; tileX++ {
			// Calculate world position for this pixel in the tile
			worldPos := utils.WorldPosition{
				X: tileWorldPos.X + int64(tileX),
				Y: tileWorldPos.Y + int64(tileY),
			}

			tileIdx := tileY*tileSize + tileX

			linePos := utils.WorldToLine(worldPos, layout)

			if entry := index.FindFileByLine(int64(linePos)); entry != nil {

				lineLengths, found := m[entry.Hash]
				if !found {
					lineLengths, err = GetBlobLineLengths(ctx, repoId, entry.Hash)
					if err != nil {
						return tile, err
					}
					m[entry.Hash] = lineLengths
				}

				lineIdxInFile := int64(linePos) - entry.LineOffset
				if lineIdxInFile >= 0 && lineIdxInFile < int64(len(lineLengths)) {
					tile[tileIdx] = int32(lineLengths[lineIdxInFile])
				}
			}

		}
	}
	return tile, nil
})

var GetTileLineIndent = core.RegisterTileComputation("indent", func(ctx context.Context, repoId string, commit plumbing.Hash, lod int64, x int64, y int64) ([]int32, error) {
	if lod != 0 {
		return make([]int32, constants.TileSize*constants.TileSize), nil
	}

	tileSize := utils.LodToSize(int(lod))
	tile := make([]int32, constants.TileSize*constants.TileSize)

	tree, err := repo.CommitToTree(ctx, repoId, commit)
	if err != nil {
		return nil, err
	}

	index, err := GetIndex(ctx, repoId, tree)
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
	tileWorldPos := utils.TileToWorld(tilePos, layout)

	m := make(map[plumbing.Hash][]int)

	// For each position in the tile, calculate the value using the provided pixel function
	for tileY := 0; tileY < tileSize; tileY++ {
		for tileX := 0; tileX < tileSize; tileX++ {
			// Calculate world position for this pixel in the tile
			worldPos := utils.WorldPosition{
				X: tileWorldPos.X + int64(tileX),
				Y: tileWorldPos.Y + int64(tileY),
			}

			tileIdx := tileY*tileSize + tileX

			linePos := utils.WorldToLine(worldPos, layout)

			if entry := index.FindFileByLine(int64(linePos)); entry != nil {

				lineIndents, found := m[entry.Hash]
				if !found {
					lineIndents, err = repo.LineIndents(ctx, repoId, entry.Hash)
					if err != nil {
						return tile, err
					}
					m[entry.Hash] = lineIndents
				}

				lineIdxInFile := int64(linePos) - entry.LineOffset
				if lineIdxInFile >= 0 && lineIdxInFile < int64(len(lineIndents)) {
					tile[tileIdx] = int32(lineIndents[lineIdxInFile])
				}
			}

		}
	}
	return tile, nil
})

var GetTileFileHash = core.RegisterTileComputation("fileHash", func(ctx context.Context, repoId string, commit plumbing.Hash, lod int64, x int64, y int64) ([]int32, error) {
	return ExecuteTileComputation(ctx, repoId, commit, lod, x, y, func(worldPos utils.WorldPosition, index *Index, layout utils.TileLayout) int32 {
		linePos := utils.WorldToLine(worldPos, layout)
		if entry := index.FindFileByLine(int64(linePos)); entry != nil {
			return utils.HashToInt32(entry.Hash)
		}
		return 0
	})
})

var GetTileFileExtension = core.RegisterTileComputation("fileExtension", func(ctx context.Context, repoId string, commit plumbing.Hash, lod int64, x int64, y int64) ([]int32, error) {
	return ExecuteTileComputation(ctx, repoId, commit, lod, x, y, func(worldPos utils.WorldPosition, index *Index, layout utils.TileLayout) int32 {
		linePos := utils.WorldToLine(worldPos, layout)
		if entry := index.FindFileByLine(int64(linePos)); entry != nil {
			ext := filepath.Ext(entry.Path)
			if len(ext) > 1 {
				ext = ext[1:]
			}

			var result int32
			for i := 0; i < len(ext) && i < 4; i++ {
				result = (result << 8) | int32(ext[i])
			}
			return result
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
	worldPos := utils.LineToWorld(linePos, layout)
	tilePos := utils.WorldToTile(worldPos, layout)

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
