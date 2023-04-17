package postgres

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

// Tx wraps pgx.Tx
type Tx struct {
	pgx.Tx
	cancelFunc context.CancelFunc
}

// Exec wraps pgx.Tx Exec
func (t *Tx) Exec(ctx context.Context, sql string, args ...interface{}) (commandTag pgconn.CommandTag, err error) {
	ct, err := t.Tx.Exec(ctx, sql, args...)
	if err != nil {
		incQueryErrors(sql, err)
		return nil, err
	}
	return ct, err
}

// Query wraps pgx.Tx Query
func (t *Tx) Query(ctx context.Context, sql string, args ...interface{}) (*Rows, error) {
	rows, err := t.Tx.Query(ctx, sql, args...)
	if err != nil {
		incQueryErrors(sql, err)
		return nil, err
	}
	return &Rows{
		cancelFunc: func() {},
		query:      sql,
		Rows:       rows,
	}, nil
}

// QueryRow wraps pgx.Tx QueryRow
func (t *Tx) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return t.Tx.QueryRow(ctx, sql, args...)
}

// Commit wraps pgx.Tx Commit
func (t *Tx) Commit(ctx context.Context) error {
	defer t.cancelFunc()

	if err := t.Tx.Commit(ctx); err != nil {
		incQueryErrors("commit", err)
		return err
	}
	return nil
}

// Rollback wraps pgx.Tx Rollback
func (t *Tx) Rollback(ctx context.Context) error {
	defer t.cancelFunc()

	if err := t.Tx.Rollback(ctx); err != nil {
		incQueryErrors("rollback", err)
		return err
	}
	return nil
}
