package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/demeero/sharelock/internal/config"
	"github.com/demeero/sharelock/internal/logbrick"
	"github.com/demeero/sharelock/internal/share"
	"github.com/demeero/sharelock/migration"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	logbrick.Configure(cfg.Log.Level, cfg.Log.AddSource)

	slog.Info("configuration loaded")

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), cfg.DB.StartupTimeout)
	defer cancelStartup()

	db, err := openDB(startupCtx, cfg.DB)
	if err != nil {
		slog.Error("open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("db connection established")

	if err := migration.Migrate(startupCtx, db); err != nil {
		slog.Error("migrate database", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	api := setupAPI(cfg.Version, mux, cfg.HTTP.DisableAPIDocs)

	shareService := share.New(cfg.Share, api.Shares, db)

	server := NewServer(mux, cfg, db)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go shareService.Vacuum.Exec(ctx, cfg.Share.VacuumInterval)

	if err := server.ListenAndServe(ctx); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
