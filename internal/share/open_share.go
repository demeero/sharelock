package share

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/demeero/sharelock/internal/errbrick"
)

type OpenedShare struct {
	CreatedAt       time.Time
	ExpiresAt       time.Time
	ID              []byte
	EncryptedBlob   []byte
	RevokeTokenHash []byte
	CryptoVersion   int
	BurnAfterOpen   bool
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

// Exec returns an available encrypted share.
func (c *OpenShare) Exec(ctx context.Context, id string) (OpenedShare, error) {
	decodedID, err := Decode(id, c.identifierSize)
	if err != nil {
		return OpenedShare{}, fmt.Errorf("identifier: %w", err)
	}

	isBurnAfterOpen, err := c.isBurnAfterOpen(ctx, decodedID, time.Now().UTC())
	if err != nil {
		return OpenedShare{}, err
	}

	if isBurnAfterOpen {
		record, err := c.popFromDB(ctx, decodedID, time.Now().UTC())
		if err != nil {
			return OpenedShare{}, err
		}

		return record, nil
	}

	record, err := c.loadFromDB(ctx, decodedID, time.Now().UTC())
	if err != nil {
		return OpenedShare{}, err
	}

	return record, nil
}

func (c *OpenShare) isBurnAfterOpen(ctx context.Context, id []byte, now time.Time) (bool, error) {
	var burnAfterOpen int

	err := c.db.QueryRowContext(ctx, `
		SELECT burn_after_open
		FROM shares
		WHERE id = ? AND expires_at > ?`, id, now.Unix()).Scan(&burnAfterOpen)
	if errors.Is(err, sql.ErrNoRows) {
		return false, errbrick.ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("check burn-after-open: %w", err)
	}

	return burnAfterOpen == 1, nil
}

func (c *OpenShare) popFromDB(ctx context.Context, id []byte, now time.Time) (OpenedShare, error) {
	var record OpenedShare
	var createdAt, expiresAt int64
	var burnAfterOpen int

	err := c.db.QueryRowContext(ctx, `
		DELETE FROM shares
		WHERE id = ?
		  AND burn_after_open = 1
		  AND expires_at > ?
		RETURNING encrypted_blob, crypto_version, created_at, expires_at, burn_after_open`, id, now.Unix()).Scan(
		&record.EncryptedBlob,
		&record.CryptoVersion,
		&createdAt,
		&expiresAt,
		&burnAfterOpen,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return OpenedShare{}, errbrick.ErrNotFound
	}
	if err != nil {
		return OpenedShare{}, fmt.Errorf("delete and return burn-after-open share: %w", err)
	}

	record.ID = id
	record.CreatedAt = time.Unix(createdAt, 0).UTC()
	record.ExpiresAt = time.Unix(expiresAt, 0).UTC()
	record.BurnAfterOpen = burnAfterOpen == 1

	return record, nil
}

func (c *OpenShare) loadFromDB(ctx context.Context, id []byte, now time.Time) (OpenedShare, error) {
	var record OpenedShare
	var createdAt, expiresAt int64
	var burnAfterOpen int

	err := c.db.QueryRowContext(ctx, `
		SELECT encrypted_blob, crypto_version, created_at, expires_at, burn_after_open
		FROM shares
		WHERE id = ? AND burn_after_open = 0 AND expires_at > ?`, id, now.Unix()).Scan(
		&record.EncryptedBlob,
		&record.CryptoVersion,
		&createdAt,
		&expiresAt,
		&burnAfterOpen,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return OpenedShare{}, errbrick.ErrNotFound
	}
	if err != nil {
		return OpenedShare{}, fmt.Errorf("select share: %w", err)
	}

	record.ID = id
	record.BurnAfterOpen = burnAfterOpen == 1
	record.CreatedAt = time.Unix(createdAt, 0).UTC()
	record.ExpiresAt = time.Unix(expiresAt, 0).UTC()

	return record, nil
}
