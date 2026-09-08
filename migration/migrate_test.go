package migration

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigrateCreatesCurrentSchemaAndIsIdempotent(t *testing.T) {
	t.Parallel()
	db, err := openTestDB(t)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()

	for range 2 {
		if err := Migrate(t.Context(), db); err != nil {
			t.Fatalf("Migrate() error = %v", err)
		}
	}

	assertTableExists(t, db, "shares")
	assertTableExists(t, db, "schema_migrations")

	var version uint
	var dirty bool
	if err := db.QueryRow(`SELECT version, dirty FROM schema_migrations`).Scan(&version, &dirty); err != nil {
		t.Fatalf("read migration version: %v", err)
	}
	if version != 1 || dirty {
		t.Fatalf("schema_migrations = (version=%d, dirty=%t), want (1, false)", version, dirty)
	}
}

func TestMigrateAdoptsExistingShareSchema(t *testing.T) {
	t.Parallel()
	db, err := openTestDB(t)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE shares (
			id BLOB PRIMARY KEY NOT NULL CHECK (length(id) = 32),
			encrypted_blob BLOB NOT NULL CHECK (length(encrypted_blob) > 0),
			crypto_version INTEGER NOT NULL CHECK (crypto_version = 1),
			created_at INTEGER NOT NULL,
			expires_at INTEGER NOT NULL,
			burn_after_open INTEGER NOT NULL CHECK (burn_after_open IN (0, 1)),
			revoke_token_hash BLOB NOT NULL CHECK (length(revoke_token_hash) = 32),
			size_bytes INTEGER NOT NULL CHECK (size_bytes > 0),
			CHECK (expires_at > created_at)
		);`); err != nil {
		t.Fatalf("create legacy shares table: %v", err)
	}

	if err := Migrate(t.Context(), db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	assertTableExists(t, db, "schema_migrations")
}

func openTestDB(t *testing.T) (*sql.DB, error) {
	t.Helper()

	return sql.Open("sqlite", filepath.Join(t.TempDir(), "sharelock.db"))
}

func assertTableExists(t *testing.T, db *sql.DB, name string) {
	t.Helper()
	var found string
	if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&found); err != nil {
		t.Fatalf("find table %q: %v", name, err)
	}
}
