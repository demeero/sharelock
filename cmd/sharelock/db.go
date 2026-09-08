package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/demeero/sharelock/internal/config"

	_ "modernc.org/sqlite"
)

func openDB(ctx context.Context, cfg config.DBConfig) (*sql.DB, error) {
	absPath, err := filepath.Abs(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o750); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	dsn := (&url.URL{
		Scheme:   "file",
		Path:     absPath,
		RawQuery: "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)",
	}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(int(cfg.MaxOpenConns))
	db.SetMaxIdleConns(int(cfg.MaxIdleConns))

	ctx, cancel := context.WithTimeout(ctx, cfg.StartupTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("ping sqlite: %w; close database: %v", err, closeErr)
		}
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return db, nil
}
