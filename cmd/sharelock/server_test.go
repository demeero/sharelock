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
	cfg := config.Config{
		HTTP:  config.HTTPConfig{Addr: ":0", ReadHeaderTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second},
		DB:    config.DBConfig{Path: filepath.Join(t.TempDir(), "sharelock.db"), MaxOpenConns: 1, MaxIdleConns: 1, StartupTimeout: time.Second},
		Share: config.ShareConfig{MaxEncryptedBytes: 1024, MaxTTL: 24 * time.Hour, IdentifierSize: 32, MaxViews: 10, VacuumInterval: time.Hour},
	}
	db, err := sql.Open("sqlite", cfg.DB.Path)
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, migration.Migrate(t.Context(), db))

	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("test", "1.0.0"))
	share.New(cfg.Share, huma.NewGroup(api, "/api/v1/shares"), db)
	server := NewServer(mux, cfg, db)

	for _, path := range []string{"/", "/s/" + strings.Repeat("a", 43)} {
		response := httptest.NewRecorder()
		server.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, http.NoBody))
		require.Equalf(t, http.StatusOK, response.Code, "GET %s body = %s", path, response.Body.String())
		require.Containsf(t, response.Body.String(), "id=\"app\"", "GET %s does not serve the frontend application", path)
	}
	openAPIDocument := httptest.NewRecorder()
	server.handler.ServeHTTP(openAPIDocument, httptest.NewRequest(http.MethodGet, "/openapi.json", http.NoBody))
	require.Equalf(t, http.StatusOK, openAPIDocument.Code, "GET /openapi.json body = %s", openAPIDocument.Body.String())
	for _, operationID := range []string{"createShare", "getShareAccess", "openShare", "revokeShare", "getShareSettings"} {
		require.Contains(t, openAPIDocument.Body.String(), "\"operationId\":\""+operationID+"\"")
	}

	settingsResponse := httptest.NewRecorder()
	server.handler.ServeHTTP(settingsResponse, httptest.NewRequest(http.MethodGet, "/api/v1/shares/settings", http.NoBody))
	require.Equalf(t, http.StatusOK, settingsResponse.Code, "settings body = %s", settingsResponse.Body.String())
	var settings struct {
		MaxEncryptedBytes uint  `json:"max_encrypted_bytes"`
		MaxTTLSeconds     int64 `json:"max_ttl_seconds"`
		MaxViews          uint  `json:"max_views"`
	}
	require.NoError(t, json.NewDecoder(settingsResponse.Body).Decode(&settings))
	require.Equal(t, cfg.Share.MaxEncryptedBytes, settings.MaxEncryptedBytes)
	require.Equal(t, int64(cfg.Share.MaxTTL/time.Second), settings.MaxTTLSeconds)
	require.Equal(t, cfg.Share.MaxViews, settings.MaxViews)

	created := createTestShare(t, server, new(int64(2)))
	openRequest := httptest.NewRequest(http.MethodPost, "/api/v1/shares/"+created.ID+"/open", http.NoBody)
	for _, wantViewsLeft := range []int64{1, 0} {
		openResponse := httptest.NewRecorder()
		server.handler.ServeHTTP(openResponse, openRequest)
		require.Equalf(t, http.StatusOK, openResponse.Code, "open body = %s", openResponse.Body.String())
		opened := decodeOpenedShare(t, openResponse)
		require.NotEmptyf(t, opened.Envelope, "open response = %#v, want opaque envelope", opened)
		require.Equal(t, &wantViewsLeft, opened.ViewsLeft)
		require.Equal(t, wantViewsLeft == 0, opened.Burned)
	}

	thirdOpen := httptest.NewRecorder()
	server.handler.ServeHTTP(thirdOpen, openRequest)
	require.Equal(t, http.StatusNotFound, thirdOpen.Code)

	revocable := createTestShare(t, server, nil)
	revokeRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/shares/"+revocable.ID, http.NoBody)
	revokeRequest.Header.Set("X-Revoke-Token", revocable.RevokeToken)
	revokeResponse := httptest.NewRecorder()
	server.handler.ServeHTTP(revokeResponse, revokeRequest)
	require.Equalf(t, http.StatusNoContent, revokeResponse.Code, "revoke body = %s", revokeResponse.Body.String())

	openRevoked := httptest.NewRecorder()
	server.handler.ServeHTTP(openRevoked, httptest.NewRequest(http.MethodPost, "/api/v1/shares/"+revocable.ID+"/open", http.NoBody))
	require.Equal(t, http.StatusNotFound, openRevoked.Code)

	accessRevoked := httptest.NewRecorder()
	server.handler.ServeHTTP(
		accessRevoked,
		httptest.NewRequest(http.MethodGet, "/api/v1/shares/"+revocable.ID+"/access", http.NoBody),
	)
	require.Equal(t, http.StatusNotFound, accessRevoked.Code)
}

