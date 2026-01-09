package viz

import (
	"github.com/chromy/mylar/internal/core"
	"github.com/julienschmidt/httprouter"
	"html/template"
	"io/fs"
	"net/http"
	"os"
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

type TemplateData struct {
	SentryDSN   string
	Environment string
	Version     string
}

func Home(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	mu.RLock()
	defer mu.RUnlock()
	t := state.Templates["home.html"]

	data := TemplateData{
		SentryDSN:   os.Getenv("SENTRY_FRONTEND_DSN"),
		Environment: GetEnvironment(),
		Version:     core.GetVersion(),
	}

	err := t.Execute(w, data)
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
