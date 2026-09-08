package main

import (
	"context"
	"embed"
	"encoding/base64"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/demeero/sharelock/internal/config"
)

//go:embed assets/*
var assets embed.FS

// Server owns Sharelock's HTTP boundary.
type Server struct {
	handler http.Handler
	cfg     config.Config
}

// NewServer configures routes, templates, static assets, health probes, and
// security headers.
func NewServer(mux *http.ServeMux, cfg config.Config, db pinger) *Server {
	server := &Server{
		cfg: cfg,
	}

	probes := health{db: db}

	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(mustSubFS("assets"))))
	mux.HandleFunc("GET /", server.index)
	mux.HandleFunc("GET /s/{id}", server.reader)
	mux.HandleFunc("GET /health/live", probes.live)
	mux.HandleFunc("GET /health/ready", probes.ready)
	server.handler = recoverMiddleware(server.securityHeaders(mux))

	return server
}

// ListenAndServe starts the web server and blocks until it stops.
// It shuts the server down gracefully when ctx is canceled, and returns nil
// in that case; any other startup or shutdown failure is returned as an error.
func (server *Server) ListenAndServe(ctx context.Context) error {
	httpServer := &http.Server{
		Addr:              server.cfg.HTTP.Addr,
		Handler:           server.handler,
		ReadHeaderTimeout: server.cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       server.cfg.HTTP.ReadTimeout,
		WriteTimeout:      server.cfg.HTTP.WriteTimeout,
		IdleTimeout:       server.cfg.HTTP.IdleTimeout,
	}

	//nolint:gosec // shutdown must run after the request-scoped context is done, so it needs its own fresh context
	go func() {
		<-ctx.Done()

		shutdownContext, cancel := context.WithTimeout(context.Background(), server.cfg.HTTP.ShutdownTimeout)
		defer cancel()
		//nolint:contextcheck // shutdownContext is intentionally detached from the cancelled parent ctx
		if err := httpServer.Shutdown(shutdownContext); err != nil {
			slog.Error("shutdown http server", "err", err)
		}
	}()

	var err error
	if server.cfg.HTTP.TLSCertFile != "" {
		err = httpServer.ListenAndServeTLS(server.cfg.HTTP.TLSCertFile, server.cfg.HTTP.TLSKeyFile)
	} else {
		err = httpServer.ListenAndServe()
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (server *Server) index(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.NotFound(writer, request)
		return
	}
	server.serveApplication(writer)
}

func (server *Server) reader(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if len(id) != base64.RawURLEncoding.EncodedLen(int(server.cfg.Share.IdentifierSize)) {
		http.NotFound(writer, request)
		return
	}
	server.serveApplication(writer)
}

func (server *Server) serveApplication(writer http.ResponseWriter) {
	application, err := assets.ReadFile("assets/app/index.html")
	if err != nil {
		slog.Error("read frontend application", "error", err)
		http.Error(writer, "frontend application is unavailable", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := writer.Write(application); err != nil {
		slog.Error("write frontend application", "error", err)
	}
}

func (server *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().
			Set("Content-Security-Policy",
				"default-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'; img-src 'self' data:; object-src 'none'; script-src 'self'; style-src 'self'")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(writer, request)
	})
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func(ctx context.Context) {
			if recovered := recover(); recovered != nil {
				slog.ErrorContext(
					ctx,
					"http panic",
					"panic", recovered,
					"stack", string(debug.Stack()),
				)

				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}(r.Context())

		next.ServeHTTP(w, r)
	})
}

func mustSubFS(directory string) fs.FS {
	result, err := fs.Sub(assets, directory)
	if err != nil {
		panic(err)
	}

	return result
}
