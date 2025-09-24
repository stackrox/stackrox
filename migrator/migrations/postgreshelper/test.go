package postgreshelper

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestPostgres is a postgres database for migration testing
type TestPostgres struct {
	postgres.DB
	gormDB   *gorm.DB
	database string
}

// GetGormDB returns the gorm.DB instance
func (tp *TestPostgres) GetGormDB() *gorm.DB {
	return tp.gormDB
}

// ForT creates and returns a Postgres for the test
func ForT(t testing.TB, disableConstraint bool) *TestPostgres {
	// Bootstrap a test database
	database := pgtest.CreateADatabaseForT(t)

	sourceWithDatabase := conn.GetConnectionStringWithDatabaseName(t, database)
	ctx := context.Background()

	// initialize pool to be used
	pool, err := postgres.Connect(ctx, sourceWithDatabase)
	require.NoError(t, err)

	return &TestPostgres{
		DB:       pool,
		gormDB:   conn.OpenGormDB(t, sourceWithDatabase, disableConstraint),
		database: database,
	}
}

// ForTExistingDB - creates and returns a Postgres for the test.  This is used primarily in testing scale where
// we may have a loaded database we want to connect to.
func ForTExistingDB(t testing.TB, disableConstraint bool, dbName string) *TestPostgres {
	//database := strings.ToLower(dbName)

	sourceWithDatabase := conn.GetConnectionStringWithDatabaseName(t, dbName)
	ctx := context.Background()

	// initialize pool to be used
	pool, err := postgres.Connect(ctx, sourceWithDatabase)
	require.NoError(t, err)

	return &TestPostgres{
		DB:       pool,
		gormDB:   conn.OpenGormDB(t, sourceWithDatabase, disableConstraint),
		database: dbName,
	}
}

// Teardown removes the postgres test database
func (tp *TestPostgres) Teardown(t testing.TB) {
	if tp == nil {
		return
	}
	tp.Close()
	pgtest.CloseGormDB(t, tp.gormDB)
	pgtest.DropDatabase(t, tp.database)
}
