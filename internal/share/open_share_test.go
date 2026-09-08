package share

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/demeero/sharelock/internal/errbrick"
)

func TestOpenShare_Exec_RejectsInvalidInput(t *testing.T) {
	tests := map[string]struct {
		wantErr error
		id      string
	}{
		"invalid identifier": {id: "not base64!!", wantErr: errbrick.ErrInvalidData},
		"unknown identifier": {id: Encode(make([]byte, testIdentifierSize)), wantErr: errbrick.ErrNotFound},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			o := NewOpenShare(newTestDB(t), testIdentifierSize)

			_, err := o.Exec(t.Context(), tt.id)

			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestOpenShare_Exec_ReturnsNotFoundForExpiredShare(t *testing.T) {
	db := newTestDB(t)
	id := bytes.Repeat([]byte{0x01}, testIdentifierSize)
	now := time.Now().UTC()
	insertShare(t, db, shareRow{
		ID:              id,
		EncryptedBlob:   []byte("payload"),
		CryptoVersion:   1,
		CreatedAt:       now.Add(-2 * time.Hour),
		ExpiresAt:       now.Add(-time.Hour),
		RevokeTokenHash: make([]byte, sha256Size),
	})
	o := NewOpenShare(db, testIdentifierSize)

	_, err := o.Exec(t.Context(), Encode(id))

	require.ErrorIs(t, err, errbrick.ErrNotFound)
}

func TestOpenShare_Exec_ReturnsShareAndKeepsItWhenNotBurnAfterOpen(t *testing.T) {
	db := newTestDB(t)
	id := bytes.Repeat([]byte{0x01}, testIdentifierSize)
	now := time.Now().UTC()
	insertShare(t, db, shareRow{
		ID:              id,
		EncryptedBlob:   []byte("payload"),
		CryptoVersion:   1,
		CreatedAt:       now,
		ExpiresAt:       now.Add(time.Hour),
		BurnAfterOpen:   false,
		RevokeTokenHash: make([]byte, sha256Size),
	})
	o := NewOpenShare(db, testIdentifierSize)

	record, err := o.Exec(t.Context(), Encode(id))
	require.NoError(t, err)

	assert.Equal(t, []byte("payload"), record.EncryptedBlob)
	assert.False(t, record.BurnAfterOpen)
	assert.Equal(t, 1, countShares(t, db))
}

func TestOpenShare_Exec_DeletesShareWhenBurnAfterOpen(t *testing.T) {
	db := newTestDB(t)
	id := bytes.Repeat([]byte{0x01}, testIdentifierSize)
	now := time.Now().UTC()
	insertShare(t, db, shareRow{
		ID:              id,
		EncryptedBlob:   []byte("payload"),
		CryptoVersion:   1,
		CreatedAt:       now,
		ExpiresAt:       now.Add(time.Hour),
		BurnAfterOpen:   true,
		RevokeTokenHash: make([]byte, sha256Size),
	})
	o := NewOpenShare(db, testIdentifierSize)

	record, err := o.Exec(t.Context(), Encode(id))
	require.NoError(t, err)
	assert.True(t, record.BurnAfterOpen)
	assert.Equal(t, 0, countShares(t, db))

	_, err = o.Exec(t.Context(), Encode(id))
	require.ErrorIs(t, err, errbrick.ErrNotFound)
}
