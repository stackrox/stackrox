package postgres

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quay/claircore/datastore/postgres"
	"github.com/stackrox/rox/pkg/retry"
)

const (
	connTries = 30
	interval  = 10 * time.Second
)

// Connect is a wrapper around ClairCore's postgres.Connect which retries the connection upon failure.
//
// At this time, this function does 30 attempts with a 10-second interval between attempts.
func Connect(ctx context.Context, connString string, applicationName string) (*pgxpool.Pool, error) {
	// ClairCore's postgres.Connect uses pgxpool.New (pgx v5), which does not
	// eagerly establish connections.
	pool, err := postgres.Connect(ctx, connString, applicationName)
	if err != nil {
		return nil, err
	}

	// Ping explicitly to verify DB reachability, with retries for the case where
	// the DB is not yet ready to accept connections.
	err = retry.WithRetry(func() error {
		return pool.Ping(ctx)
	}, retry.Tries(connTries), retry.OnFailedAttempts(func(err error) {
		slog.ErrorContext(ctx, "failed to connect to postgres database", "reason", err)
	}), retry.BetweenAttempts(func(previousAttemptNumber int) {
		slog.WarnContext(ctx, "retrying connection to postgres database", "attempt", previousAttemptNumber+1)
		time.Sleep(interval)
	}))
	if err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
