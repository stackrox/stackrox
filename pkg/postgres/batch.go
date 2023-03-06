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

	return b.BatchResults.Close()
}
