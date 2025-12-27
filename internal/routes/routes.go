package routes

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"sync"
)

type Route struct {
	Id      string
	Path    string
	Method  string
	Handler func(w http.ResponseWriter, r *http.Request, ps httprouter.Params)
}

type State struct {
	Routes map[string]Route
}

var (
	mu    sync.RWMutex
	state State
)

func isValidMethod(method string) bool {
	switch method {
	case http.MethodGet:
		return true
	case http.MethodHead:
		return true
	case http.MethodPost:
		return true
	case http.MethodPut:
		return true
	default:
		return false
	}
}

func Register(route Route) {
	mu.Lock()
	defer mu.Unlock()

	if !isValidMethod(route.Method) {
		panic(fmt.Sprintf("invalid HTTP method %s", route.Method))
	}

	if _, found := state.Routes[route.Id]; found {
		panic(fmt.Sprintf("route already registered %s", route.Id))
	}
	state.Routes[route.Id] = route
}

func Get(id string) (Route, bool) {
	mu.RLock()
	defer mu.RUnlock()

	route, found := state.Routes[id]
	return route, found
}

func List() []string {
	mu.RLock()
	defer mu.RUnlock()

	list := make([]string, 0, len(state.Routes))
	for id := range state.Routes {
		list = append(list, id)
	}
	return list
}

func init() {
	mu.Lock()
	defer mu.Unlock()
	state = State{}
	state.Routes = make(map[string]Route)
}
