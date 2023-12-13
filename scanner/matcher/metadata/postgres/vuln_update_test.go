//go:build scanner_integration

package postgres

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/quay/claircore/datastore/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testCtx = context.Background()

func initDB(t *testing.T, name string) (MetadataStore, *pgxpool.Pool, *pgxpool.Pool) {
	pgConn := "postgresql://postgres@127.0.0.1:5432/postgres?sslmode=disable"
	pgPool, err := postgres.Connect(testCtx, pgConn, name)
	require.NoError(t, err)
	createDatabase := `CREATE DATABASE ` + name
	_, err = pgPool.Exec(testCtx, createDatabase)
	require.NoError(t, err)

	dbConn := fmt.Sprintf("postgresql://postgres@127.0.0.1:5432/%s?sslmode=disable", name)
	dbPool, err := postgres.Connect(testCtx, dbConn, name)
	require.NoError(t, err)
	store, err := InitPostgresMetadataStore(testCtx, dbPool, true)
	require.NoError(t, err)
	return store, pgPool, dbPool
}

func dropDB(t *testing.T, name string, pool *pgxpool.Pool) {
	dropDatabase := `DROP DATABASE IF EXISTS ` + name
	_, err := pool.Exec(testCtx, dropDatabase)
	require.NoError(t, err)
}

func TestVulnUpdateStore(t *testing.T) {
	store, pgPool, dbPool := initDB(t, "vuln_update_test")
	defer dropDB(t, "vuln_update_test", pgPool)
	defer dbPool.Close()

	// Initial timestamp should be "empty"
	timestamp, err := store.GetLastVulnerabilityUpdate(testCtx)
	require.NoError(t, err)
	assert.Equal(t, time.Time{}, timestamp)

	now, err := http.ParseTime(time.Now().UTC().Format(http.TimeFormat))
	require.NoError(t, err)

	err = store.SetLastVulnerabilityUpdate(testCtx, now)
	require.NoError(t, err)
	timestamp, err = store.GetLastVulnerabilityUpdate(testCtx)
	require.NoError(t, err)
	assert.Equal(t, now, timestamp)
}
