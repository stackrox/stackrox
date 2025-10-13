package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// BatchResults wraps pgx.BatchResults
type BatchResults struct {
	pgx.BatchResults
	cancel context.CancelFunc
}

// Close wraps pgx.BatchResults Close
func (b *BatchResults) Close() error {
	defer b.cancel()

	if err := b.BatchResults.Close(); err != nil {
		incQueryErrors("batch", err)
		return toErrox(err)
	}
	return nil
}

// Exec wraps pgx.BatchResults Exec
func (b *BatchResults) Exec() (pgconn.CommandTag, error) {
	ct, err := b.BatchResults.Exec()
	if err != nil {
		incQueryErrors("batch", err)
		return pgconn.CommandTag{}, toErrox(err)
	}
	return ct, err
}
