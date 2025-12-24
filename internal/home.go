package viz

import (
	"html/template"
	"sync"
	"io/fs"
	"github.com/chromy/viz/internal/routes"
	"path/filepath"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type State struct {
	Templates map[string]*template.Template
}

var (
	mu    sync.RWMutex
	state State
)

func Home(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	mu.RLock()
	defer mu.RUnlock()
	t := state.Templates["home.html"]

	err := t.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func init() {
	mu.Lock()
	defer mu.Unlock()
	state = State{}
	state.Templates = make(map[string]*template.Template)

	pagePaths, err := fs.Glob(templatesFS, "pages/*")
	if err != nil {
		panic(err)
	}

	for _, pagePath := range pagePaths {
		name := filepath.Base(pagePath)
		t := template.New(name)
		t = template.Must(template.ParseFS(templatesFS, "base.html", pagePath))
		state.Templates[name] = t
	}

	routes.Register(routes.Route{
		Id: "home",
		Method: http.MethodGet,
		Path: "/",
		Handler: Home,
	})
}

