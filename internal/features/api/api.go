package api

import (
	"github.com/chromy/viz/internal/routes"
	"net/http"
)

func init() {
	routes.Register(routes.Route{
		Id: "sku.show",
		Path: "/api",
		Handler: func(tmpls routes.Templates, w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		},
	})
}
