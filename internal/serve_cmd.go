package viz

import (
	"context"
	"github.com/chromy/viz/internal/routes"
	"github.com/chromy/viz/internal/features/repo"
	"log"
	"net/http"
	"strconv"
	"github.com/julienschmidt/httprouter"
)


func DoServe(ctx context.Context, port uint) {
	router := httprouter.New()

	router.ServeFiles("/static/*filepath", http.FS(staticFS))

	routeIds := routes.List()
	for _, id := range routeIds {
		if route, found := routes.Get(id); found {
			router.Handle(route.Method, route.Path, route.Handler);
		}
	}

	repo.AddFromPath("self", ".")

	log.Printf("ready serve http://localhost:%d", port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(int(port)), router))
}
