package postgres

import (
	"context"
)

// DB is an interface to interact with database.
//
//go:generate mockgen-wrapper
type DB interface {
	Executable
	Queryable
	Begin(ctx context.Context) (*Tx, error)
	Acquire(ctx context.Context) (*Conn, error)
	Config() *Config
	Ping(ctx context.Context) error
	Close()
}
