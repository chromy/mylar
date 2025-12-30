package core

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type Route struct {
	Id      string
	Path    string
	Method  string
	Handler func(w http.ResponseWriter, r *http.Request, ps httprouter.Params)
}

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

func RegisterRoute(route Route) {
	mu.Lock()
	defer mu.Unlock()

	if !isValidMethod(route.Method) {
		panic(fmt.Sprintf("invalid HTTP method %s", route.Method))
	}

	if _, found := routes[route.Id]; found {
		panic(fmt.Sprintf("route already registered %s", route.Id))
	}
	routes[route.Id] = route
}


func GetRoute(id string) (Route, bool) {
	mu.RLock()
	defer mu.RUnlock()

	route, found := routes[id]
	return route, found
}

func ListRoutes() []string {
	mu.RLock()
	defer mu.RUnlock()

	list := make([]string, 0, len(routes))
	for id := range routes {
		list = append(list, id)
	}
	return list
}
