package share

import (
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// newTestDB opens a fresh SQLite database with the schema exported to
// testdata/schema.sql (see `task back:db:ddl:export`) applied to it. The DSN
// mirrors cmd/sharelock/db.go's pragmas so concurrent access (e.g. Vacuum
// running alongside test assertions) behaves like production instead of
// racing into SQLITE_BUSY.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	schema, err := os.ReadFile(filepath.Join("..", "..", "testdata", "schema.sql"))
	require.NoError(t, err)

	dsn := (&url.URL{
		Scheme:   "file",
		Path:     filepath.Join(t.TempDir(), "share.db"),
		RawQuery: "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)",
	}).String()
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(string(schema))
	require.NoError(t, err)

	return db
}

// sha256Size is the byte length of a SHA-256 digest, matching the
// revoke_token_hash column's CHECK (length(revoke_token_hash) = 32) constraint.
const sha256Size = 32

type shareRow struct {
	CreatedAt       time.Time
	ExpiresAt       time.Time
	ID              []byte
	EncryptedBlob   []byte
	RevokeTokenHash []byte
	CryptoVersion   int
	BurnAfterOpen   bool
}

// insertShare writes a row directly, bypassing CreateShare, so tests can set
// up preconditions (e.g. an already-expired share) that Exec would reject.
func insertShare(t *testing.T, db *sql.DB, row shareRow) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO shares (
			id, encrypted_blob, crypto_version, created_at, expires_at,
			burn_after_open, revoke_token_hash, size_bytes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		row.ID,
		row.EncryptedBlob,
		row.CryptoVersion,
		row.CreatedAt.Unix(),
		row.ExpiresAt.Unix(),
		boolToInteger(row.BurnAfterOpen),
		row.RevokeTokenHash,
		len(row.EncryptedBlob),
	)
	require.NoError(t, err)
}

func countShares(t *testing.T, db *sql.DB) int {
	t.Helper()

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM shares`).Scan(&count))

	return count
}
