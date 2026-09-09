package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetupAPI_ServesDocsByDefault(t *testing.T) {
	mux := http.NewServeMux()
	setupAPI("1.0.0", mux, false)

	for _, path := range []string{"/openapi.json", "/docs"} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, http.NoBody))
		require.Equalf(t, http.StatusOK, response.Code, "GET %s body = %s", path, response.Body.String())
	}
}

func TestSetupAPI_DisablesDocsWhenConfigured(t *testing.T) {
	mux := http.NewServeMux()
	setupAPI("1.0.0", mux, true)

	for _, path := range []string{"/openapi.json", "/docs"} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, http.NoBody))
		require.Equalf(t, http.StatusNotFound, response.Code, "GET %s body = %s", path, response.Body.String())
	}
}
