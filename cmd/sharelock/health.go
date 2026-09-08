package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// readinessTimeout bounds the dependency check behind GET /health/ready, so a
// stalled database cannot hold a probe open for the whole request timeout.
const readinessTimeout = 2 * time.Second

// pinger reports whether a runtime dependency is reachable.
type pinger interface {
	PingContext(ctx context.Context) error
}

// healthRespBody is the payload returned by both probes.
type healthRespBody struct {
	Status string `json:"status"`
}

// health serves the liveness and readiness probes. They live outside the
// OpenAPI surface: orchestrators address them directly, and they are not part
// of the API contract consumed by the frontend.
type health struct {
	db pinger
}

// live reports that the process is running. It deliberately touches no
// dependency, so an unreachable database never triggers a restart.
func (h health) live(writer http.ResponseWriter, _ *http.Request) {
	writeHealth(writer, http.StatusOK, "ok")
}

// ready reports whether the process can serve traffic, which requires a
// reachable database.
func (h health) ready(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), readinessTimeout)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		slog.ErrorContext(ctx, "readiness probe failed", "err", err)
		writeHealth(writer, http.StatusServiceUnavailable, "unavailable")
		return
	}

	writeHealth(writer, http.StatusOK, "ok")
}

func writeHealth(writer http.ResponseWriter, status int, state string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(healthRespBody{Status: state}); err != nil {
		slog.Error("write health response", "err", err)
	}
}
