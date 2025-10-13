//go:build scanner_db_integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quay/claircore/datastore/postgres"
	"github.com/stretchr/testify/require"
)

func testDB(t *testing.T, ctx context.Context, name string) *pgxpool.Pool {
	t.Helper()

	pgConn := "postgresql://postgres@127.0.0.1:5432/postgres?sslmode=disable"
	pgPool, err := postgres.Connect(ctx, pgConn, name)
	require.NoError(t, err)
	createDatabase := `CREATE DATABASE ` + name
	_, err = pgPool.Exec(ctx, createDatabase)
	require.NoError(t, err)
	t.Cleanup(pgPool.Close)

	dbConn := fmt.Sprintf("postgresql://postgres@127.0.0.1:5432/%s?sslmode=disable", name)
	dbPool, err := postgres.Connect(ctx, dbConn, name)
	require.NoError(t, err)
	t.Cleanup(func() {
		dropDatabase := `DROP DATABASE IF EXISTS ` + name
		_, _ = pgPool.Exec(ctx, dropDatabase)
	})
	t.Cleanup(dbPool.Close)

	return dbPool
}
