package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/chromy/viz/internal/core"
	"github.com/chromy/viz/internal/features/repo"
	"github.com/chromy/viz/internal/schemas"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/julienschmidt/httprouter"
)

type TileMetadata struct {
	X   int64 `json:"x"`
	Y   int64 `json:"y"`
	Lod int64 `json:"lod"`
}

func ComputeHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	repoId := ps.ByName("repoId")
	rawHash := ps.ByName("hash")
	computationId := ps.ByName("computationId")

	computation, found := core.GetBlobComputation(computationId)
	if !found {
		http.Error(w, fmt.Sprintf("Computation '%s' unknown", computationId), http.StatusNotFound)
		return
	}

	if !plumbing.IsHash(rawHash) {
		http.Error(w, fmt.Sprintf("Could not parse hash '%s'", rawHash), http.StatusNotFound)
		return
	}
	hash := plumbing.NewHash(rawHash)

	result, err := computation.Execute(r.Context(), repoId, hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func TileHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	repoName := ps.ByName("repoId")
	if repoName == "" {
		http.Error(w, "repo must be set", http.StatusBadRequest)
		return
	}

	committish := ps.ByName("committish")
	if committish == "" {
		http.Error(w, "committish must be set", http.StatusBadRequest)
		return
	}

	rawX := ps.ByName("x")
	if rawX == "" {
		http.Error(w, "x must be set", http.StatusBadRequest)
		return
	}
	x, err := strconv.ParseInt(rawX, 10, 64)
	if err != nil {
		http.Error(w, "x must be number", http.StatusBadRequest)
		return
	}

	rawY := ps.ByName("y")
	if rawY == "" {
		http.Error(w, "y must be set", http.StatusBadRequest)
		return
	}
	y, err := strconv.ParseInt(rawY, 10, 64)
	if err != nil {
		http.Error(w, "y must be number", http.StatusBadRequest)
		return
	}

	rawLod := ps.ByName("lod")
	if rawLod == "" {
		http.Error(w, "lod must be set", http.StatusBadRequest)
		return
	}
	lod, err := strconv.ParseInt(rawLod, 10, 64)
	if err != nil {
		http.Error(w, "lod must be number", http.StatusBadRequest)
		return
	}

	tileComputationId := ps.ByName("tileComputationId")

	tileComputation, found := core.GetTileComputation(tileComputationId)
	if !found {
		http.Error(w, fmt.Sprintf("tile computation %s not found", tileComputationId), http.StatusNotFound)
		return
	}

	repository, err := repo.ResolveRepo(r.Context(), repoName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hash, err := repo.ResolveCommittishToTreeish(repository, committish)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tile, err := tileComputation.Execute(r.Context(), repoName, hash, lod, x, y)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	metadata := TileMetadata{
		X:   x,
		Y:   y,
		Lod: lod,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(tile); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

}

func init() {

	core.RegisterRoute(core.Route{
		Id:      "api.compute",
		Method:  http.MethodGet,
		Path:    "/api/compute/:computationId/:repoId/:hash",
		Handler: ComputeHandler,
	})

	core.RegisterRoute(core.Route{
		Id:      "api.tile",
		Method:  http.MethodGet,
		Path:    "/api/tile/:tileComputationId/:repoId/:committish/:lod/:x/:y",
		Handler: TileHandler,
	})

	schemas.Register("api.TileMetadata", TileMetadata{})
}
