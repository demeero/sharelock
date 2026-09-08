package share

import (
	"database/sql"

	"github.com/demeero/sharelock/internal/config"

	"github.com/danielgtaylor/huma/v2"
)

type Share struct {
	Create *CreateShare
	Open   *OpenShare
	Revoke *RevokeShare
	Vacuum *Vacuum
}

func New(cfg config.ShareConfig, api huma.API, db *sql.DB) *Share {
	createShare := NewCreateShare(db, cfg.MaxEncryptedBytes, cfg.IdentifierSize, cfg.MaxTTL)
	openShare := NewOpenShare(db, cfg.IdentifierSize)
	revokeShare := NewRevokeShare(db, cfg.IdentifierSize)
	vacuumShare := NewVacuum(db)

	share := &Share{
		Create: createShare,
		Open:   openShare,
		Revoke: revokeShare,
		Vacuum: vacuumShare,
	}

	RegisterRoutes(api, share, cfg)

	return share
}
