package handlers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

var startTime = time.Now()

// Health handles GET /health.
func Health(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"uptime":  time.Since(startTime).String(),
		"version": "1.0.0",
		"runtime": map[string]any{
			"goroutines": runtime.NumGoroutine(),
			"heapMB":     memStats.HeapAlloc / 1024 / 1024,
		},
	})
}
