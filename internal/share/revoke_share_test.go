package share

import (
	"bytes"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/demeero/sharelock/internal/errbrick"
)

func TestRevokeShare_Exec_RejectsInvalidInput(t *testing.T) {
	validID := Encode(make([]byte, testIdentifierSize))
	validToken := Encode(make([]byte, testIdentifierSize))

	tests := map[string]struct {
		id          string
		revokeToken string
	}{
		"invalid identifier":   {id: "not base64!!", revokeToken: validToken},
		"invalid revoke token": {id: validID, revokeToken: "not base64!!"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Validation fails before the database is touched, so a nil *sql.DB is fine here.
			r := NewRevokeShare(nil, testIdentifierSize)

			err := r.Exec(t.Context(), tt.id, tt.revokeToken)

			require.ErrorIs(t, err, errbrick.ErrInvalidData)
		})
	}
}

func TestRevokeShare_Exec_ReturnsNotFoundForWrongToken(t *testing.T) {
	db := newTestDB(t)
	id := bytes.Repeat([]byte{0x01}, testIdentifierSize)
	revokeToken := bytes.Repeat([]byte{0x02}, testIdentifierSize)
	hash := sha256.Sum256(revokeToken)
	now := time.Now().UTC()
	insertShare(t, db, shareRow{
		ID:              id,
		EncryptedBlob:   []byte("payload"),
		CreatedAt:       now,
		ExpiresAt:       now.Add(time.Hour),
		RevokeTokenHash: hash[:],
	})
	r := NewRevokeShare(db, testIdentifierSize)

	wrongToken := bytes.Repeat([]byte{0x03}, testIdentifierSize)
	err := r.Exec(t.Context(), Encode(id), Encode(wrongToken))

	require.ErrorIs(t, err, errbrick.ErrNotFound)
	require.Equal(t, 1, countShares(t, db))
}

func TestRevokeShare_Exec_DeletesShareWithMatchingToken(t *testing.T) {
	db := newTestDB(t)
	id := bytes.Repeat([]byte{0x01}, testIdentifierSize)
	revokeToken := bytes.Repeat([]byte{0x02}, testIdentifierSize)
	hash := sha256.Sum256(revokeToken)
	now := time.Now().UTC()
	insertShare(t, db, shareRow{
		ID:              id,
		EncryptedBlob:   []byte("payload"),
		CreatedAt:       now,
		ExpiresAt:       now.Add(time.Hour),
		RevokeTokenHash: hash[:],
	})
	r := NewRevokeShare(db, testIdentifierSize)

	err := r.Exec(t.Context(), Encode(id), Encode(revokeToken))

	require.NoError(t, err)
	require.Equal(t, 0, countShares(t, db))
}
