package viz

import (
	"context"
	"github.com/chromy/viz/internal/core"
	"github.com/chromy/viz/internal/features/repo"
	"github.com/getsentry/sentry-go"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"strconv"
)

import _ "net/http/pprof"

func DoServe(ctx context.Context, port uint) {
	router := httprouter.New()

	router.ServeFiles("/static/*filepath", http.FS(staticFS))

	routeIds := core.ListRoutes()
	for _, id := range routeIds {
		if route, found := core.GetRoute(id); found {
			router.Handle(route.Method, route.Path, withSentry(route.Handler))
		}
	}

	repo.AddFromPath(ctx, "self", ".")
	repo.AddFromPath(ctx, "perfetto", "/Users/chromy/src/perfetto")

	router.Handler(http.MethodGet, "/debug/pprof/*item", http.DefaultServeMux)

	log.Printf("ready serve http://localhost:%d", port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(int(port)), router))
}

func withSentry(handler httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		hub := sentry.GetHubFromContext(r.Context())
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
			r = r.WithContext(sentry.SetHubOnContext(r.Context(), hub))
		}

		hub.Scope().SetTag("path", r.URL.Path)
		hub.Scope().SetTag("method", r.Method)

		defer func() {
			if err := recover(); err != nil {
				hub.CaptureException(err.(error))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		handler(w, r, ps)
	}
}
