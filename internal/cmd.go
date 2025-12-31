package viz

import (
	"context"
	"embed"
	"flag"
	"fmt"
	_ "github.com/chromy/viz/internal/features"
	"github.com/chromy/viz/internal/schemas"
	"github.com/getsentry/sentry-go"
	"io/fs"
	"log"
	"os"
	"time"
)

//go:embed static/*
//go:embed templates/*
var assetsFS embed.FS
var staticFS, _ = fs.Sub(assetsFS, "static")
var templatesFS, _ = fs.Sub(assetsFS, "templates")

func Usage() {
	fmt.Fprintf(os.Stderr, "viz <subcommand>\n")
}

func Cmd() {
	if len(os.Args) < 2 {
		Usage()
		os.Exit(1)
	}

	initSentry()
	defer sentry.Flush(2 * time.Second)

	ctx := context.Background()

	serve := func(args []string) int {
		fs := flag.NewFlagSet("serve", flag.ExitOnError)
		port := fs.Uint("port", 8080, "port to listen on")
		if err := fs.Parse(args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}
		DoServe(ctx, *port)
		return 0
	}

	assets := func(args []string) int {
		fs := flag.NewFlagSet("assets", flag.ExitOnError)
		if err := fs.Parse(args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}
		DoAssets(ctx)
		return 0
	}

	dev := func(args []string) int {
		fs := flag.NewFlagSet("dev", flag.ExitOnError)
		port := fs.Uint("port", 8080, "port to listen on")
		if err := fs.Parse(args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}
		DoDev(ctx, *port)
		return 0
	}

	schemasCmd := func(args []string) int {
		fs := flag.NewFlagSet("schemas", flag.ExitOnError)
		if err := fs.Parse(args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}
		DoSchemas(ctx)
		return 0
	}

	main := func(args []string) int {
		cmd := args[1]
		subArgs := args[2:]
		switch cmd {
		case "serve":
			return serve(subArgs)
		case "assets":
			return assets(subArgs)
		case "dev":
			return dev(subArgs)
		case "schemas":
			return schemasCmd(subArgs)
		default:
			fmt.Fprintf(os.Stderr, "Unknown subcommand '%s'\n", cmd)
			Usage()
			return 1
		}
	}

	os.Exit(main(os.Args))
}

func DoAssets(ctx context.Context) {
	fs.WalkDir(assetsFS, ".", func(path string, d fs.DirEntry, err error) error {
		fmt.Println(path)
		return nil
	})
}

func DoSchemas(ctx context.Context) {
	fmt.Printf("%s", schemas.ToZodSchema())
}

func initSentry() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		log.Println("SENTRY_DSN not set, Sentry disabled")
		return
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      getEnvironment(),
		TracesSampleRate: 1.0,
		Debug:            os.Getenv("SENTRY_DEBUG") == "true",
	})
	if err != nil {
		log.Printf("sentry.Init: %s", err)
	} else {
		log.Println("Sentry initialized")
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
