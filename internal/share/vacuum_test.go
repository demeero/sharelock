package share

import (
	"bytes"
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVacuum_Exec_DeletesExpiredShares(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		db := newTestDB(t)
		now := time.Now().UTC()
		expiredID := bytes.Repeat([]byte{0x01}, testIdentifierSize)
		liveID := bytes.Repeat([]byte{0x02}, testIdentifierSize)
		insertShare(t, db, shareRow{
			ID:              expiredID,
			EncryptedBlob:   []byte("payload"),
			CryptoVersion:   1,
			CreatedAt:       now.Add(-2 * time.Hour),
			ExpiresAt:       now.Add(-time.Hour),
			RevokeTokenHash: make([]byte, sha256Size),
		})
		insertShare(t, db, shareRow{
			ID:              liveID,
			EncryptedBlob:   []byte("payload"),
			CryptoVersion:   1,
			CreatedAt:       now,
			ExpiresAt:       now.Add(time.Hour),
			RevokeTokenHash: make([]byte, sha256Size),
		})

		v := NewVacuum(db)
		ctx, cancel := context.WithCancel(t.Context())
		const interval = time.Second
		go v.Exec(ctx, interval)

		// Advance the fake clock past the first tick, then wait for the
		// vacuum goroutine to finish that tick's DELETE and block on the
		// ticker again before asserting on the database.
		time.Sleep(interval)
		synctest.Wait()
		require.Equal(t, 1, countShares(t, db))

		cancel()
		synctest.Wait()

		var remainingID []byte
		require.NoError(t, db.QueryRow(`SELECT id FROM shares`).Scan(&remainingID))
		assert.Equal(t, liveID, remainingID)
	})
}
