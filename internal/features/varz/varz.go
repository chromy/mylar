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

type MemoryStats struct {
	Alloc        uint64 `json:"alloc"`         // bytes allocated and not yet freed
	TotalAlloc   uint64 `json:"total_alloc"`   // bytes allocated (even if freed)
	Sys          uint64 `json:"sys"`           // bytes obtained from system
	NumGC        uint32 `json:"num_gc"`        // number of garbage collections
	HeapAlloc    uint64 `json:"heap_alloc"`    // bytes allocated and not yet freed (same as Alloc)
	HeapSys      uint64 `json:"heap_sys"`      // bytes obtained from system
	HeapInuse    uint64 `json:"heap_inuse"`    // bytes in in-use spans
	HeapReleased uint64 `json:"heap_released"` // bytes released to the OS
	StackInuse   uint64 `json:"stack_inuse"`   // bytes in stack spans
	StackSys     uint64 `json:"stack_sys"`     // bytes in stack spans
}

type VarzResponse struct {
	Version   string      `json:"version"`
	BuildTime string      `json:"build_time"`
	GoVersion string      `json:"go_version"`
	StartTime time.Time   `json:"start_time"`
	Uptime    string      `json:"uptime"`
	Memory    MemoryStats `json:"memory"`
}

var (
	buildTime = "unknown"
	startTime = time.Now()
)

func VarzHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	uptime := time.Since(startTime)

	// Collect memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	memory := MemoryStats{
		Alloc:        memStats.Alloc,
		TotalAlloc:   memStats.TotalAlloc,
		Sys:          memStats.Sys,
		NumGC:        memStats.NumGC,
		HeapAlloc:    memStats.HeapAlloc,
		HeapSys:      memStats.HeapSys,
		HeapInuse:    memStats.HeapInuse,
		HeapReleased: memStats.HeapReleased,
		StackInuse:   memStats.StackInuse,
		StackSys:     memStats.StackSys,
	}

	response := VarzResponse{
		Version:   core.GetVersion(),
		BuildTime: buildTime,
		GoVersion: runtime.Version(),
		StartTime: startTime,
		Uptime:    uptime.String(),
		Memory:    memory,
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
