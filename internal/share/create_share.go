package share

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"

	"github.com/demeero/sharelock/internal/errbrick"
)

// CreateInput is the server-visible part of a newly created share.
// A nil Views means the share stays readable until it expires.
type CreateInput struct {
	Views          *int64
	EncryptedBlob  []byte
	AccessEnvelope []byte
	ExpiresIn      time.Duration
}

// CreatedShare is returned once a share was committed. The revoke token is a
// one-time capability and must be shown only to the creating browser.
type CreatedShare struct {
	ID          string
	RevokeToken string
}

type createDBInput struct {
	CreatedAt       time.Time
	ExpiresAt       time.Time
	ViewsLeft       *int64
	ID              []byte
	EncryptedBlob   []byte
	AccessEnvelope  []byte
	RevokeTokenHash []byte
}

type CreateShare struct {
	db                *sql.DB
	maxEncryptedBytes uint
	identifierSize    uint
	maxViews          uint
	maxTTL            time.Duration
}

func NewCreateShare(db *sql.DB, maxEncryptedBytes, identifierSize, maxViews uint, maxTTL time.Duration) *CreateShare {
	return &CreateShare{
		db:                db,
		maxEncryptedBytes: maxEncryptedBytes,
		maxTTL:            maxTTL,
		identifierSize:    identifierSize,
		maxViews:          maxViews,
	}
}

// Exec validates and stores an opaque encrypted share.
func (c *CreateShare) Exec(ctx context.Context, input CreateInput) (CreatedShare, error) {
	if err := c.validate(input); err != nil {
		return CreatedShare{}, err
	}

	id := make([]byte, c.identifierSize)
	if _, err := rand.Read(id); err != nil {
		return CreatedShare{}, fmt.Errorf("generate share identifier: %w", err)
	}

	revokeToken := make([]byte, c.identifierSize)
	if _, err := rand.Read(revokeToken); err != nil {
		return CreatedShare{}, fmt.Errorf("generate revoke token: %w", err)
	}

	now := time.Now().UTC()
	hash := sha256.Sum256(revokeToken)
	dbInput := createDBInput{
		ID:              id,
		EncryptedBlob:   input.EncryptedBlob,
		AccessEnvelope:  input.AccessEnvelope,
		CreatedAt:       now,
		ExpiresAt:       now.Add(input.ExpiresIn),
		ViewsLeft:       input.Views,
		RevokeTokenHash: hash[:],
	}
	if err := c.insert(ctx, dbInput); err != nil {
		return CreatedShare{}, fmt.Errorf("store share: %w", err)
	}

	return CreatedShare{
		ID:          Encode(id),
		RevokeToken: Encode(revokeToken),
	}, nil
}

func (c *CreateShare) validate(input CreateInput) error {
	encryptedBytes := len(input.EncryptedBlob) + len(input.AccessEnvelope)
	if len(input.EncryptedBlob) == 0 || encryptedBytes > int(c.maxEncryptedBytes) {
		return fmt.Errorf("%w: encrypted payload size must be between 1 and %d bytes", errbrick.ErrInvalidData, c.maxEncryptedBytes)
	}
	if input.ExpiresIn <= 0 || input.ExpiresIn > c.maxTTL {
		return fmt.Errorf("%w: expiration must be between 1 and %.0f seconds", errbrick.ErrInvalidData, c.maxTTL.Seconds())
	}
	if input.Views != nil && (*input.Views < 1 || *input.Views > int64(c.maxViews)) {
		return fmt.Errorf("%w: views must be between 1 and %d", errbrick.ErrInvalidData, c.maxViews)
	}

	return nil
}

func (c *CreateShare) insert(ctx context.Context, input createDBInput) error {
	_, err := c.db.ExecContext(ctx, `
		INSERT INTO shares (
			id, encrypted_blob, access_envelope, created_at, expires_at,
			views_left, revoke_token_hash, size_bytes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		input.ID,
		input.EncryptedBlob,
		input.AccessEnvelope,
		input.CreatedAt.Unix(),
		input.ExpiresAt.Unix(),
		input.ViewsLeft,
		input.RevokeTokenHash,
		len(input.EncryptedBlob)+len(input.AccessEnvelope),
	)

	return err
}
