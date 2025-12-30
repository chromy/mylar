package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/chromy/viz/internal/core"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/julienschmidt/httprouter"
)

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

func init() {
	core.RegisterRoute(core.Route{
		Id:      "api.compute",
		Method:  http.MethodGet,
		Path:    "/api/compute/:computationId/:repoId/:hash",
		Handler: ComputeHandler,
	})
}

