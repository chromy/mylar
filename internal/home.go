package viz

import (
	"github.com/chromy/viz/internal/core"
	"github.com/julienschmidt/httprouter"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"sync"
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

	core.RegisterRoute(core.Route{
		Id:      "home.index",
		Method:  http.MethodGet,
		Path:    "/",
		Handler: Home,
	})

	core.RegisterRoute(core.Route{
		Id:      "home.others",
		Method:  http.MethodGet,
		Path:    "/app/*path",
		Handler: Home,
	})
}
