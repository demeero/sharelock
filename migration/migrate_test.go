package migration

import (
	"bytes"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// currentSchemaVersion is the highest embedded migration number.
const currentSchemaVersion = 3

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
	if version != currentSchemaVersion || dirty {
		t.Fatalf("schema_migrations = (version=%d, dirty=%t), want (%d, false)", version, dirty, currentSchemaVersion)
	}

	assertColumnExists(t, db, "shares", "views_left")
	assertColumnExists(t, db, "shares", "access_envelope")
	assertColumnAbsent(t, db, "shares", "crypto_version")
}

// TestMigrateReplacesBurnAfterOpenWithViews pins the data mapping of migration
// 000002: a burn-after-open share becomes a single-view share, and a reusable
// one becomes a share with no view limit at all.
func TestMigrateReplacesBurnAfterOpenWithViews(t *testing.T) {
	t.Parallel()
	db, err := openTestDB(t)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()

	migrator, err := newMigrator(db)
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}
	if err := migrator.Steps(1); err != nil {
		t.Fatalf("apply first migration: %v", err)
	}

	burnID := bytes.Repeat([]byte{0x01}, 32)
	reusableID := bytes.Repeat([]byte{0x02}, 32)
	for _, row := range []struct {
		id            []byte
		burnAfterOpen int
	}{{burnID, 1}, {reusableID, 0}} {
		if _, err := db.Exec(`
			INSERT INTO shares (
				id, encrypted_blob, crypto_version, created_at, expires_at,
				burn_after_open, revoke_token_hash, size_bytes
			) VALUES (?, ?, 1, 1000, 2000, ?, ?, 7)`,
			row.id, []byte("payload"), row.burnAfterOpen, bytes.Repeat([]byte{0x03}, 32)); err != nil {
			t.Fatalf("insert legacy share: %v", err)
		}
	}

	if err := Migrate(t.Context(), db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	for _, tt := range []struct {
		want *int64
		name string
		id   []byte
	}{
		{name: "burn after open", id: burnID, want: new(int64(1))},
		{name: "reusable", id: reusableID, want: nil},
	} {
		var got *int64
		if err := db.QueryRow(`SELECT views_left FROM shares WHERE id = ?`, tt.id).Scan(&got); err != nil {
			t.Fatalf("read %s views_left: %v", tt.name, err)
		}
		switch {
		case tt.want == nil && got != nil:
			t.Fatalf("%s views_left = %d, want NULL", tt.name, *got)
		case tt.want != nil && got == nil:
			t.Fatalf("%s views_left = NULL, want %d", tt.name, *tt.want)
		case tt.want != nil && *got != *tt.want:
			t.Fatalf("%s views_left = %d, want %d", tt.name, *got, *tt.want)
		}
	}

	var accessEnvelope []byte
	if err := db.QueryRow(`SELECT access_envelope FROM shares WHERE id = ?`, burnID).Scan(&accessEnvelope); err != nil {
		t.Fatalf("read preserved access envelope: %v", err)
	}
	if accessEnvelope != nil {
		t.Fatalf("legacy share access_envelope = %q, want NULL", accessEnvelope)
	}

	// The rebuild drops the table, so the expiry index must have been recreated.
	var index string
	if err := db.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type = 'index' AND name = 'shares_expiry_idx'`).Scan(&index); err != nil {
		t.Fatalf("find shares_expiry_idx: %v", err)
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

func TestMigrateDoesNotDiscardPasswordProtectedSharesOnDown(t *testing.T) {
	t.Parallel()
	db, err := openTestDB(t)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()

	migrator, err := newMigrator(db)
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}
	if err := migrator.Steps(3); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	id := bytes.Repeat([]byte{0x01}, 32)
	if _, err := db.Exec(`
		INSERT INTO shares (
			id, encrypted_blob, access_envelope, created_at, expires_at,
			views_left, revoke_token_hash, size_bytes
		) VALUES (?, ?, ?, 1000, 2000, NULL, ?, 15)`,
		id, []byte("payload"), []byte("verifier"), bytes.Repeat([]byte{0x03}, 32)); err != nil {
		t.Fatalf("insert password-protected share: %v", err)
	}

	if err := migrator.Steps(-1); err == nil {
		t.Fatal("downgrade unexpectedly succeeded with a password-protected share")
	}

	var accessEnvelope []byte
	if err := db.QueryRow(`SELECT access_envelope FROM shares WHERE id = ?`, id).Scan(&accessEnvelope); err != nil {
		t.Fatalf("read password-protected share after failed downgrade: %v", err)
	}
	if !bytes.Equal(accessEnvelope, []byte("verifier")) {
		t.Fatalf("password-protected share changed by failed downgrade: verifier=%q", accessEnvelope)
	}
}

func openTestDB(t *testing.T) (*sql.DB, error) {
	t.Helper()

	return sql.Open("sqlite", filepath.Join(t.TempDir(), "sharelock.db"))
}

func assertColumnExists(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()
	var count int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&count); err != nil {
		t.Fatalf("inspect %s.%s: %v", table, column, err)
	}
	if count != 1 {
		t.Fatalf("column %s.%s not found", table, column)
	}
}

func assertColumnAbsent(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()
	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column,
	).Scan(&count); err != nil {
		t.Fatalf("inspect %s.%s: %v", table, column, err)
	}
	if count != 0 {
		t.Fatalf("column %s.%s unexpectedly exists", table, column)
	}
}

func assertTableExists(t *testing.T, db *sql.DB, name string) {
	t.Helper()
	var found string
	if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&found); err != nil {
		t.Fatalf("find table %q: %v", name, err)
	}
}
