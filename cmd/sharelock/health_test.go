package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/demeero/sharelock/internal/config"

	_ "modernc.org/sqlite"
)

type stubPinger struct {
	err error
}

func (s stubPinger) PingContext(context.Context) error {
	return s.err
}

func TestServer_HealthProbes_DatabaseReachable(t *testing.T) {
	server := newHealthTestServer(t, openHealthTestDB(t))

	for path, want := range map[string]string{"/health/live": "ok", "/health/ready": "ok"} {
		response := probeHealth(t, server, path)
		require.Equalf(t, http.StatusOK, response.Code, "GET %s body = %s", path, response.Body.String())
		require.Equal(t, want, decodeHealthStatus(t, response))
	}
}

func TestServer_HealthProbes_DatabaseUnreachable(t *testing.T) {
	server := newHealthTestServer(t, stubPinger{err: errors.New("database is gone")})

	// Liveness must stay green so an unreachable database does not restart a
	// healthy process; only readiness pulls the instance out of rotation.
	live := probeHealth(t, server, "/health/live")
	require.Equalf(t, http.StatusOK, live.Code, "GET /health/live body = %s", live.Body.String())
	require.Equal(t, "ok", decodeHealthStatus(t, live))

	ready := probeHealth(t, server, "/health/ready")
	require.Equalf(t, http.StatusServiceUnavailable, ready.Code, "GET /health/ready body = %s", ready.Body.String())
	require.Equal(t, "unavailable", decodeHealthStatus(t, ready))
}

func newHealthTestServer(t *testing.T, db pinger) *Server {
	t.Helper()
	cfg := config.Config{
		HTTP:  config.HTTPConfig{Addr: ":0", ReadHeaderTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second},
		Share: config.ShareConfig{MaxEncryptedBytes: 1024, MaxTTL: 24 * time.Hour, IdentifierSize: 32, VacuumInterval: time.Hour},
	}
	return NewServer(http.NewServeMux(), cfg, db)
}

func openHealthTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "sharelock.db"))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func probeHealth(t *testing.T, server *Server, path string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	server.handler.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody))
	return response
}

func decodeHealthStatus(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var body healthRespBody
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	return body.Status
}
