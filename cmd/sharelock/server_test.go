package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/stretchr/testify/require"

	"github.com/demeero/sharelock/internal/config"
	"github.com/demeero/sharelock/internal/share"
	"github.com/demeero/sharelock/migration"

	_ "modernc.org/sqlite"
)

func TestShareHTTPFlow(t *testing.T) {
	t.Parallel()
	cfg := config.Config{
		HTTP:  config.HTTPConfig{Addr: ":0", ReadHeaderTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second},
		DB:    config.DBConfig{Path: filepath.Join(t.TempDir(), "sharelock.db"), MaxOpenConns: 1, MaxIdleConns: 1, StartupTimeout: time.Second},
		Share: config.ShareConfig{MaxEncryptedBytes: 1024, MaxTTL: 24 * time.Hour, IdentifierSize: 32, VacuumInterval: time.Hour},
	}
	db, err := sql.Open("sqlite", cfg.DB.Path)
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, migration.Migrate(t.Context(), db))

	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("test", "1.0.0"))
	share.New(cfg.Share, huma.NewGroup(api, "/api/v1/shares"), db)
	server := NewServer(mux, cfg)

	for _, path := range []string{"/", "/s/" + strings.Repeat("a", 43)} {
		response := httptest.NewRecorder()
		server.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, http.NoBody))
		require.Equalf(t, http.StatusOK, response.Code, "GET %s body = %s", path, response.Body.String())
		require.Containsf(t, response.Body.String(), "id=\"app\"", "GET %s does not serve the frontend application", path)
	}
	openAPIDocument := httptest.NewRecorder()
	server.handler.ServeHTTP(openAPIDocument, httptest.NewRequest(http.MethodGet, "/openapi.json", http.NoBody))
	require.Equalf(t, http.StatusOK, openAPIDocument.Code, "GET /openapi.json body = %s", openAPIDocument.Body.String())
	for _, operationID := range []string{"createShare", "openShare", "revokeShare"} {
		require.Contains(t, openAPIDocument.Body.String(), "\"operationId\":\""+operationID+"\"")
	}

	created := createTestShare(t, server, true)
	openRequest := httptest.NewRequest(http.MethodPost, "/api/v1/shares/"+created.ID+"/open", http.NoBody)
	openResponse := httptest.NewRecorder()
	server.handler.ServeHTTP(openResponse, openRequest)
	require.Equalf(t, http.StatusOK, openResponse.Code, "first open body = %s", openResponse.Body.String())
	var opened struct {
		Envelope string `json:"envelope"`
		Burned   bool   `json:"burned"`
	}
	require.NoError(t, json.NewDecoder(openResponse.Body).Decode(&opened))
	require.NotEmptyf(t, opened.Envelope, "open response = %#v, want opaque envelope", opened)
	require.Truef(t, opened.Burned, "open response = %#v, want burn marker", opened)

	secondOpen := httptest.NewRecorder()
	server.handler.ServeHTTP(secondOpen, openRequest)
	require.Equal(t, http.StatusNotFound, secondOpen.Code)

	revocable := createTestShare(t, server, false)
	revokeRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/shares/"+revocable.ID, http.NoBody)
	revokeRequest.Header.Set("X-Revoke-Token", revocable.RevokeToken)
	revokeResponse := httptest.NewRecorder()
	server.handler.ServeHTTP(revokeResponse, revokeRequest)
	require.Equalf(t, http.StatusNoContent, revokeResponse.Code, "revoke body = %s", revokeResponse.Body.String())

	openRevoked := httptest.NewRecorder()
	server.handler.ServeHTTP(openRevoked, httptest.NewRequest(http.MethodPost, "/api/v1/shares/"+revocable.ID+"/open", http.NoBody))
	require.Equal(t, http.StatusNotFound, openRevoked.Code)
}

type testShareResponse struct {
	ID          string `json:"id"`
	RevokeToken string `json:"revoke_token"`
}

func createTestShare(t *testing.T, server *Server, burnAfterOpen bool) testShareResponse {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"envelope":           `{"version":1,"algorithm":"AES-GCM","iv":"opaque","ciphertext":"opaque"}`,
		"expires_in_seconds": 3600,
		"burn_after_open":    burnAfterOpen,
	})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/shares", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.handler.ServeHTTP(response, request)
	require.Equalf(t, http.StatusCreated, response.Code, "create body = %s", response.Body.String())
	var created testShareResponse
	require.NoError(t, json.NewDecoder(response.Body).Decode(&created))
	return created
}
