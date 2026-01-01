package viz

import (
	"context"
	"github.com/chromy/viz/internal/core"
	"github.com/chromy/viz/internal/features/repo"
	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/http"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"os"
	"strconv"
)

import _ "net/http/pprof"

func loadInitialRepos(ctx context.Context) {
	//if err := repo.AddFromGithub(ctx, "google", "perfetto"); err != nil {
	//	log.Printf("Failed to add perfetto repo from github: %v", err)
	//}
	//if err := repo.AddFromGithub(ctx, "getsentry", "sentry"); err != nil {
	//	log.Printf("Failed to add sentry repo from github: %v", err)
	//}
	//if err := repo.AddFromPath(ctx, "path:self", "."); err != nil {
	//	log.Printf("Failed to add self repo: %v", err)
	//}
	//if err := repo.AddFromPath(ctx, "path:perfetto", "/Users/chromy/src/perfetto"); err != nil {
	//	log.Printf("Failed to add perfetto repo from path: %v", err)
	//}

	if _, err := repo.ResolveRepo(ctx, "gh:go-git:go-billy"); err != nil {
		log.Printf("Failed to add go-billy repo from github: %v", err)
	}

	//if _, err := repo.ResolveRepo(ctx, "gh:githubtraining:hellogitworld"); err != nil {
	//	log.Printf("load initial repos: %v", err)
	//}
}

func DoServe(ctx context.Context, port uint, memcachedUrl string) {
	initSentry()

	if memcachedUrl != "" {
		log.Printf("using memcached at %s", memcachedUrl)
	}

	router := httprouter.New()

	router.ServeFiles("/static/*filepath", http.FS(staticFS))

	routeIds := core.ListRoutes()
	for _, id := range routeIds {
		if route, found := core.GetRoute(id); found {
			router.Handle(route.Method, route.Path, route.Handler)
		}
	}

	go loadInitialRepos(ctx)

	router.Handler(http.MethodGet, "/debug/pprof/*item", http.DefaultServeMux)

	sentryHandler := sentryhttp.New(sentryhttp.Options{})

	log.Printf("ready serve http://localhost:%d", port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(int(port)), sentryHandler.Handle(router)))
}

func initSentry() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		log.Println("SENTRY_DSN not set, sentry disabled")
		return
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      getEnvironment(),
		TracesSampleRate: 1.0,
		Debug:            os.Getenv("SENTRY_DEBUG") == "true",
	})
	if err == nil {
		log.Println("sentry initialized")
	} else {
		log.Fatalf("sentry.Init: %s", err)
	}
}

func getEnvironment() string {
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		return env
	}
	if env := os.Getenv("GO_ENV"); env != "" {
		return env
	}
	return "development"
}
