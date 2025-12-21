package archive

import (
	"github.com/chromy/viz/internal/routes"
	"net/http"
	"io/fs"
	"os"
	"sync"
	"encoding/json"
)

var (
	mu   sync.RWMutex
	root fs.FS
)

func GetFS() fs.FS {
	mu.RLock()
	defer mu.RUnlock()
	return root
}

type FileMetadata struct {
	Path     string         `json:"path"`
	Name     string         `json:"name"`
	Size     int64          `json:"size"`
	IsDir    bool           `json:"isDir"`
	Children []string       `json:"children,omitempty"`
}

func init() {
	mu.Lock()
	defer mu.Unlock()

	root = os.DirFS(".")

	routes.Register(routes.Route{
		Id: "archive.get",
		Path: "/api/archive/get",
		Handler: func(tmpls routes.Templates, w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				path := r.URL.Query().Get("path")
				if path == "" {
					http.Error(w, "path parameter is required", http.StatusBadRequest)
					return
				}
				
				fsRoot := GetFS()
				info, err := fs.Stat(fsRoot, path)
				if err != nil {
					http.Error(w, "file not found", http.StatusNotFound)
					return
				}
				
				metadata := FileMetadata{
					Path:    path,
					Name:    info.Name(),
					Size:    info.Size(),
					IsDir:   info.IsDir(),
				}
				
				if info.IsDir() {
					entries, err := fs.ReadDir(fsRoot, path)
					if err == nil {
						children := make([]string, 0, len(entries))
						for _, entry := range entries {
							children = append(children, entry.Name())
						}
						metadata.Children = children
					}
				}
				
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(metadata); err != nil {
					http.Error(w, "failed to encode response", http.StatusInternalServerError)
				}
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		},
	})
}
