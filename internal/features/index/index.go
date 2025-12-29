package index

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromy/viz/internal/constants"
	"github.com/chromy/viz/internal/features/repo"
	"github.com/chromy/viz/internal/routes"
	"github.com/chromy/viz/internal/schemas"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/julienschmidt/httprouter"
	"io"
	"iter"
	"net/http"
	"sort"
)

type IndexEntry struct {
	Path       string `json:"path"`
	LineOffset int64  `json:"lineOffset"`
	LineCount  int64  `json:"lineCount"`
}

type Index struct {
	Entries []IndexEntry `json:"entries"`
}

func ComputeIndex(ctx context.Context, repository *git.Repository, hash plumbing.Hash) (*Index, error) {
	// TODO: Try and load from cache

	obj, err := repository.Object(plumbing.AnyObject, hash)
	if err != nil {
		return nil, err
	}

	switch obj := obj.(type) {
	case *object.Blob:
		return computeIndexForBlob(obj)
	case *object.Tree:
		return computeIndexForTree(ctx, repository, obj)
	default:
		return nil, fmt.Errorf("unexpected object %v", obj)
	}
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
	// TODO: Try and load from cache

	obj, err := repository.Object(plumbing.AnyObject, hash)
	if err != nil {
		return nil, err
	}

	switch obj := obj.(type) {
	case *object.Blob:
		return computeLineLengthForBlob(ctx, obj)
	case *object.Tree:
		return computeLineLengthForTree(ctx, repository, obj)
	default:
		return nil, fmt.Errorf("unexpected object %v", obj)
	}
}

func computeGranularLineLengthForBlob(ctx context.Context, blob *object.Blob) (*GranularLineLength, error) {
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
		return &GranularLineLength{LinesLengths: []int64{totalSize}}, nil
	}

	// For text files, process lines normally using the updated Lines function
	var lineLengths []int64
	for line, err := range Lines(bufferedReader) {
		if err != nil {
			return nil, err
		}
		lineLengths = append(lineLengths, int64(len(line)))
	}

	return &GranularLineLength{LinesLengths: lineLengths}, nil
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

func TileLineLength(ctx context.Context, repository *git.Repository, level int, x int, y int) ([]int64, error) {
	tile := make([]int64, constants.TileSize*constants.TileSize)

	for i := range tile {
		tile[i] = int64(i)
	}

	return tile, nil
}

func TileLineLengthHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

	// For now, using default values for level, x, y
	// TODO: Extract these from query parameters if needed
	level := 0
	x := 0
	y := 0

	tile, err := TileLineLength(r.Context(), repository, level, x, y)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tile); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func init() {
	routes.Register(routes.Route{
		Id:      "index.get",
		Method:  http.MethodGet,
		Path:    "/api/repo/:repo/:committish/index/",
		Handler: IndexHandler,
	})

	routes.Register(routes.Route{
		Id:      "index.line_length",
		Method:  http.MethodGet,
		Path:    "/api/repo/:repo/:committish/line_length/",
		Handler: LineLengthHandler,
	})

	routes.Register(routes.Route{
		Id:      "tile.line_length",
		Method:  http.MethodGet,
		Path:    "/api/repo/:repo/:committish/tile/length/",
		Handler: TileLineLengthHandler,
	})

	schemas.Register("index.IndexEntry", IndexEntry{})
	schemas.Register("index.Index", Index{})
	schemas.Register("index.GranularLineLength", GranularLineLength{})
	schemas.Register("index.LineLength", LineLength{})
}
