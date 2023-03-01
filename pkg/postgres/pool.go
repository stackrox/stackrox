package postgres

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// New creates a new DB wrapper
func New(ctx context.Context, config *Config) (*DB, error) {
	pool, err := pgxpool.ConnectConfig(ctx, config.Config)
	if err != nil {
		return nil, err
	}
	return &DB{
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
func Connect(ctx context.Context, sourceWithDatabase string) (*DB, error) {
	pool, err := pgxpool.Connect(ctx, sourceWithDatabase)
	if err != nil {
		return nil, err
	}
	return &DB{Pool: pool}, nil
}

// DB wraps pgxpool.Pool
type DB struct {
	*pgxpool.Pool
}

// Begin wraps pgxpool.Pool Begin
func (d *DB) Begin(ctx context.Context) (*Tx, error) {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &Tx{
		Tx: tx,
	}, nil
}

// Exec wraps pgxpool.Pool Exec
func (d *DB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return d.Pool.Exec(ctx, sql, args...)
}

// Query wraps pgxpool.Pool Query
func (d *DB) Query(ctx context.Context, sql string, args ...interface{}) (*Rows, error) {
	rows, err := d.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &Rows{
		Rows: rows,
	}, nil
}

// QueryRow wraps pgxpool.Pool QueryRow
func (d *DB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return d.Pool.QueryRow(ctx, sql, args...)
}

// Acquire wraps pgxpool.Acquire
func (d *DB) Acquire(ctx context.Context) (*Conn, error) {

	conn, err := d.Pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return &Conn{Conn: conn}, nil
}

// Config wraps pgxpool.Config
func (d *DB) Config() *Config {
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
