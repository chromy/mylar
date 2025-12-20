package viz

import (
	"context"
	"github.com/chromy/viz/internal/routes"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"html/template"
)

func LoadTemplates() map[string]*template.Template {
	tmpls := make(map[string]*template.Template)

	pagePaths, err := fs.Glob(templatesFS, "pages/*")
	if err != nil {
		panic(err)
	}
	for _, pagePath := range pagePaths {
		name := filepath.Base(pagePath)
		tmpl := template.New(name)
		tmpl = template.Must(tmpl.ParseFS(templatesFS, "base.html", pagePath))
		tmpls[name] = tmpl
	}

	return tmpls
}

func DoServe(ctx context.Context, port uint) {
	tmpls := LoadTemplates()

	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	routeIds := routes.List()
	for _, id := range routeIds {
		if route, found := routes.Get(id); found {
			mux.HandleFunc(route.Path, func(w http.ResponseWriter, r *http.Request) {
				route.Handler(tmpls, w, r)
			});
		}
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			indexHandler(tmpls["home.html"], w, r)
		} else {
			notFoundHandler(tmpls["404.html"], w, r)
		}
	})

	log.Printf("ready serve http://localhost:%d", port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(int(port)), mux))
}

func indexHandler(tmpl *template.Template, w http.ResponseWriter, r *http.Request) {
	err := tmpl.Execute(w, "Hello, world!")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func notFoundHandler(tmpl *template.Template, w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	err := tmpl.Execute(w, r.URL.Path)
	if err != nil {
		http.Error(w, "404 - Page Not Found", http.StatusNotFound)
	}
}
