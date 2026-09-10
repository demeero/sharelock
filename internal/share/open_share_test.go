package share

import (
	"bytes"
	"sync"
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
		CreatedAt:       now.Add(-2 * time.Hour),
		ExpiresAt:       now.Add(-time.Hour),
		RevokeTokenHash: make([]byte, sha256Size),
	})
	o := NewOpenShare(db, testIdentifierSize)

	_, err := o.Exec(t.Context(), Encode(id))

	require.ErrorIs(t, err, errbrick.ErrNotFound)
}

func TestOpenShare_Exec_KeepsShareWithoutViewLimit(t *testing.T) {
	db := newTestDB(t)
	id := bytes.Repeat([]byte{0x01}, testIdentifierSize)
	now := time.Now().UTC()
	insertShare(t, db, shareRow{
		ID:              id,
		EncryptedBlob:   []byte("payload"),
		CreatedAt:       now,
		ExpiresAt:       now.Add(time.Hour),
		RevokeTokenHash: make([]byte, sha256Size),
	})
	o := NewOpenShare(db, testIdentifierSize)

	for range 3 {
		record, err := o.Exec(t.Context(), Encode(id))
		require.NoError(t, err)
		assert.Equal(t, []byte("payload"), record.EncryptedBlob)
		assert.Nil(t, record.ViewsLeft)
	}

	remaining, found := viewsLeft(t, db, id)
	require.True(t, found)
	assert.Nil(t, remaining)
}

func TestOpenShare_Exec_CountsDownViewsAndDeletesTheLastOne(t *testing.T) {
	db := newTestDB(t)
	id := bytes.Repeat([]byte{0x01}, testIdentifierSize)
	now := time.Now().UTC()
	insertShare(t, db, shareRow{
		ID:              id,
		EncryptedBlob:   []byte("payload"),
		CreatedAt:       now,
		ExpiresAt:       now.Add(time.Hour),
		ViewsLeft:       new(int64(3)),
		RevokeTokenHash: make([]byte, sha256Size),
	})
	o := NewOpenShare(db, testIdentifierSize)

	for _, want := range []int64{2, 1, 0} {
		record, err := o.Exec(t.Context(), Encode(id))
		require.NoError(t, err)
		assert.Equal(t, []byte("payload"), record.EncryptedBlob)
		assert.Equal(t, new(want), record.ViewsLeft)
	}

	assert.Equal(t, 0, countShares(t, db))

	_, err := o.Exec(t.Context(), Encode(id))
	require.ErrorIs(t, err, errbrick.ErrNotFound)
}

// TestOpenShare_Exec_ConcurrentOpensConsumeEachViewOnce is the reason claiming a
// view is a single UPDATE inside a transaction: a read-modify-write would let
// concurrent readers decrement the same value and serve more views than the
// share was created with.
func TestOpenShare_Exec_ConcurrentOpensConsumeEachViewOnce(t *testing.T) {
	const (
		allowedViews = 5
		readers      = 20
	)

	db := newTestDB(t)
	id := bytes.Repeat([]byte{0x01}, testIdentifierSize)
	now := time.Now().UTC()
	insertShare(t, db, shareRow{
		ID:              id,
		EncryptedBlob:   []byte("payload"),
		CreatedAt:       now,
		ExpiresAt:       now.Add(time.Hour),
		ViewsLeft:       new(int64(allowedViews)),
		RevokeTokenHash: make([]byte, sha256Size),
	})
	o := NewOpenShare(db, testIdentifierSize)

	var (
		mu        sync.Mutex
		remaining []int64
		failures  []error
		wg        sync.WaitGroup
	)
	for range readers {
		wg.Go(func() {
			record, err := o.Exec(t.Context(), Encode(id))

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failures = append(failures, err)
				return
			}
			remaining = append(remaining, *record.ViewsLeft)
		})
	}
	wg.Wait()

	assert.Len(t, remaining, allowedViews)
	require.Len(t, failures, readers-allowedViews)
	// Losing readers must be turned away because the views ran out, not because
	// they collided on the database.
	for _, err := range failures {
		require.ErrorIs(t, err, errbrick.ErrNotFound)
	}
	// Every successful reader must have claimed a distinct view.
	assert.ElementsMatch(t, []int64{4, 3, 2, 1, 0}, remaining)
	assert.Equal(t, 0, countShares(t, db))
}
