package share

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/demeero/sharelock/internal/errbrick"
)

// testIdentifierSize matches the shares.id and revoke_token_hash CHECK
// constraints in testdata/schema.sql, which both require 32-byte values.
const testIdentifierSize = 32

func TestCreateShare_Exec_RejectsInvalidInput(t *testing.T) {
	tests := map[string]struct {
		input             CreateInput
		maxEncryptedBytes uint
	}{
		"empty encrypted blob": {
			input:             CreateInput{EncryptedBlob: nil, ExpiresIn: time.Minute, CryptoVersion: 1},
			maxEncryptedBytes: 1024,
		},
		"encrypted blob exceeding max": {
			input:             CreateInput{EncryptedBlob: []byte("too big"), ExpiresIn: time.Minute, CryptoVersion: 1},
			maxEncryptedBytes: 4,
		},
		"non-positive expires in": {
			input:             CreateInput{EncryptedBlob: []byte("payload"), ExpiresIn: 0, CryptoVersion: 1},
			maxEncryptedBytes: 1024,
		},
		"expires in exceeding max TTL": {
			input:             CreateInput{EncryptedBlob: []byte("payload"), ExpiresIn: 2 * time.Hour, CryptoVersion: 1},
			maxEncryptedBytes: 1024,
		},
		"unsupported crypto version": {
			input:             CreateInput{EncryptedBlob: []byte("payload"), ExpiresIn: time.Minute, CryptoVersion: 2},
			maxEncryptedBytes: 1024,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Validation fails before the database is touched, so a nil *sql.DB is fine here.
			c := NewCreateShare(nil, tt.maxEncryptedBytes, testIdentifierSize, time.Hour)

			_, err := c.Exec(t.Context(), tt.input)

			require.ErrorIs(t, err, errbrick.ErrInvalidData)
		})
	}
}

func TestCreateShare_Exec_StoresShare(t *testing.T) {
	db := newTestDB(t)
	c := NewCreateShare(db, 1024, testIdentifierSize, time.Hour)

	before := time.Now().UTC()
	created, err := c.Exec(t.Context(), CreateInput{
		EncryptedBlob: []byte("payload"),
		ExpiresIn:     time.Minute,
		BurnAfterOpen: true,
		CryptoVersion: 1,
	})
	require.NoError(t, err)

	id, err := Decode(created.ID, testIdentifierSize)
	require.NoError(t, err)
	revokeToken, err := Decode(created.RevokeToken, testIdentifierSize)
	require.NoError(t, err)

	var (
		encryptedBlob   []byte
		revokeTokenHash []byte
		cryptoVersion   int
		createdAt       int64
		expiresAt       int64
		burnAfterOpen   int
		sizeBytes       int
	)
	require.NoError(t, db.QueryRow(`
		SELECT encrypted_blob, crypto_version, created_at, expires_at, burn_after_open, revoke_token_hash, size_bytes
		FROM shares WHERE id = ?`, id).Scan(
		&encryptedBlob, &cryptoVersion, &createdAt, &expiresAt, &burnAfterOpen, &revokeTokenHash, &sizeBytes,
	))

	assert.Equal(t, []byte("payload"), encryptedBlob)
	assert.Equal(t, 1, cryptoVersion)
	assert.Equal(t, 1, burnAfterOpen)
	assert.Equal(t, len("payload"), sizeBytes)
	assert.GreaterOrEqual(t, createdAt, before.Unix())
	assert.Equal(t, createdAt+60, expiresAt)
	expectedHash := sha256.Sum256(revokeToken)
	assert.Equal(t, expectedHash[:], revokeTokenHash)
}
