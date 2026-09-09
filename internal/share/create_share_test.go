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

// testMaxViews is the open limit CreateShare is configured with in tests.
const testMaxViews = 10

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
		"non-positive views": {
			input:             CreateInput{EncryptedBlob: []byte("payload"), ExpiresIn: time.Minute, Views: new(int64(0)), CryptoVersion: 1},
			maxEncryptedBytes: 1024,
		},
		"views exceeding max views": {
			input: CreateInput{
				EncryptedBlob: []byte("payload"),
				ExpiresIn:     time.Minute,
				Views:         new(int64(testMaxViews + 1)),
				CryptoVersion: 1,
			},
			maxEncryptedBytes: 1024,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Validation fails before the database is touched, so a nil *sql.DB is fine here.
			c := NewCreateShare(nil, tt.maxEncryptedBytes, testIdentifierSize, testMaxViews, time.Hour)

			_, err := c.Exec(t.Context(), tt.input)

			require.ErrorIs(t, err, errbrick.ErrInvalidData)
		})
	}
}

func TestCreateShare_Exec_StoresShare(t *testing.T) {
	db := newTestDB(t)
	c := NewCreateShare(db, 1024, testIdentifierSize, testMaxViews, time.Hour)

	before := time.Now().UTC()
	created, err := c.Exec(t.Context(), CreateInput{
		EncryptedBlob: []byte("payload"),
		ExpiresIn:     time.Minute,
		Views:         new(int64(3)),
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
		remainingViews  *int64
		sizeBytes       int
	)
	require.NoError(t, db.QueryRow(`
		SELECT encrypted_blob, crypto_version, created_at, expires_at, views_left, revoke_token_hash, size_bytes
		FROM shares WHERE id = ?`, id).Scan(
		&encryptedBlob, &cryptoVersion, &createdAt, &expiresAt, &remainingViews, &revokeTokenHash, &sizeBytes,
	))

	assert.Equal(t, []byte("payload"), encryptedBlob)
	assert.Equal(t, 1, cryptoVersion)
	assert.Equal(t, new(int64(3)), remainingViews)
	assert.Equal(t, len("payload"), sizeBytes)
	assert.GreaterOrEqual(t, createdAt, before.Unix())
	assert.Equal(t, createdAt+60, expiresAt)
	expectedHash := sha256.Sum256(revokeToken)
	assert.Equal(t, expectedHash[:], revokeTokenHash)
}

func TestCreateShare_Exec_StoresUnlimitedShareWithoutViews(t *testing.T) {
	db := newTestDB(t)
	c := NewCreateShare(db, 1024, testIdentifierSize, testMaxViews, time.Hour)

	created, err := c.Exec(t.Context(), CreateInput{
		EncryptedBlob: []byte("payload"),
		ExpiresIn:     time.Minute,
		CryptoVersion: 1,
	})
	require.NoError(t, err)

	id, err := Decode(created.ID, testIdentifierSize)
	require.NoError(t, err)

	remaining, found := viewsLeft(t, db, id)
	require.True(t, found)
	assert.Nil(t, remaining)
}
