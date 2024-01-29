package pgtest

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/random"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"k8s.io/utils/env"
)

const driverName = "pgx"

// TestPostgres is a Postgres instance used in tests
type TestPostgres struct {
	postgres.DB
	database string
}

// CreateADatabaseForT creates a postgres database for test
func CreateADatabaseForT(t testing.TB) string {
	suffix, err := random.GenerateString(5, random.AlphanumericCharacters)
	require.NoError(t, err)

	database := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_") + suffix)
	database = strings.ToLower(strings.ReplaceAll(database, "-", "_"))

	CreateDatabase(t, database)

	return database
}

// CreateDatabase - creates a database for testing
func CreateDatabase(t testing.TB, database string) {
	// Bootstrap the test database by connecting to the default postgres database and running create
	sourceWithPostgresDatabase := conn.GetConnectionStringWithDatabaseName(t, "postgres")

	db, err := sql.Open(driverName, sourceWithPostgresDatabase)
	require.NoError(t, err)

	// Checks to see if DB already exists
	existsStmt := "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_database WHERE datname = $1)"

	row := db.QueryRow(existsStmt, database)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		exists = false
	}

	// Only create the test DB if it does not exist
	if !exists {
		_, err = db.Exec("CREATE DATABASE " + database)
		require.NoError(t, err)
	}
	require.NoError(t, db.Close())
}

// DropDatabase - drops the database specified from the testing scope
func DropDatabase(t testing.TB, database string) {
	// Connect to the admin postgres database to drop the test database.
	if database != "postgres" {
		sourceWithPostgresDatabase := conn.GetConnectionStringWithDatabaseName(t, "postgres")
		db, err := sql.Open(driverName, sourceWithPostgresDatabase)
		require.NoError(t, err)

		_, _ = db.Exec("DROP DATABASE " + database)
		require.NoError(t, db.Close())
	}
}

// ForT creates and returns a Postgres for the test
func ForT(t testing.TB) *TestPostgres {
	// Bootstrap a test database
	database := CreateADatabaseForT(t)

	sourceWithDatabase := conn.GetConnectionStringWithDatabaseName(t, database)

	CreateDatabase(t, database)

	// Create all the tables for the database
	gormDB := OpenGormDB(t, sourceWithDatabase)
	pkgSchema.ApplyAllSchemasIncludingTests(context.Background(), gormDB, t)
	CloseGormDB(t, gormDB)

	// initialize pool to be used
	pool := ForTCustomPool(t, database)

	return &TestPostgres{
		DB:       pool,
		database: database,
	}
}

// ForTCustomDB - creates and returns a Postgres for the test.  This is used primarily in testing migrations,
// so we do not want to run Gorm to create the schemas as the clone management will do that.
func ForTCustomDB(t testing.TB, dbName string) *TestPostgres {
	database := strings.ToLower(dbName)

	// Bootstrap the test database by connecting to the default postgres database and running create
	CreateDatabase(t, database)

	// initialize pool to be used
	pool := ForTCustomPool(t, dbName)

	return &TestPostgres{
		DB:       pool,
		database: database,
	}
}

// ForTCustomPool - gets a connection pool to a specific database.
func ForTCustomPool(t testing.TB, dbName string) postgres.DB {
	sourceWithDatabase := conn.GetConnectionStringWithDatabaseName(t, dbName)
	ctx := context.Background()

	// initialize pool to be used
	pool, err := postgres.Connect(ctx, sourceWithDatabase)
	require.NoError(t, err)

	return pool
}

// GetGormDB opens a Gorm DB to the Postgres DB
func (tp *TestPostgres) GetGormDB(t testing.TB) *gorm.DB {
	source := conn.GetConnectionStringWithDatabaseName(t, tp.database)
	return OpenGormDB(t, source)
}

// Teardown tears down a Postgres instance used in tests
func (tp *TestPostgres) Teardown(t testing.TB) {
	if tp == nil {
		return
	}
	tp.Close()

	DropDatabase(t, tp.database)
}

// GetConnectionString returns a connection string for integration testing with Postgres
func GetConnectionString(t testing.TB) string {
	return conn.GetConnectionStringWithDatabaseName(t, env.GetString("POSTGRES_DB", "postgres"))
}

// GetConnectionStringWithDatabaseName returns a connection string for integration testing with Postgres
func GetConnectionStringWithDatabaseName(t testing.TB, database string) string {
	return conn.GetConnectionStringWithDatabaseName(t, database)
}

// OpenGormDB opens a Gorm DB to the Postgres DB
func OpenGormDB(t testing.TB, source string) *gorm.DB {
	return conn.OpenGormDB(t, source, false)
}

// CloseGormDB closes connection to a Gorm DB
func CloseGormDB(t testing.TB, db *gorm.DB) {
	conn.CloseGormDB(t, db)
}

// SkipIfPostgresEnabled skips the tests if the Postgres flag is on
func SkipIfPostgresEnabled(t testing.TB) {
	t.Skip("Skipping test because Postgres is enabled")
	t.SkipNow()
}
