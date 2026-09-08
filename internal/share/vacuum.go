package share

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

type Vacuum struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewVacuum(db *sql.DB) *Vacuum {
	return &Vacuum{
		db:     db,
		logger: slog.Default().With("worker", "vacuum"),
	}
}

// Exec periodically removes expired shares until ctx is canceled.
func (v *Vacuum) Exec(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	v.logger.Info("worker started", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			v.logger.Debug("vacuum tick")

			count, err := v.deleteExpired(ctx, time.Now().UTC())
			if err != nil {
				v.logger.Error("delete expired shares", "error", err)
				continue
			}
			if count > 0 {
				v.logger.Info("deleted expired shares", "count", count)
			}
		}
	}
}

func (v *Vacuum) deleteExpired(ctx context.Context, now time.Time) (uint, error) {
	result, err := v.db.ExecContext(ctx, `DELETE FROM shares WHERE expires_at <= ?`, now.Unix())
	if err != nil {
		return 0, fmt.Errorf("delete expired shares: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read affected rows: %w", err)
	}

	return uint(count), nil
}
