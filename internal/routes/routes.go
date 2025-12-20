package routes

import (
	"fmt"
	"sync"
	"net/http"
	"html/template"
)

type Templates map[string]*template.Template

type Route struct {
	Id         string
	Path       string
	Handler    func(tmpls Templates, w http.ResponseWriter, r *http.Request)
}

var (
	mu       sync.RWMutex
	routes = make(map[string]Route)
)

func Register(route Route) {
	mu.Lock()
	defer mu.Unlock()

	if _, found := routes[route.Id]; found {
		panic(fmt.Sprintf("route already registered: %s", route.Id))
	}
	routes[route.Id] = route
}

func Get(id string) (Route, bool) {
	mu.RLock()
	defer mu.RUnlock()

	route, found := routes[id]
	return route, found
}

func List() []string {
	mu.RLock()
	defer mu.RUnlock()

	list := make([]string, 0, len(routes))
	for id := range routes {
		list = append(list, id)
	}
	return list
}


