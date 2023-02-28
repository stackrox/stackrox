package postgres

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Conn is a wrapper around pgxpool.Conn
type Conn struct {
	*pgxpool.Conn
}

// Begin wraps pgxpool.Conn Begin
func (c *Conn) Begin(ctx context.Context) (*Tx, error) {
	tx, err := c.Conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &Tx{
		Tx: tx,
	}, nil
}

// Exec wraps pgxpool.Conn Exec
func (c *Conn) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return c.Conn.Exec(ctx, sql, args...)
}

// Query wraps pgxpool.Conn Query
func (c *Conn) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return c.Conn.Query(ctx, sql, args...)
}

// QueryRow wraps pgxpool.Conn QueryRow
func (c *Conn) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return c.Conn.QueryRow(ctx, sql, args...)
}

// SendBatch wraps pgxpool.Conn SendBatch
func (c *Conn) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return c.Conn.SendBatch(ctx, b)
}
