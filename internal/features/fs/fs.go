package raw

import (
	"github.com/chromy/viz/internal/routes"
	"net/http"
	"io/fs"
	"os"
	"sync"
	"fmt"
)

var (
	mu   sync.RWMutex
	root fs.FS
)

func init() {
	mu.Lock()
	defer mu.Unlock()

	root = os.DirFS(".")

	routes.Register(routes.Route{
		Id: "raw.get",
		Path: "/api/fs/get",
		Handler: func(tmpls routes.Templates, w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				path := r.URL.Query().Get("path")
				if path == "" {
					http.Error(w, "Method not allowed", http.StatusBadRequest)
				} else {
					fmt.Fprintln(w, path)
				}
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		},
	})
}
