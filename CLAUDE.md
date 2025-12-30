# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is "viz" - a code visualization tool that serves Git repositories with an interactive file browser and viewer. The application consists of a Go backend that serves static files and provides Git repository APIs, paired with a React/TypeScript frontend for interactive visualization.

## Development Commands

### Go Backend
- `go run ./cmd/viz serve --port 8080` - Start production server
- `go run ./cmd/viz dev --port 8080` - Start development server with assets rebuilding
- `go run ./cmd/viz assets` - List embedded assets
- `go test ./...` - Run all Go tests
- `go build ./cmd/viz` - Build the viz binary

### Frontend/Assets
- Frontend assets are built using esbuild and embedded into the Go binary via `//go:embed`

## Architecture

### Backend Structure
The Go backend uses a modular feature-based architecture:

- **Routes System**: `internal/routes/` - Thread-safe route registration system where features can register HTTP handlers with unique IDs
- **Features**: `internal/features/` - Self-contained modules that register their own routes:
  - `index/` - Git object indexing and line counting for files/trees
  - `repo/` - Repository management and filesystem navigation APIs  
  - `archive/` - (Import only, likely for git archive functionality)
- **Commands**: Three main subcommands in `internal/cmd.go`:
  - `serve` - Production server
  - `dev` - Development server with asset rebuilding
  - `assets` - Asset listing utility

### Frontend Structure
React SPA using wouter for routing:

- **Main App**: `js/main.tsx` - Root component with home page (repo list) and repo viewer routes
- **Viewer**: `js/viewer.tsx` - Interactive canvas-based visualization component using gl-matrix for 2D transformations
- **Camera**: `js/camera.tsx` - Camera system for pan/zoom functionality
- **Query System**: `js/query.tsx` - Data fetching with Zod schema validation

### Key APIs
- `GET /api/repo` - List available repositories
- `GET /api/fs/get` - Get filesystem information (used by viewer)
- Repository data flows through Zod schemas for type safety

### Asset Pipeline
- Frontend TypeScript/React code is bundled with esbuild
- CSS uses Tailwind CSS v4
- Assets are embedded into Go binary via `//go:embed static/* templates/*`
- Templates use Go's html/template system

## Development Notes

- Features register routes via `routes.Register()` in their init() functions
- Git operations use go-git library for repository access
- Frontend uses strict TypeScript configuration with `noUncheckedIndexedAccess` and `exactOptionalPropertyTypes`
- The viewer component renders to HTML5 Canvas with 2D transformations for code visualization
- ./tools/presubmit to run presubmit checks.

- Always run tools/test after changes.
- Remeber js files should be named with underscores not dashes.