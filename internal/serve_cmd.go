package viz

import (
	"context"
	"github.com/chromy/viz/internal/cache"
	"github.com/chromy/viz/internal/core"
	"github.com/chromy/viz/internal/features/repo"
	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/http"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
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
		log.Printf("initial repo resolution failed: %v", err)
	}
	if _, err := repo.ResolveRepo(ctx, "gh:google:perfetto"); err != nil {
		log.Printf("initial repo resolution failed: %v", err)
	}
	//if _, err := repo.ResolveRepo(ctx, "gh:google:perfetto"); err != nil {
	//	log.Printf("Failed to add go-billy repo from github: %v", err)
	//}

	//if _, err := repo.ResolveRepo(ctx, "gh:githubtraining:hellogitworld"); err != nil {
	//	log.Printf("load initial repos: %v", err)
	//}
}

func DoServe(ctx context.Context, port uint, memcached string) {
	initSentry()

	// Initialize cache
	var cacheImpl cache.Cache
	if memcached != "" {
		log.Printf("using memcached at %s", memcached)
		servers := strings.Split(memcached, ",")
		cacheImpl = cache.NewMemcachedCache(servers...)
	} else {
		log.Printf("using in-memory cache")
		cacheImpl = cache.NewMemoryCache()
	}
	core.InitCache(cacheImpl)

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

	srv := &http.Server{
		Addr:           ":"+strconv.Itoa(int(port)),
		Handler:        sentryHandler.Handle(router),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
	}
	log.Printf("ready serve http://localhost:%d", port)
	log.Fatal(srv.ListenAndServe())
}

func initSentry() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		log.Println("SENTRY_DSN not set, sentry disabled")
		return
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      GetEnvironment(),
		TracesSampleRate: 1.0,
		Debug:            os.Getenv("SENTRY_DEBUG") == "true",
	})
	if err == nil {
		log.Println("sentry initialized")
	} else {
		log.Fatalf("sentry.Init: %s", err)
	}
}

func GetEnvironment() string {
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		return env
	}
	if env := os.Getenv("GO_ENV"); env != "" {
		return env
	}
	return "development"
}
