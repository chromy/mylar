package index

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/chromy/viz/internal/cache"
	"github.com/chromy/viz/internal/features/repo"
	"github.com/chromy/viz/internal/core"
	"github.com/chromy/viz/internal/schemas"
	"github.com/chromy/viz/internal/utils"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/julienschmidt/httprouter"
	"io"
	"iter"
	"net/http"
	"sort"
	"strconv"
	"strings"
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

var (
	// Global cache instance for index calculations
	indexCache cache.Cache
)

func generateCacheKey(parts ...string) string {
	combined := strings.Join(parts, ":")
	h := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%x", h)
}

func ComputeIndex(ctx context.Context, repository *git.Repository, hash plumbing.Hash) (*Index, error) {
	// Try to load from cache first
	cacheKey := generateCacheKey("index", hash.String())
	if cached, err := indexCache.Get(cacheKey); err == nil {
		var index Index
		if err := json.Unmarshal(cached, &index); err == nil {
			return &index, nil
		}
		// If unmarshaling fails, continue with computation
	}

	obj, err := repository.Object(plumbing.AnyObject, hash)
	if err != nil {
		return nil, err
	}

	var index *Index

	switch obj := obj.(type) {
	case *object.Blob:
		index, err = computeIndexForBlob(obj)
	case *object.Tree:
		index, err = computeIndexForTree(ctx, repository, obj)
	default:
		return nil, fmt.Errorf("unexpected object %v", obj)
	}

	if err != nil {
		return nil, err
	}

	// Cache the result with 1 hour expiration
	if indexData, err := json.Marshal(index); err == nil {
		indexCache.Add(cacheKey, indexData, time.Hour)
	}

	return index, nil
}

func computeIndexForBlob(blob *object.Blob) (*Index, error) {
	reader, err := blob.Reader()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	lineCount, err := countLines(reader)
	if err != nil {
		return nil, err
	}

	entry := IndexEntry{
		Path:       ".",
		LineOffset: 0,
		LineCount:  lineCount,
		Hash:       blob.Hash,
	}

	return &Index{Entries: []IndexEntry{entry}}, nil
}

