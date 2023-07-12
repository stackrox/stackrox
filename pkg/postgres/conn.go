package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stackrox/rox/pkg/contextutil"
)

// Conn is a wrapper around pgxpool.Conn
type Conn struct {
	PgxPoolConn
}

// Release wraps pgxpool.Conn Release
func (c *Conn) Release() {
	if c != nil {
		c.PgxPoolConn.Release()
	}
}

// Begin wraps pgxpool.Conn Begin
func (c *Conn) Begin(ctx context.Context) (*Tx, error) {
	if tx, ok := TxFromContext(ctx); ok {
		return &Tx{
			Tx:         tx.Tx,
			cancelFunc: tx.cancelFunc,
			mode:       inner,
		}, nil
	}

	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, defaultTimeout)

	tx, err := c.PgxPoolConn.Begin(ctx)
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
		ct, err = c.PgxPoolConn.Exec(ctx, sql, args...)
	}
	if err != nil {
		incQueryErrors(sql, err)
		return pgconn.CommandTag{}, toErrox(err)
	}
	return ct, nil
}

// Query wraps pgxpool.Conn Query
func (c *Conn) Query(ctx context.Context, sql string, args ...interface{}) (*Rows, error) {
	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, defaultTimeout)

	if tx, ok := TxFromContext(ctx); ok {
		rows, err := tx.Query(ctx, sql, args...)
		incQueryErrors(sql, err)
		return rows, err
	}

	rows, err := c.PgxPoolConn.Query(ctx, sql, args...)
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
		row = c.PgxPoolConn.QueryRow(ctx, sql, args...)
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
		batchResults = c.PgxPoolConn.SendBatch(ctx, b)
	}

	return &BatchResults{
		BatchResults: batchResults,
		cancel:       cancel,
	}
}

// CopyFrom wraps pgxpool.Conn CopyFrom
func (c *Conn) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (rows int64, err error) {
	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, defaultTimeout)
	defer cancel()

	if tx, ok := TxFromContext(ctx); ok {
		rows, err = tx.CopyFrom(ctx, tableName, columnNames, rowSrc)
	} else {
		rows, err = c.PgxPoolConn.CopyFrom(ctx, tableName, columnNames, rowSrc)
	}
	incQueryErrors("copyfrom", err)
	return rows, err
}
