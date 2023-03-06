package postgres

import (
	"github.com/jackc/pgx/v4"
)

// Rows wraps pgx.Rows
type Rows struct {
	rowsScanned int
	pgx.Rows
}

// Close wraps pgx.Rows Close
func (r *Rows) Close() {
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
