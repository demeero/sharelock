package share

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/demeero/sharelock/internal/errbrick"
)

// AccessedShare contains the opaque password-verifier data needed to validate
// a password in the browser before a view is claimed.
type AccessedShare struct {
	AccessEnvelope []byte
}

// AccessShare reads the password-verifier data without consuming a share view.
type AccessShare struct {
	db             *sql.DB
	identifierSize uint
}

// NewAccessShare builds a read-only access metadata query.
func NewAccessShare(db *sql.DB, identifierSize uint) *AccessShare {
	return &AccessShare{db: db, identifierSize: identifierSize}
}

// Exec returns access metadata for an available share without changing its view count.
func (c *AccessShare) Exec(ctx context.Context, id string) (AccessedShare, error) {
	decodedID, err := Decode(id, c.identifierSize)
	if err != nil {
		return AccessedShare{}, fmt.Errorf("identifier: %w", err)
	}

	var record AccessedShare
	err = c.db.QueryRowContext(ctx, `
		SELECT access_envelope
		FROM shares
		WHERE id = ? AND expires_at > ?`, decodedID, time.Now().UTC().Unix()).Scan(
		&record.AccessEnvelope,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return AccessedShare{}, errbrick.ErrNotFound
	}
	if err != nil {
		return AccessedShare{}, fmt.Errorf("read share access metadata: %w", err)
	}

	return record, nil
}
