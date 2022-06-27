package pgtest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/random"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"k8s.io/utils/env"

	_ "github.com/lib/pq"
)

type TestPostgres struct {
	*pgxpool.Pool
	database string
}

func createDatabase(t testing.TB, database string) {
	// Bootstrap the test database by connecting to the default postgres database and running create
	sourceWithPostgresDatabase := getConnectionStringWithDatabaseName("postgres")
	db, err := sql.Open("postgres", sourceWithPostgresDatabase)
	require.NoError(t, err)

	_, err = db.Exec("CREATE DATABASE " + database)
	require.NoError(t, err)
	require.NoError(t, db.Close())
}

func dropDatabase(t testing.TB, database string) {
	// Bootstrap the test database by connecting to the default postgres database and running create
	sourceWithPostgresDatabase := getConnectionStringWithDatabaseName("postgres")
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

	sourceWithDatabase := getConnectionStringWithDatabaseName(database)
	ctx := context.Background()

	// Create all the tables for the database
	gormDB := OpenGormDB(t, sourceWithDatabase)
	pkgSchema.ApplyAllSchemas(context.Background(), gormDB)
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
	return getConnectionStringWithDatabaseName(env.GetString("POSTGRES_DB", "postgres"))
}

func getConnectionStringWithDatabaseName(name string) string {
	return fmt.Sprintf("%s database=%s", getConnectionStringWithoutDatabaseName(), name)
}

func getConnectionStringWithoutDatabaseName() string {
	user := os.Getenv("USER")
	if _, ok := os.LookupEnv("CI"); ok {
		user = "postgres"
	}
	pass := env.GetString("POSTGRES_PASSWORD", "")
	host := env.GetString("POSTGRES_HOST", "localhost")
	src := fmt.Sprintf("host=%s port=5432 user=%s sslmode=disable statement_timeout=600000", host, user)
	if pass != "" {
		src += fmt.Sprintf(" password=%s", pass)
	}
	return src
}

// OpenGormDB opens a Gorm DB to the Postgres DB
func OpenGormDB(t testing.TB, source string) *gorm.DB {
	gormDB, err := gorm.Open(postgres.Open(source), &gorm.Config{NamingStrategy: pgutils.NamingStrategy})
	require.NoError(t, err, "failed to connect to connect with gorm db")
	return gormDB
}

// CloseGormDB closes connection to a Gorm DB
func CloseGormDB(t testing.TB, db *gorm.DB) {
	if db == nil {
		return
	}
	genericDB, err := db.DB()
	require.NoError(t, err)
	if err == nil {
		_ = genericDB.Close()
	}
}