func computeIndexForTree(ctx context.Context, repository *git.Repository, tree *object.Tree) (*Index, error) {
	var allEntries []IndexEntry
	var currentOffset int64

	entries := make([]object.TreeEntry, 0, len(tree.Entries))
	entries = append(entries, tree.Entries...)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	for _, entry := range entries {
		childIndex, err := ComputeIndex(ctx, repository, entry.Hash)
		if err != nil {
			return nil, err
		}

		for _, childEntry := range childIndex.Entries {
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

	return &Index{Entries: allEntries}, nil
}

// isBinary checks if content appears to be binary by looking for null bytes
// in the first 8000 bytes (similar to how Git detects binary files)
func isBinary(content []byte) bool {
	// Check up to 8000 bytes for null bytes
	checkLen := len(content)
	if checkLen > 8000 {
		checkLen = 8000
	}
	return bytes.IndexByte(content[:checkLen], 0) != -1
}

func countLines(reader io.Reader) (int64, error) {
	bufferedReader := bufio.NewReader(reader)

	// Peek at the beginning to check if it's binary - use smaller size to avoid buffer full errors
	peek, err := bufferedReader.Peek(512)
	if err != nil && err != io.EOF && err.Error() != "bufio: buffer full" {
		return 0, err
	}
	// If we got "buffer full", just use what we could peek
	if err != nil && err.Error() == "bufio: buffer full" {
		peek, _ = bufferedReader.Peek(bufferedReader.Buffered())
	}

	// For binary files, read all content and return file size as one line
	if len(peek) > 0 && isBinary(peek) {
		// Reset reader and read all content to get actual size
		content, err := io.ReadAll(bufferedReader)
		if err != nil {
			return 0, err
		}
		return int64(len(content) + len(peek)), nil
	}

	// For text files, count lines using ReadLine for large line support
	var lineCount int64
	var hasContent bool

	for {
		_, isPrefix, err := bufferedReader.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}

		hasContent = true

		// Only count as a line when we've read the complete line (not a prefix)
		if !isPrefix {
			lineCount++
		}
	}

	// Handle files without trailing newline
	if lineCount == 0 && hasContent {
		lineCount = 1
	}

	return lineCount, nil
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

	repository, err := repo.Get(r.Context(), repoName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hash, err := repo.ResolveCommittishToTreeish(repository, committish)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	index, err := ComputeIndex(r.Context(), repository, hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(index); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

type GranularLineLength struct {
	LinesLengths []int64
}

type LineLength struct {
	Maximum int64 `json:"maximum"`
}

func ComputeLineLength(ctx context.Context, repository *git.Repository, hash plumbing.Hash) (*LineLength, error) {
	// Try to load from cache first
	cacheKey := generateCacheKey("linelength", hash.String())
	if cached, err := indexCache.Get(cacheKey); err == nil {
		var lineLength LineLength
		if err := json.Unmarshal(cached, &lineLength); err == nil {
			return &lineLength, nil
		}
		// If unmarshaling fails, continue with computation
	}

	obj, err := repository.Object(plumbing.AnyObject, hash)
	if err != nil {
		return nil, err
	}

	var lineLength *LineLength

	switch obj := obj.(type) {
	case *object.Blob:
		lineLength, err = computeLineLengthForBlob(ctx, obj)
	case *object.Tree:
		lineLength, err = computeLineLengthForTree(ctx, repository, obj)
	default:
		return nil, fmt.Errorf("unexpected object %v", obj)
	}

	if err != nil {
		return nil, err
	}

	// Cache the result with 1 hour expiration
	if lineLengthData, err := json.Marshal(lineLength); err == nil {
		indexCache.Add(cacheKey, lineLengthData, time.Hour)
	}

	return lineLength, nil
}

func computeGranularLineLengthForBlob(ctx context.Context, blob *object.Blob) (*GranularLineLength, error) {
	// Try to load from cache first
	cacheKey := generateCacheKey("granular", blob.Hash.String())
	if cached, err := indexCache.Get(cacheKey); err == nil {
		var granular GranularLineLength
		if err := json.Unmarshal(cached, &granular); err == nil {
			return &granular, nil
		}
		// If unmarshaling fails, continue with computation
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	bufferedReader := bufio.NewReader(reader)

	// Peek at the beginning to check if it's binary - use smaller size to avoid buffer full errors
	peek, err := bufferedReader.Peek(512)
	if err != nil && err != io.EOF && err.Error() != "bufio: buffer full" {
		return nil, err
	}
	// If we got "buffer full", just use what we could peek
	if err != nil && err.Error() == "bufio: buffer full" {
		peek, _ = bufferedReader.Peek(bufferedReader.Buffered())
	}

	// For binary files, return the file size as one line
	if len(peek) > 0 && isBinary(peek) {
		// Read all remaining content to get actual size
		content, err := io.ReadAll(bufferedReader)
		if err != nil {
			return nil, err
		}
		totalSize := int64(len(content) + len(peek))
		granular := &GranularLineLength{LinesLengths: []int64{totalSize}}

		// Cache the result with 1 hour expiration
		if granularData, err := json.Marshal(granular); err == nil {
			indexCache.Add(cacheKey, granularData, time.Hour)
		}

		return granular, nil
	}

	// For text files, process lines normally using the updated Lines function
	var lineLengths []int64
	for line, err := range Lines(bufferedReader) {
		if err != nil {
			return nil, err
		}
		lineLengths = append(lineLengths, int64(len(line)))
	}

	granular := &GranularLineLength{LinesLengths: lineLengths}

	// Cache the result with 1 hour expiration
	if granularData, err := json.Marshal(granular); err == nil {
		indexCache.Add(cacheKey, granularData, time.Hour)
	}

	return granular, nil
}

func computeLineLengthForBlob(ctx context.Context, blob *object.Blob) (*LineLength, error) {
	granular, err := computeGranularLineLengthForBlob(ctx, blob)
	if err != nil {
		return nil, err
	}

	var maximum int64

	for _, n := range granular.LinesLengths {
		maximum = max(n, maximum)
	}

	return &LineLength{Maximum: int64(maximum)}, nil
}

func computeLineLengthForTree(ctx context.Context, repository *git.Repository, tree *object.Tree) (*LineLength, error) {
	var maxLength int64

	for _, entry := range tree.Entries {
		childLineLength, err := ComputeLineLength(ctx, repository, entry.Hash)
		if err != nil {
			return nil, err
		}

		if childLineLength.Maximum > maxLength {
			maxLength = childLineLength.Maximum
		}
	}

	return &LineLength{Maximum: maxLength}, nil
}

func Lines(reader io.Reader) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		bufferedReader := bufio.NewReader(reader)
		var line []byte

		for {
			chunk, isPrefix, err := bufferedReader.ReadLine()
			if err == io.EOF {
				// If we have a partial line without a trailing newline, yield it
				if len(line) > 0 {
					yield(string(line), nil)
				}
				break
			}
			if err != nil {
				yield("", err)
				return
			}

			// Append chunk to current line
			line = append(line, chunk...)

			// If this isn't a prefix (i.e., we've read the complete line), yield it
			if !isPrefix {
				if !yield(string(line), nil) {
					return
				}
				line = line[:0] // Reset line buffer
			}
		}
	}
}

func LineLengthHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

	hash, err := repo.ResolveCommittishToTreeish(repository, committish)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	index, err := ComputeLineLength(r.Context(), repository, hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(index); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func TileLineLength(ctx context.Context, repository *git.Repository, hash plumbing.Hash, lod int64, x int64, y int64) ([]int64, error) {
	// Try to load from cache first
	cacheKey := generateCacheKey("tile", hash.String(), fmt.Sprintf("%d", lod), fmt.Sprintf("%d", x), fmt.Sprintf("%d", y))
	if cached, err := indexCache.Get(cacheKey); err == nil {
		var tile []int64
		if err := json.Unmarshal(cached, &tile); err == nil {
			return tile, nil
		}
		// If unmarshaling fails, continue with computation
	}

	tileSize := utils.LodToSize(int(lod))
	tile := make([]int64, tileSize*tileSize)

	// Get the index for the repository
	index, err := ComputeIndex(ctx, repository, hash)
	if err != nil {
		return nil, err
	}

	// Create layout from index
	var lastLine utils.LinePosition = 0
	for _, entry := range index.Entries {
		entryEnd := utils.LinePosition(entry.LineOffset + entry.LineCount)
		if entryEnd > lastLine {
			lastLine = entryEnd
		}
	}
	layout := utils.TileLayout{LastLine: lastLine}

	// Create the tile position representing this tile
	tilePos := utils.TilePosition{
		Lod:     lod,
		TileX:   x,
		TileY:   y,
		OffsetX: 0,
		OffsetY: 0,
	}

	// Get the world position for the top-left corner of this tile
	tileWorldPos := utils.TileToWorld(tilePos, layout)

	// For each position in the tile, find the corresponding line and get its length
	for tileY := 0; tileY < tileSize; tileY++ {
		for tileX := 0; tileX < tileSize; tileX++ {
			// Calculate world position for this pixel in the tile
			worldPos := utils.WorldPosition{
				X: tileWorldPos.X + int64(tileX),
				Y: tileWorldPos.Y + int64(tileY),
			}

			// Convert to line position
			linePos := utils.WorldToLine(worldPos, layout)

			// Find the file entry that contains this line
			for _, entry := range index.Entries {
				entryStartLine := entry.LineOffset
				entryEndLine := entry.LineOffset + entry.LineCount

				if int64(linePos) >= entryStartLine && int64(linePos) < entryEndLine {
					// Get the object directly using the hash from the index entry
					obj, err := repository.Object(plumbing.AnyObject, entry.Hash)
					if err != nil {
						continue // Skip files we can't read
					}

					// Only process blobs (files)
					if blob, ok := obj.(*object.Blob); ok {
						// Use computeGranularLineLengthForBlob to get the length of each line
						granular, err := computeGranularLineLengthForBlob(ctx, blob)
						if err != nil {
							continue // Skip files we can't process
						}

						// Get the line index within this file
						lineIdxInFile := int64(linePos) - entryStartLine
						if lineIdxInFile >= 0 && lineIdxInFile < int64(len(granular.LinesLengths)) {
							tileIdx := tileY*tileSize + tileX
							tile[tileIdx] = granular.LinesLengths[lineIdxInFile]
						}
					}
					break // Found the file containing this line
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

	tile, err := TileLineLength(r.Context(), repository, hash, lod, x, y)
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

	core.RegisterRoute(core.Route{
		Id:      "index.line_length",
		Method:  http.MethodGet,
		Path:    "/api/repo/:repo/:committish/line_length/",
		Handler: LineLengthHandler,
	})

	core.RegisterRoute(core.Route{
		Id:      "tile.line_length",
		Method:  http.MethodGet,
		Path:    "/api/repo/:repo/:committish/tile/:lod/:x/:y/length",
		Handler: TileLineLengthHandler,
	})

	schemas.Register("index.IndexEntry", IndexEntry{})
	schemas.Register("index.Index", Index{})
	schemas.Register("index.GranularLineLength", GranularLineLength{})
	schemas.Register("index.LineLength", LineLength{})
	schemas.Register("tile.TileMetadata", TileMetadata{})
}
