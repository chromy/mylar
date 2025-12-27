package index

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/julienschmidt/httprouter"
	"github.com/chromy/viz/internal/routes"
	"github.com/chromy/viz/internal/features/repo"
)

type IndexEntry struct {
	Path string
	LineOffset int64
	LineCount int64
}

type Index struct {
	Entries []IndexEntry
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
		return &Index{Entries: []IndexEntry{}}, nil
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
	commitish := ps.ByName("commitish")
	path := ps.ByName("path")

	if repoName == "" {
		http.Error(w, "repo must be set", http.StatusBadRequest)
		return
	}

	if commitish == "" {
		http.Error(w, "commitish must be set", http.StatusBadRequest)
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

	hash, err := repo.ResolveCommitish(repository, commitish)
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
		Id: "index.get",
		Method: http.MethodGet,
		Path: "/api/repo/:repo/:commitish/index/*path",
		Handler: IndexHandler,
	})
}

//import (
//	"github.com/chromy/viz/internal/routes"
//	"github.com/chromy/viz/internal/features/archive"
//	"net/http"
//	"io/fs"
//	"sync"
//	"encoding/json"
//	"path/filepath"
//	"log"
//)
//
//type IndexStatus int
//
//const (
//	WalkingDirectory IndexStatus = iota
//	MeasuringFiles
//	Ready
//)
//
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
//
//type IndexEntriesResponse struct {
//	Entries []IndexFileEntry `json:"entries"`
//}
//
//var (
//	mu   sync.RWMutex
//	fileCount int64
//	totalBytes int64
//	entries []IndexFileEntry
//	status IndexStatus
//)
//
//func SetStatus(next IndexStatus) {
//	mu.Lock()
//	defer mu.Unlock()
//	status = next
//}
//
//func BuildIndex(root fs.FS) {
//
//	SetStatus(WalkingDirectory)
//
//	var wg sync.WaitGroup
//
//	fs.WalkDir(root, ".", func (path string, entry fs.DirEntry, err error) error {
//		if entry.IsDir() && entry.Name() == ".git" {
//			return filepath.SkipDir
//		}
//		if entry.IsDir() && entry.Name() == "node_modules" {
//			return filepath.SkipDir
//		}
//
//		if !entry.IsDir() {
//			mu.Lock()
//			defer mu.Unlock()
//			fileCount += 1
//
//			index := len(entries)
//
//			indexEntry := IndexFileEntry{}
//			indexEntry.Path = path
//			indexEntry.Name = entry.Name()
//			indexEntry.Index = index
//
//			entries = append(entries, indexEntry)
//			log.Printf("%v", len(entries))
//
//			wg.Go(func() {
//				info, err := fs.Stat(root, path)
//				if err != nil {
//					return
//				}
//
//				sizeBytes := info.Size()
//
//				mu.Lock()
//				defer mu.Unlock()
//				entries[index].SizeBytes = sizeBytes
//			})
//		}
//		return nil
//	})
//
//	SetStatus(MeasuringFiles)
//
//	wg.Wait()
//
//	mu.Lock()
//	var offsetBytes int64
//	for i := range entries {
//		entries[i].OffsetBytes = offsetBytes
//		offsetBytes += entries[i].SizeBytes
//	}
//	totalBytes = offsetBytes
//	mu.Unlock()
//
//
//	SetStatus(Ready)
//}
//
//
//func init() {
//	root := archive.GetFS()
//	go BuildIndex(root)
//
//	routes.Register(routes.Route{
//		Id: "index.status",
//		Path: "/api/index/status",
//		Handler: func(tmpls routes.Templates, w http.ResponseWriter, r *http.Request) {
//			switch r.Method {
//			case http.MethodGet:
//
//				mu.RLock()
//				defer mu.RUnlock()
//				resp := IndexStatusResponse{}
//				resp.Message = "Loading..."
//				resp.FileCount = fileCount
//				resp.Status = status
//
//				w.Header().Set("Content-Type", "application/json")
//				if err := json.NewEncoder(w).Encode(resp); err != nil {
//					http.Error(w, "failed to encode response", http.StatusInternalServerError)
//				}
//			default:
//				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
//			}
//		},
//	})
//
//	routes.Register(routes.Route{
//		Id: "index.entries",
//		Path: "/api/index/entries",
//		Handler: func(tmpls routes.Templates, w http.ResponseWriter, r *http.Request) {
//			switch r.Method {
//			case http.MethodGet:
//				mu.RLock()
//				defer mu.RUnlock()
//				resp := IndexEntriesResponse{}
//				resp.Entries = entries;
//				w.Header().Set("Content-Type", "application/json")
//				if err := json.NewEncoder(w).Encode(resp); err != nil {
//					http.Error(w, "failed to encode response", http.StatusInternalServerError)
//				}
//			default:
//				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
//			}
//		},
//	})
//
//	routes.Register(routes.Route{
//		Id: "index.foo",
//		Path: "/api/index/foo",
//		Handler: func(tmpls routes.Templates, w http.ResponseWriter, r *http.Request) {
//			switch r.Method {
//			case http.MethodGet:
//				mu.RLock()
//				defer mu.RUnlock()
//				resp := IndexEntriesResponse{}
//				resp.Entries = entries;
//				w.Header().Set("Content-Type", "application/json")
//				if err := json.NewEncoder(w).Encode(resp); err != nil {
//					http.Error(w, "failed to encode response", http.StatusInternalServerError)
//				}
//			default:
//				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
//			}
//		},
//	})
//
//}

