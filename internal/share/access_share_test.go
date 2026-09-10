package share

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/demeero/sharelock/internal/errbrick"
)

func TestAccessShare_Exec_ReturnsPasswordVerifierWithoutConsumingView(t *testing.T) {
	db := newTestDB(t)
	id := bytes.Repeat([]byte{0x01}, testIdentifierSize)
	insertShare(t, db, shareRow{
		ID:              id,
		EncryptedBlob:   []byte("payload"),
		AccessEnvelope:  []byte("verifier"),
		CreatedAt:       time.Now().UTC(),
		ExpiresAt:       time.Now().UTC().Add(time.Hour),
		ViewsLeft:       new(int64(1)),
		RevokeTokenHash: make([]byte, sha256Size),
	})

	record, err := NewAccessShare(db, testIdentifierSize).Exec(t.Context(), Encode(id))

	require.NoError(t, err)
	assert.Equal(t, []byte("verifier"), record.AccessEnvelope)
	remaining, found := viewsLeft(t, db, id)
	require.True(t, found)
	assert.Equal(t, new(int64(1)), remaining)
}

func TestAccessShare_Exec_ReturnsMetadataWithoutPasswordVerifier(t *testing.T) {
	db := newTestDB(t)
	id := bytes.Repeat([]byte{0x01}, testIdentifierSize)
	insertShare(t, db, shareRow{
		ID:              id,
		EncryptedBlob:   []byte("payload"),
		CreatedAt:       time.Now().UTC(),
		ExpiresAt:       time.Now().UTC().Add(time.Hour),
		RevokeTokenHash: make([]byte, sha256Size),
	})

	record, err := NewAccessShare(db, testIdentifierSize).Exec(t.Context(), Encode(id))

	require.NoError(t, err)
	assert.Empty(t, record.AccessEnvelope)
}

func TestAccessShare_Exec_ReturnsNotFoundForUnavailableShare(t *testing.T) {
	db := newTestDB(t)
	id := bytes.Repeat([]byte{0x01}, testIdentifierSize)
	insertShare(t, db, shareRow{
		ID:              id,
		EncryptedBlob:   []byte("payload"),
		CreatedAt:       time.Now().UTC().Add(-2 * time.Hour),
		ExpiresAt:       time.Now().UTC().Add(-time.Hour),
		RevokeTokenHash: make([]byte, sha256Size),
	})

	_, err := NewAccessShare(db, testIdentifierSize).Exec(t.Context(), Encode(id))

	require.ErrorIs(t, err, errbrick.ErrNotFound)
}
