package index

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromy/viz/internal/features/repo"
	"github.com/chromy/viz/internal/routes"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/julienschmidt/httprouter"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
)

//type IndexFileEntry struct {
//	Path string `json:"path"`
//	Name string `json:"name"`
//	SizeBytes int64 `json:"sizeBytes"`
//	OffsetBytes int64 `json:"offsetBytes"`
//	Index int `json:"index"`
//}
//
//type IndexStatusResponse struct {
//	Message   string      `json:"message"`
//	FileCount int64       `json:"fileCount"`
//	Status    IndexStatus `json:"status"`
//}

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
		log.Printf("is blob: %v\n", hash)
		return computeIndexForBlob(obj)
	case *object.Tree:
		log.Printf("is tree: %v\n", hash)
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

func countLines(reader io.Reader) (int64, error) {
	scanner := bufio.NewScanner(reader)
	var lineCount int64

	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	if lineCount == 0 {
		content, err := io.ReadAll(reader)
		if err != nil {
			return 0, err
		}
		if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
			lineCount = 1
		}
	}

	return lineCount, nil
}

func IndexHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	repoName := ps.ByName("repo")
	committish := ps.ByName("committish")
	path := ps.ByName("path")

	if repoName == "" {
		http.Error(w, "repo must be set", http.StatusBadRequest)
		return
	}

	if committish == "" {
		http.Error(w, "committish must be set", http.StatusBadRequest)
		return
	}

	if path == "" {
		http.Error(w, "path must be set", http.StatusBadRequest)
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

func init() {
	routes.Register(routes.Route{
		Id:      "index.get",
		Method:  http.MethodGet,
		Path:    "/api/repo/:repo/:committish/index/*path",
		Handler: IndexHandler,
	})
}
