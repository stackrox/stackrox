package postgres

import (
	"context"

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

// Connect wraps pgxpool.connect
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

// Conn is a wrapper around pgxpool.Conn
type Conn struct {
	*pgxpool.Conn
}

// Begin wraps conn.Begin
func (c *Conn) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := c.Conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &txWrapper{
		Tx: tx,
	}, nil
}

type txWrapper struct {
	pgx.Tx
}