// TestShareHTTPFlow_UnlimitedViews covers a share created without a view limit:
// it stays readable and reports no remaining-view count.
func TestShareHTTPFlow_UnlimitedViews(t *testing.T) {
	server := newTestServer(t)

	created := createTestShare(t, server, nil)
	openRequest := httptest.NewRequest(http.MethodPost, "/api/v1/shares/"+created.ID+"/open", http.NoBody)
	for range 3 {
		response := httptest.NewRecorder()
		server.handler.ServeHTTP(response, openRequest)
		require.Equalf(t, http.StatusOK, response.Code, "open body = %s", response.Body.String())
		opened := decodeOpenedShare(t, response)
		require.Nil(t, opened.ViewsLeft)
		require.False(t, opened.Burned)
	}
}

func TestShareHTTPFlow_RejectsViewsAboveMax(t *testing.T) {
	server := newTestServer(t)

	body, err := json.Marshal(map[string]any{
		"envelope":           testEnvelope,
		"expires_in_seconds": 3600,
		"views":              11,
	})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/shares", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.handler.ServeHTTP(response, request)

	require.Equalf(t, http.StatusBadRequest, response.Code, "create body = %s", response.Body.String())
	require.Contains(t, response.Body.String(), "views must be between 1 and 10")
}

func TestShareHTTPFlow_PasswordProtectedShareDoesNotConsumeViewDuringAccessCheck(t *testing.T) {
	server := newTestServer(t)
	body, err := json.Marshal(map[string]any{
		"envelope":           testPasswordEnvelope,
		"access_envelope":    testAccessEnvelope,
		"expires_in_seconds": 3600,
		"views":              1,
	})
	require.NoError(t, err)
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/shares", bytes.NewReader(body))
	createRequest.Header.Set("Content-Type", "application/json")
	createResponse := httptest.NewRecorder()
	server.handler.ServeHTTP(createResponse, createRequest)
	require.Equal(t, http.StatusCreated, createResponse.Code)

	var created testShareResponse
	require.NoError(t, json.NewDecoder(createResponse.Body).Decode(&created))

	accessResponse := httptest.NewRecorder()
	server.handler.ServeHTTP(
		accessResponse,
		httptest.NewRequest(http.MethodGet, "/api/v1/shares/"+created.ID+"/access", http.NoBody),
	)
	require.Equalf(t, http.StatusOK, accessResponse.Code, "access body = %s", accessResponse.Body.String())
	var access struct {
		AccessEnvelope string `json:"access_envelope"`
	}
	require.NoError(t, json.NewDecoder(accessResponse.Body).Decode(&access))
	require.JSONEq(t, testAccessEnvelope, access.AccessEnvelope)

	openResponse := httptest.NewRecorder()
	server.handler.ServeHTTP(
		openResponse,
		httptest.NewRequest(http.MethodPost, "/api/v1/shares/"+created.ID+"/open", http.NoBody),
	)
	require.Equalf(t, http.StatusOK, openResponse.Code, "open body = %s", openResponse.Body.String())
	opened := decodeOpenedShare(t, openResponse)
	require.True(t, opened.Burned)
}

const testEnvelope = `{"algorithm":"AES-GCM","iv":"opaque","ciphertext":"opaque"}`
const testPasswordEnvelope = `{"algorithm":"AES-GCM","iv":"opaque","ciphertext":"opaque"}`
const testAccessEnvelope = `{"algorithm":"AES-GCM","iv":"opaque","ciphertext":"opaque","kdf":"opaque"}`

type testShareResponse struct {
	ID          string `json:"id"`
	RevokeToken string `json:"revoke_token"`
}

type testOpenedShare struct {
	ViewsLeft *int64 `json:"views_left"`
	Envelope  string `json:"envelope"`
	Burned    bool   `json:"burned"`
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := config.Config{
		HTTP:  config.HTTPConfig{Addr: ":0", ReadHeaderTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second},
		DB:    config.DBConfig{Path: filepath.Join(t.TempDir(), "sharelock.db"), MaxOpenConns: 1, MaxIdleConns: 1, StartupTimeout: time.Second},
		Share: config.ShareConfig{MaxEncryptedBytes: 1024, MaxTTL: 24 * time.Hour, IdentifierSize: 32, MaxViews: 10, VacuumInterval: time.Hour},
	}
	db, err := sql.Open("sqlite", cfg.DB.Path)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	require.NoError(t, migration.Migrate(t.Context(), db))

	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("test", "1.0.0"))
	share.New(cfg.Share, huma.NewGroup(api, "/api/v1/shares"), db)

	return NewServer(mux, cfg, db)
}

func decodeOpenedShare(t *testing.T, response *httptest.ResponseRecorder) testOpenedShare {
	t.Helper()
	var opened testOpenedShare
	require.NoError(t, json.NewDecoder(response.Body).Decode(&opened))
	return opened
}

// createTestShare creates a share with the given view limit; a nil views means
// the share stays readable until it expires.
func createTestShare(t *testing.T, server *Server, views *int64) testShareResponse {
	t.Helper()
	payload := map[string]any{
		"envelope":           testEnvelope,
		"expires_in_seconds": 3600,
	}
	if views != nil {
		payload["views"] = *views
	}
	body, err := json.Marshal(payload)
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
