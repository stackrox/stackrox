package postgres

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

// Tx wraps pgx.Tx
type Tx struct {
	pgx.Tx
}

// Exec wraps pgx.Tx Exec
func (t *Tx) Exec(ctx context.Context, sql string, args ...interface{}) (commandTag pgconn.CommandTag, err error) {
	return t.Tx.Exec(ctx, sql, args...)
}

// Query wraps pgx.Tx Query
func (t *Tx) Query(ctx context.Context, sql string, args ...interface{}) (*Rows, error) {
	rows, err := t.Tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &Rows{
		Rows: rows,
	}, nil
}

// QueryRow wraps pgx.Tx QueryRow
func (t *Tx) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return t.Tx.QueryRow(ctx, sql, args...)
}
