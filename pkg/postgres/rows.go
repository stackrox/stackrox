package postgres

import (
	"context"

	"github.com/jackc/pgx/v4"
)

// Row wraps pgx.Row
type Row struct {
	pgx.Row
	cancelFunc context.CancelFunc
}

// Scan wraps pgx.Row Scan
func (r *Row) Scan(dest ...interface{}) error {
	defer r.cancelFunc()

	return r.Row.Scan(dest...)
}

// Rows wraps pgx.Rows
type Rows struct {
	rowsScanned int
	pgx.Rows
	cancelFunc context.CancelFunc
}

// Close wraps pgx.Rows Close
func (r *Rows) Close() {
	defer r.cancelFunc()

	// Eventually extended for metrics to report number of returned rows
	r.Rows.Close()
}

// Scan wraps pgx.Rows Scan
func (r *Rows) Scan(dest ...interface{}) error {
	err := r.Rows.Scan(dest...)
	if err != nil {
		return err
	}
	r.rowsScanned++
	return nil
}
