package share

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/demeero/sharelock/internal/errbrick"
)

// OpenedShare is a share whose view was claimed by the caller. A nil ViewsLeft
// means the share has no open limit; zero means this read consumed the last
// view and the record was deleted.
type OpenedShare struct {
	CreatedAt     time.Time
	ExpiresAt     time.Time
	ViewsLeft     *int64
	ID            []byte
	EncryptedBlob []byte
}

type OpenShare struct {
	db             *sql.DB
	identifierSize uint
}

func NewOpenShare(db *sql.DB, identifierSize uint) *OpenShare {
	return &OpenShare{
		db:             db,
		identifierSize: identifierSize,
	}
}

// Exec claims one view of an available encrypted share and returns it. Claiming
// and deleting the last view happen in a single transaction, so concurrent
// readers can never consume more views than the share was created with.
func (c *OpenShare) Exec(ctx context.Context, id string) (OpenedShare, error) {
	decodedID, err := Decode(id, c.identifierSize)
	if err != nil {
		return OpenedShare{}, fmt.Errorf("identifier: %w", err)
	}

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return OpenedShare{}, fmt.Errorf("begin open transaction: %w", err)
	}
	defer func() {
		// A rollback after Commit returns ErrTxDone and is expected here.
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.ErrorContext(ctx, "rollback open transaction", "error", err)
		}
	}()

	record, err := claimView(ctx, tx, decodedID, time.Now().UTC())
	if err != nil {
		return OpenedShare{}, err
	}

	if record.ViewsLeft != nil && *record.ViewsLeft == 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM shares WHERE id = ?`, decodedID); err != nil {
			return OpenedShare{}, fmt.Errorf("delete consumed share: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return OpenedShare{}, fmt.Errorf("commit open transaction: %w", err)
	}

	return record, nil
}

// claimView decrements the remaining views and returns the share in one
// statement. A NULL views_left stays NULL because NULL - 1 is NULL, so shares
// without an open limit are never consumed. This write is the transaction's
// first statement on purpose: it takes the write lock immediately instead of
// upgrading from a read snapshot, which in WAL mode can fail with
// SQLITE_BUSY_SNAPSHOT that busy_timeout does not retry.
func claimView(ctx context.Context, tx *sql.Tx, id []byte, now time.Time) (OpenedShare, error) {
	var record OpenedShare
	var createdAt, expiresAt int64

	err := tx.QueryRowContext(ctx, `
		UPDATE shares
		SET views_left = views_left - 1
		WHERE id = ?
		  AND expires_at > ?
		  AND (views_left IS NULL OR views_left > 0)
		RETURNING encrypted_blob, created_at, expires_at, views_left`, id, now.Unix()).Scan(
		&record.EncryptedBlob,
		&createdAt,
		&expiresAt,
		&record.ViewsLeft,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return OpenedShare{}, errbrick.ErrNotFound
	}
	if err != nil {
		return OpenedShare{}, fmt.Errorf("claim share view: %w", err)
	}

	record.ID = id
	record.CreatedAt = time.Unix(createdAt, 0).UTC()
	record.ExpiresAt = time.Unix(expiresAt, 0).UTC()

	return record, nil
}
