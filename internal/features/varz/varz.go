package varz

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/chromy/viz/internal/core"
	"github.com/chromy/viz/internal/schemas"
	"github.com/julienschmidt/httprouter"
)

type VarzResponse struct {
	Version   string    `json:"version"`
	BuildTime string    `json:"build_time"`
	GoVersion string    `json:"go_version"`
	StartTime time.Time `json:"start_time"`
	Uptime    string    `json:"uptime"`
}

var (
	version   = "dev"
	buildTime = "unknown"
	startTime = time.Now()
)

func VarzHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	uptime := time.Since(startTime)
	
	response := VarzResponse{
		Version:   version,
		BuildTime: buildTime,
		GoVersion: runtime.Version(),
		StartTime: startTime,
		Uptime:    uptime.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func init() {
	core.RegisterRoute(core.Route{
		Id:      "varz",
		Method:  http.MethodGet,
		Path:    "/api/varz",
		Handler: VarzHandler,
	})

	schemas.Register("varz.VarzResponse", VarzResponse{})
}