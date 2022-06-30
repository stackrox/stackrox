package pgtest

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/random"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"k8s.io/utils/env"

	// unknow reason
	_ "github.com/lib/pq"
)

// TestPostgres refers to Postgres database for testing
type TestPostgres struct {
	*pgxpool.Pool
	database string
}

func createDatabase(t testing.TB, database string) {
	// Bootstrap the test database by connecting to the default postgres database and running create
	sourceWithPostgresDatabase := conn.GetConnectionStringWithDatabaseName("postgres")
	db, err := sql.Open("postgres", sourceWithPostgresDatabase)
	require.NoError(t, err)

	_, err = db.Exec("CREATE DATABASE " + database)
	require.NoError(t, err)
	require.NoError(t, db.Close())
}

func dropDatabase(t testing.TB, database string) {
	// Bootstrap the test database by connecting to the default postgres database and running create
	sourceWithPostgresDatabase := conn.GetConnectionStringWithDatabaseName("postgres")
	db, err := sql.Open("postgres", sourceWithPostgresDatabase)
	require.NoError(t, err)

	_, err = db.Exec("DROP DATABASE " + database)
	require.NoError(t, err)
	require.NoError(t, db.Close())
}

// ForT creates and returns a Postgres for the test
func ForT(t testing.TB) *TestPostgres {
	suffix, err := random.GenerateString(5, random.AlphanumericCharacters)
	require.NoError(t, err)

	database := strings.ToLower(t.Name() + suffix)

	// Bootstrap the test database by connecting to the default postgres database and running create
	createDatabase(t, database)

	sourceWithDatabase := conn.GetConnectionStringWithDatabaseName(database)
	ctx := context.Background()

	// Create all the tables for the database
	gormDB := OpenGormDB(t, sourceWithDatabase)
	pkgSchema.ApplyAllSchemasIncludingTests(context.Background(), gormDB, t)
	CloseGormDB(t, gormDB)

	// initialize pool to be used
	pool, err := pgxpool.Connect(ctx, sourceWithDatabase)
	require.NoError(t, err)

	return &TestPostgres{
		Pool:     pool,
		database: database,
	}
}

// Teardown tears down a Postgres instance used in tests
func (tp *TestPostgres) Teardown(t testing.TB) {
	if tp == nil {
		return
	}
	tp.Close()
	dropDatabase(t, tp.database)
}

// GetConnectionString returns a connection string for integration testing with Postgres
func GetConnectionString(_ *testing.T) string {
	return conn.GetConnectionStringWithDatabaseName(env.GetString("POSTGRES_DB", "postgres"))
}

// OpenGormDB opens a Gorm DB to the Postgres DB
func OpenGormDB(t testing.TB, source string) *gorm.DB {
	return conn.OpenGormDB(t, source, false)
}

// OpenGormDBWithDisabledConstraints
func OpenGormDBWithDisabledConstraints(t testing.TB, source string) *gorm.DB {
	return conn.OpenGormDB(t, source, true)
}

// CloseGormDB closes connection to a Gorm DB
func CloseGormDB(t testing.TB, db *gorm.DB) {
	conn.CloseGormDB(t, db)
}

// CleanUpDB removes public schema together with all tables
func CleanUpDB(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	conn.CleanUpDB(ctx, t, pool)
}
