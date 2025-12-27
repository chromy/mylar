package viz

import (
	"context"
	"embed"
	"flag"
	"fmt"
	_ "github.com/chromy/viz/internal/features"
	"github.com/chromy/viz/internal/schemas"
	"io/fs"
	"os"
	"sort"
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
	allSchemas := schemas.GetAllSchemas()
	fmt.Printf("import { z } from \"zod\";\n\n")

	// Sort schema IDs for deterministic output
	var ids []string
	for id := range allSchemas {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		schema := allSchemas[id]
		fmt.Printf("// %s\n%s\n\n", id, schema)
	}
}
