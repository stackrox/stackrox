package postgres

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/contextutil"
)

// Conn is a wrapper around pgxpool.Conn
type Conn struct {
	*pgxpool.Conn
}

// Release wraps pgxpool.Conn Release
func (c *Conn) Release() {
	if c != nil {
		c.Conn.Release()
	}
}

// Begin wraps pgxpool.Conn Begin
func (c *Conn) Begin(ctx context.Context) (*Tx, error) {
	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, defaultTimeout)

	tx, err := c.Conn.Begin(ctx)
	if err != nil {
		incQueryErrors("begin", err)
		return nil, err
	}
	return &Tx{
		Tx:         tx,
		cancelFunc: cancel,
	}, nil
}

// Exec wraps pgxpool.Conn Exec
func (c *Conn) Exec(ctx context.Context, sql string, args ...interface{}) (ct pgconn.CommandTag, err error) {
	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, defaultTimeout)
	defer cancel()

	tx, ok := TxFromContext(ctx)
	if ok {
		ct, err = tx.Exec(ctx, sql, args...)
	} else {
		ct, err = c.Conn.Exec(ctx, sql, args...)
	}

	if err != nil {
		incQueryErrors(sql, err)
		return nil, err
	}
	return ct, err
}

// Query wraps pgxpool.Conn Query
func (c *Conn) Query(ctx context.Context, sql string, args ...interface{}) (*Rows, error) {
	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, defaultTimeout)

	if tx, ok := TxFromContext(ctx); ok {
		rows, err := tx.Query(ctx, sql, args...)
		if err != nil {
			incQueryErrors(sql, err)
		}
		return rows, err
	}

	rows, err := c.Conn.Query(ctx, sql, args...)
	if err != nil {
		incQueryErrors(sql, err)
		return nil, err
	}

	return &Rows{
		Rows:       rows,
		query:      sql,
		cancelFunc: cancel,
	}, nil
}

// QueryRow wraps pgxpool.Conn QueryRow
func (c *Conn) QueryRow(ctx context.Context, sql string, args ...interface{}) *Row {
	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, defaultTimeout)

	var row pgx.Row
	if tx, ok := TxFromContext(ctx); ok {
		row = tx.QueryRow(ctx, sql, args...)
	} else {
		row = c.Conn.QueryRow(ctx, sql, args...)
	}

	return &Row{
		Row:        row,
		query:      sql,
		cancelFunc: cancel,
	}
}

// SendBatch wraps pgxpool.Conn SendBatch
func (c *Conn) SendBatch(ctx context.Context, b *pgx.Batch) *BatchResults {
	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, defaultTimeout)

	var batchResults pgx.BatchResults
	if tx, ok := TxFromContext(ctx); ok {
		batchResults = tx.SendBatch(ctx, b)
	} else {
		batchResults = c.Conn.SendBatch(ctx, b)
	}

	return &BatchResults{
		BatchResults: batchResults,
		cancel:       cancel,
	}
}

// CopyFrom wraps pgxpool.Conn CopyFrom
func (c *Conn) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, defaultTimeout)
	defer cancel()

	if tx, ok := TxFromContext(ctx); ok {
		return tx.CopyFrom(ctx, tableName, columnNames, rowSrc)
	}
	return c.Conn.CopyFrom(ctx, tableName, columnNames, rowSrc)
}
