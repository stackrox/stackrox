package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DB is an interface to interact with database.
//
//go:generate mockgen-wrapper
type DB interface {
	Begin(ctx context.Context) (*Tx, error)
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (*Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Acquire(ctx context.Context) (*Conn, error)
	Config() *Config
	Ping(ctx context.Context) error
	Close()
}
