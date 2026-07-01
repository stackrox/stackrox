package pgtest

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/random"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	driverName = "pgx"

	// defaultDatabaseName is needed to create and drop databases. Without it we can't create or drop databases, it is a catch-22
	// because a database is needed for the connection.
	defaultDatabaseName = "postgres"

	templateDBName = "test_template"
)

// TestPostgres is a Postgres instance used in tests
type TestPostgres struct {
	postgres.DB
	database string
}

// CreateADatabaseForT creates a postgres database for test
func CreateADatabaseForT(t testing.TB) string {
	database := generateDatabaseName(t)
	CreateDatabase(t, database)
	return database
}

// CreateDatabase - creates a database for testing
func CreateDatabase(t testing.TB, database string) {
	// Bootstrap the test database by connecting to the default postgres database and running create
	sourceWithPostgresDatabase := conn.GetConnectionStringWithDatabaseName(t, defaultDatabaseName)

	db, err := sql.Open(driverName, sourceWithPostgresDatabase)
	require.NoError(t, err)

	// Checks to see if DB already exists
	existsStmt := "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_database WHERE datname = $1)"

	row := db.QueryRow(existsStmt, database)
	var exists bool
	err = row.Scan(&exists)
	require.NoError(t, err)

	// Only create the test DB if it does not exist
	if !exists {
		_, err = db.Exec("CREATE DATABASE " + pq.QuoteIdentifier(database))
		require.NoError(t, err)
	}
	require.NoError(t, db.Close())
}

// DropDatabase - drops the database specified from the testing scope
func DropDatabase(t testing.TB, database string) {
	// Connect to the admin postgres database to drop the test database.
	if database != defaultDatabaseName {
		sourceWithPostgresDatabase := conn.GetConnectionStringWithDatabaseName(t, defaultDatabaseName)
		db, err := sql.Open(driverName, sourceWithPostgresDatabase)
		require.NoError(t, err)

		_, _ = db.Exec("DROP DATABASE " + pq.QuoteIdentifier(database))
		require.NoError(t, db.Close())
	}
}

func ensureTemplateDB(t testing.TB) {
	sourceWithPostgresDatabase := conn.GetConnectionStringWithDatabaseName(t, defaultDatabaseName)
	db, err := sql.Open(driverName, sourceWithPostgresDatabase)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()

	// Serialize across concurrent test binaries.
	_, err = db.Exec("SELECT pg_advisory_lock(1)")
	require.NoError(t, err)
	defer func() { _, _ = db.Exec("SELECT pg_advisory_unlock(1)") }()

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_database WHERE datname = $1)", templateDBName).Scan(&exists)
	require.NoError(t, err)
	// CI Postgres is ephemeral per job (fresh runner VM or docker run --rm), so an
	// existing template is always from the current schema in this job.
	if exists {
		t.Log("reusing existing test template database")
		return
	}

	t.Log("creating test template database with all schemas")
	_, err = db.Exec("CREATE DATABASE " + pq.QuoteIdentifier(templateDBName))
	require.NoError(t, err)

	// Drop the template if schema population fails so the next caller
	// does not clone an incomplete database.
	success := false
	defer func() {
		if !success {
			_, _ = db.Exec("DROP DATABASE IF EXISTS " + pq.QuoteIdentifier(templateDBName))
		}
	}()

	src := conn.GetConnectionStringWithDatabaseName(t, templateDBName)
	gormDB := OpenGormDB(t, src)
	pkgSchema.ApplyAllSchemasIncludingTests(context.Background(), gormDB, t)
	CloseGormDB(t, gormDB)
	success = true
}

func createDatabaseFromTemplate(t testing.TB, database string) {
	sourceWithPostgresDatabase := conn.GetConnectionStringWithDatabaseName(t, defaultDatabaseName)
	db, err := sql.Open(driverName, sourceWithPostgresDatabase)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()

	_, err = db.Exec("CREATE DATABASE " + pq.QuoteIdentifier(database) + " TEMPLATE " + pq.QuoteIdentifier(templateDBName))
	require.NoError(t, err)
}

func generateDatabaseName(t testing.TB) string {
	suffix := random.GenerateString(5, random.AlphanumericCharacters)

	h := fnv.New64a()
	_, err := io.WriteString(h, t.Name())
	require.NoError(t, err)

	return fmt.Sprintf("%x_%s", h.Sum64(), suffix)
}

// ForT creates and returns a Postgres for the test
// It will teardown DB at the end of the test.
func ForT(t testing.TB) *TestPostgres {
	var database string

	if testutils.IsRunningInCI() {
		ensureTemplateDB(t)
		database = generateDatabaseName(t)
		t.Log("cloning test database from template")
		createDatabaseFromTemplate(t, database)
	} else {
		// Bootstrap a test database
		database = CreateADatabaseForT(t)

		sourceWithDatabase := conn.GetConnectionStringWithDatabaseName(t, database)

		// Create all the tables for the database
		gormDB := OpenGormDB(t, sourceWithDatabase)
		pkgSchema.ApplyAllSchemasIncludingTests(context.Background(), gormDB, t)
		CloseGormDB(t, gormDB)
	}

	// initialize pool to be used
	pool := ForTCustomPool(t, database)
	require.NoError(t, pkgSchema.ApplyAllIndexes(context.Background(), pool, 30*time.Second))

	testPg := &TestPostgres{
		DB:       pool,
		database: database,
	}

	t.Cleanup(func() {
		testPg.teardown(t)
	})

	return testPg
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
	sourceWithDatabase := conn.GetSingleConnectionString(t, dbName)
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

func (tp *TestPostgres) teardown(t testing.TB) {
	if tp == nil {
		return
	}
	tp.Close()

	if !env.PostgresKeepTestDB.BooleanSetting() {
		DropDatabase(t, tp.database)
	}
}

// GetConnectionString returns a connection string for integration testing with Postgres
func GetConnectionString(t testing.TB) string {
	database := CreateADatabaseForT(t)
	t.Cleanup(func() {
		DropDatabase(t, database)
	})
	return conn.GetConnectionStringWithDatabaseName(t, database)
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
