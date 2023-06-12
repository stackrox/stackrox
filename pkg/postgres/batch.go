package postgres

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
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
		return nil, toErrox(err)
	}
	return ct, err
}
