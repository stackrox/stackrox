package postgres

import (
	"context"

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
		return err
	}
	return nil
}
