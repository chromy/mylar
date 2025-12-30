package viz

import (
	"context"
	"github.com/chromy/viz/internal/core"
	"github.com/chromy/viz/internal/features/repo"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"strconv"
)

func DoServe(ctx context.Context, port uint) {
	router := httprouter.New()

	router.ServeFiles("/static/*filepath", http.FS(staticFS))

	routeIds := core.ListRoutes()
	for _, id := range routeIds {
		if route, found := core.GetRoute(id); found {
			router.Handle(route.Method, route.Path, route.Handler)
		}
	}

	repo.AddFromPath(ctx, "self", ".")
	repo.AddFromPath(ctx, "perfetto", "/Users/chromy/src/perfetto")

	log.Printf("ready serve http://localhost:%d", port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(int(port)), router))
}
