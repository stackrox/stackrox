package postgres

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
)

// Postgres is a wrapper around access to the database
type Postgres struct {
	*pgxpool.Pool
}

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
