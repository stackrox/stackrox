package postgres

import (
	"context"
	"runtime/debug"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/contextutil"
)

var query = "select distinct(policy_categories.Id), policy_categories.Name as policy_category from policy_categories inner join policy_category_edges on policy_categories.Id = policy_category_edges.CategoryId where policy_category_edges.PolicyId"

// New creates a new DB wrapper
func New(ctx context.Context, config *Config) (*db, error) {
	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.ConnectConfig(ctx, config.Config)
	if err != nil {
		incQueryErrors("connect", err)
		return nil, err
	}
	return &db{
		Pool: pool,
	}, nil
}

// ParseConfig wraps pgxpool.ParseConfig
func ParseConfig(source string) (*Config, error) {
	config, err := pgxpool.ParseConfig(source)
	if err != nil {
		return nil, err
	}
	return &Config{Config: config}, nil
}

// Connect wraps pgxpool.Connect
func Connect(ctx context.Context, sourceWithDatabase string) (*db, error) {
	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.Connect(ctx, sourceWithDatabase)
	if err != nil {
		incQueryErrors("connect", err)
		return nil, err
	}
	return &db{Pool: pool}, nil
}

// db wraps pgxpool.Pool
type db struct {
	*pgxpool.Pool
}

// Begin wraps pgxpool.Pool Begin
func (d *db) Begin(ctx context.Context) (*Tx, error) {
	if tx, ok := TxFromContext(ctx); ok {
		return &Tx{
			Tx:         tx.Tx,
			cancelFunc: tx.cancelFunc,
			mode:       inner,
		}, nil
	}

	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, defaultTimeout)

	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		incQueryErrors("begin", err)
		return nil, err
	}
	return &Tx{
		Tx:         tx,
		cancelFunc: cancel,
	}, nil
}

// Exec wraps pgxpool.Pool Exec
func (d *db) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, defaultTimeout)
	defer cancel()

	var err error
	var ct pgconn.CommandTag

	tx, ok := TxFromContext(ctx)
	if ok {
		ct, err = tx.Exec(ctx, sql, args...)
	} else {
		ct, err = d.Pool.Exec(ctx, sql, args...)
	}
	if err != nil {
		incQueryErrors(sql, err)
		return nil, toErrox(err)
	}
	return ct, nil
}

// Query wraps pgxpool.Pool Query
func (d *db) Query(ctx context.Context, sql string, args ...interface{}) (*Rows, error) {
	if strings.Contains(sql, query) {
		debug.PrintStack()
	}
	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, defaultTimeout)
	rows := &Rows{
		query:      sql,
		cancelFunc: cancel,
	}
	var err error
	if tx, ok := TxFromContext(ctx); ok {
		rows.Rows, err = tx.Query(ctx, sql, args...)
	} else {
		rows.Rows, err = d.Pool.Query(ctx, sql, args...)
	}

	if err != nil {
		incQueryErrors(sql, err)
		return nil, err
	}
	return rows, nil
}

// QueryRow wraps pgxpool.Pool QueryRow
func (d *db) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, defaultTimeout)

	if strings.Contains(sql, query) {
		debug.PrintStack()
	}

	row := &Row{
		query:      sql,
		cancelFunc: cancel,
	}

	if tx, ok := TxFromContext(ctx); ok {
		row.Row = tx.QueryRow(ctx, sql, args...)
	} else {
		row.Row = d.Pool.QueryRow(ctx, sql, args...)
	}
	return row
}

// Acquire wraps pgxpool.Acquire
func (d *db) Acquire(ctx context.Context) (*Conn, error) {
	conn, err := d.Pool.Acquire(ctx)
	if err != nil {
		incQueryErrors("acquire", err)
		return nil, err
	}
	return &Conn{PgxPoolConn: conn}, nil
}

// Config wraps pgxpool.Config
func (d *db) Config() *Config {
	return &Config{
		Config: d.Pool.Config(),
	}
}

// Config is a wrapper around pgxpool.Config
type Config struct {
	*pgxpool.Config
}

// Copy is a wrapper around pgx.Config Copy
func (c *Config) Copy() *Config {
	return &Config{
		Config: c.Config.Copy(),
	}
}
