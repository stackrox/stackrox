package pgtest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	"github.com/stackrox/rox/pkg/random"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"k8s.io/utils/env"
)

const (
	driverName = "pgx"

	// defaultDatabaseName is needed to create and drop databases. Without it we can't create or drop databases, it is a catch-22
	// because a database is needed for the connection.
	defaultDatabaseName = "postgres"
)

// TestPostgres is a Postgres instance used in tests
type TestPostgres struct {
	postgres.DB
	database string
}

func TestToDBName(t testing.TB) string {
	suffixLength := 5

	// There is a limit on how large the name could be in PostgreSQL, so we
	// need to cut the generated name to fit into it. Since the idea is to have
	// a randomized suffix, truncate the test name rather than suffix. The
	// actual maximum database name is 63, and could be verified via:
	//
	//     SELECT length(repeat('abcde', 100)::NAME);
	//
	// But we reduce it to 60 to keep a small buffer.
	maxDBName := 60

	nameLength := min(len(t.Name()), maxDBName-suffixLength)
	truncatedTestName := t.Name()[:nameLength]

	suffix, err := random.GenerateString(suffixLength,
		random.AlphanumericCharacters)
	require.NoError(t, err)

	database := truncatedTestName + suffix
	database = strings.ReplaceAll(database, "/", "_")
	database = strings.ReplaceAll(database, "-", "_")
	database = strings.ToLower(database)

	return database
}

// CreateADatabaseForT creates a postgres database for test
func CreateADatabaseForT(t testing.TB) string {
	database := TestToDBName(t)
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
	if database != defaultDatabaseName {
		sourceWithPostgresDatabase := conn.GetConnectionStringWithDatabaseName(t, defaultDatabaseName)
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

	// Create all the tables for the database
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, fmt.Sprintf("%s application_name=%s",
		sourceWithDatabase, "migrator"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to PostgreSQL:\n  %v\n", err)
		os.Exit(1)
	}

	defer conn.Close(ctx)

	// Create a new migrator with schema_version as a version table in the database
	migrator, err := migrate.NewMigrator(ctx, conn, "schema_version")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing migrator:\n  %v\n", err)
		os.Exit(1)
	}

	// Find root of the project
	rootDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get working directory: %v\n", err)
	} else {
		for !strings.HasSuffix(rootDir, "stackrox") {
			rootDir = filepath.Dir(rootDir)
		}
	}

	err = migrator.LoadMigrations(os.DirFS(fmt.Sprintf("%s/migrations", rootDir)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading migrations:\n  %v\n", err)
		os.Exit(1)
	}
	if len(migrator.Migrations) == 0 {
		fmt.Fprintln(os.Stderr, "No migrations found")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt)
	go func() {
		<-interruptChan
		cancel()       // Cancel any in progress migrations
		signal.Reset() // Only listen for one interrupt. If another interrupt signal is received allow it to terminate the program.
	}()

	err = migrator.Migrate(ctx)

	if err != nil {
		if mgErr, ok := err.(migrate.MigrationPgError); ok {
			fmt.Fprintln(os.Stderr, "Migration failed: %v, %v, %v, %v",
				err, mgErr.PgError, mgErr.Detail, mgErr.Position)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	} else {
		fmt.Fprintf(os.Stderr, "Migration succeed\n")
	}

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
	return conn.GetConnectionStringWithDatabaseName(t, env.GetString("POSTGRES_DB", defaultDatabaseName))
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
