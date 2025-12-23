package index

import (
	"github.com/chromy/viz/internal/routes"
	"github.com/chromy/viz/internal/features/archive"
	"net/http"
	"io/fs"
	"sync"
	"encoding/json"
	"path/filepath"
	"log"
)

type IndexStatus int

const (
	WalkingDirectory IndexStatus = iota
	MeasuringFiles
	Ready
)

type IndexFileEntry struct {
	Path string `json:"path"`
	Name string `json:"name"`
	SizeBytes int64 `json:"sizeBytes"`
	OffsetBytes int64 `json:"offsetBytes"`
	Index int `json:"index"`
}

type IndexStatusResponse struct {
	Message   string      `json:"message"`
	FileCount int64       `json:"fileCount"`
	Status    IndexStatus `json:"status"`
}

type IndexEntriesResponse struct {
	Entries []IndexFileEntry `json:"entries"`
}

var (
	mu   sync.RWMutex
	fileCount int64
	totalBytes int64
	entries []IndexFileEntry
	status IndexStatus
)

func SetStatus(next IndexStatus) {
	mu.Lock()
	defer mu.Unlock()
	status = next
}

func BuildIndex(root fs.FS) {

	SetStatus(WalkingDirectory)

	var wg sync.WaitGroup

	fs.WalkDir(root, ".", func (path string, entry fs.DirEntry, err error) error {
		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		if entry.IsDir() && entry.Name() == "node_modules" {
			return filepath.SkipDir
		}

		if !entry.IsDir() {
			mu.Lock()
			defer mu.Unlock()
			fileCount += 1

			index := len(entries)

			indexEntry := IndexFileEntry{}
			indexEntry.Path = path
			indexEntry.Name = entry.Name()
			indexEntry.Index = index

			entries = append(entries, indexEntry)
			log.Printf("%v", len(entries))

			wg.Go(func() {
				info, err := fs.Stat(root, path)
				if err != nil {
					return
				}

				sizeBytes := info.Size()

				mu.Lock()
				defer mu.Unlock()
				entries[index].SizeBytes = sizeBytes
			})
		}
		return nil
	})

	SetStatus(MeasuringFiles)

	wg.Wait()

	mu.Lock()
	var offsetBytes int64
	for i := range entries {
		entries[i].OffsetBytes = offsetBytes
		offsetBytes += entries[i].SizeBytes
	}
	totalBytes = offsetBytes
	mu.Unlock()


	SetStatus(Ready)
}


func init() {
	root := archive.GetFS()
	go BuildIndex(root)

	routes.Register(routes.Route{
		Id: "index.status",
		Path: "/api/index/status",
		Handler: func(tmpls routes.Templates, w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:

				mu.RLock()
				defer mu.RUnlock()
				resp := IndexStatusResponse{}
				resp.Message = "Loading..."
				resp.FileCount = fileCount
				resp.Status = status

				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(resp); err != nil {
					http.Error(w, "failed to encode response", http.StatusInternalServerError)
				}
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		},
	})

	routes.Register(routes.Route{
		Id: "index.entries",
		Path: "/api/index/entries",
		Handler: func(tmpls routes.Templates, w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				mu.RLock()
				defer mu.RUnlock()
				resp := IndexEntriesResponse{}
				resp.Entries = entries;
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(resp); err != nil {
					http.Error(w, "failed to encode response", http.StatusInternalServerError)
				}
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		},
	})

	routes.Register(routes.Route{
		Id: "index.foo",
		Path: "/api/index/foo",
		Handler: func(tmpls routes.Templates, w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				mu.RLock()
				defer mu.RUnlock()
				resp := IndexEntriesResponse{}
				resp.Entries = entries;
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(resp); err != nil {
					http.Error(w, "failed to encode response", http.StatusInternalServerError)
				}
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		},
	})

}

