package index

import (
	"github.com/chromy/viz/internal/routes"
	"github.com/chromy/viz/internal/features/archive"
	"net/http"
	"io/fs"
	"sync"
	"encoding/json"
)

var (
	mu   sync.RWMutex
	fileCount int64
)

type IndexStatus struct {
	Message 	string         `json:"message"`
	FileCount int64  `json:"fileCount"`
}

func BuildIndex(root fs.FS) {
	fs.WalkDir(root, ".", func (path string, entry fs.DirEntry, err error) error {
		if entry.IsDir() {
			mu.Lock()
			defer mu.Unlock()
			fileCount += 1
		}
		return nil
	})
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

				mu.Lock()
				defer mu.Unlock()
				status := IndexStatus{}
				status.Message = "Loading..."
				status.FileCount = fileCount

				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(status); err != nil {
					http.Error(w, "failed to encode response", http.StatusInternalServerError)
				}
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		},
	})
}

