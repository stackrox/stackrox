package postgres

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// ConnectConfig wraps pgxpool.Pool with the Postgres struct
func ConnectConfig(ctx context.Context, config *pgxpool.Config) (*Postgres, error) {
	pool, err := pgxpool.ConnectConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return &Postgres{
		Pool: pool,
	}, nil
}

// Connect wraps pgxpool.Pool with the Postgres struct
func Connect(ctx context.Context, connString string) (*Postgres, error) {
	pool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		return nil, err
	}
	return &Postgres{
		Pool: pool,
	}, nil
}

// Postgres is a wrapper around access to the database
type Postgres struct {
	*pgxpool.Pool
}

func (p *Postgres) Begin(ctx context.Context) (pgx.Tx, error) {

	return p.Pool.Begin(ctx)
}

func (p *Postgres) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {

	return p.Pool.Query(ctx, sql, args...)
}

func (p *Postgres) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {

	return p.Pool.QueryRow(ctx, sql, args...)
}

func (p *Postgres) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {

	return p.Pool.Exec(ctx, sql, args...)
}
