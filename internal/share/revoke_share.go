package share

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"

	"github.com/demeero/sharelock/internal/errbrick"
)

type RevokeShare struct {
	db             *sql.DB
	identifierSize uint
}

func NewRevokeShare(db *sql.DB, identifierSize uint) *RevokeShare {
	return &RevokeShare{db: db, identifierSize: identifierSize}
}

// Exec deletes a share when its matching revoke token is supplied.
func (c *RevokeShare) Exec(ctx context.Context, id, revokeToken string) error {
	decodedID, err := Decode(id, c.identifierSize)
	if err != nil {
		return fmt.Errorf("identifier: %w", err)
	}

	decodedRevokeToken, err := Decode(revokeToken, c.identifierSize)
	if err != nil {
		return fmt.Errorf("revokeToken: %w", err)
	}

	hash := sha256.Sum256(decodedRevokeToken)
	if err := c.delFromDB(ctx, decodedID, hash[:]); err != nil {
		return err
	}

	return nil
}

// Revoke deletes a share only when its revoke capability matches.
func (c *RevokeShare) delFromDB(ctx context.Context, id, revokeTokenHash []byte) error {
	result, err := c.db.ExecContext(ctx, `
		DELETE FROM shares WHERE id = ? AND revoke_token_hash = ?`, id, revokeTokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return errbrick.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("delete share: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if count == 0 {
		return errbrick.ErrNotFound
	}

	return nil
}
