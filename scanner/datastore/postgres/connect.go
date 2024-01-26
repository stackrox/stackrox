package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/retry"
)

const (
	connTries = 30
	interval  = 10 * time.Second
)

// Connect is a wrapper around ClairCore's postgres.Connect which retries the connection upon failure.
//
// At this time, this function does 30 attempts with a 10-second interval between attempts.
func Connect(ctx context.Context, connString string, applicationName string) (pool *pgxpool.Pool, err error) {
	err = retry.WithRetry(func() error {
		pool, err = postgres.Connect(ctx, connString, applicationName)
		return err
	}, retry.Tries(connTries), retry.OnFailedAttempts(func(err error) {
		zlog.Error(ctx).Err(err).Msg("failed to connect to postgres database")
	}), retry.BetweenAttempts(func(previousAttemptNumber int) {
		zlog.Warn(ctx).Int("attempt", previousAttemptNumber+1).Msg("retrying connection to postgres database")
		time.Sleep(interval)
	}))
	if err != nil {
		return nil, err
	}
	return pool, nil
}
