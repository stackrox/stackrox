package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxRetries      = 30
	retryInterval   = 10 * time.Second
	initialConnWait = 5 * time.Second
)

// Connect establishes a connection pool to PostgreSQL with retry logic.
// It sets the application_name runtime parameter and verifies connectivity with a ping.
func Connect(ctx context.Context, connString string, applicationName string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Set application name for connection tracking
	config.ConnConfig.RuntimeParams["application_name"] = applicationName

	var pool *pgxpool.Pool
	var lastErr error

	// Allow initial connection delay
	time.Sleep(initialConnWait)

	for attempt := range maxRetries {
		pool, err = pgxpool.NewWithConfig(ctx, config)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed to create pool: %w", attempt+1, err)
			slog.Warn("Failed to create connection pool", "attempt", attempt+1, "error", err)

			if attempt < maxRetries-1 {
				time.Sleep(retryInterval)
			}
			continue
		}

		// Verify connection with ping
		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			lastErr = fmt.Errorf("attempt %d failed to ping database: %w", attempt+1, err)
			slog.Warn("Failed to ping database", "attempt", attempt+1, "error", err)

			if attempt < maxRetries-1 {
				time.Sleep(retryInterval)
			}
			continue
		}

		slog.Info("Successfully connected to PostgreSQL",
			"attempt", attempt+1,
			"application_name", applicationName)
		return pool, nil
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, lastErr)
}
