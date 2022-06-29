package conn

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"k8s.io/utils/env"
)

// GetConnectionString returns a connection string for integration testing with Postgres
func GetConnectionString(_ *testing.T) string {
	return GetConnectionStringWithDatabaseName(env.GetString("POSTGRES_DB", "postgres"))
}

// GetConnectionStringWithDatabaseName returns a connection string with the passed database
func GetConnectionStringWithDatabaseName(database string) string {
	user := os.Getenv("USER")
	if _, ok := os.LookupEnv("CI"); ok {
		user = "postgres"
	}
	pass := env.GetString("POSTGRES_PASSWORD", "")
	host := env.GetString("POSTGRES_HOST", "localhost")
	src := fmt.Sprintf("host=%s port=5432 user=%s database=%s sslmode=disable statement_timeout=600000", host, user, database)
	if pass != "" {
		src += fmt.Sprintf(" password=%s", pass)
	}
	return src
}

// OpenGormDB opens a Gorm DB to the Postgres DB
func OpenGormDB(t testing.TB, source string, disableConstraint bool) *gorm.DB {
	gormDB, err := gorm.Open(
		postgres.Open(source),
		&gorm.Config{
			NamingStrategy:                           pgutils.NamingStrategy,
			DisableForeignKeyConstraintWhenMigrating: disableConstraint,
		},
	)
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

func CleanUpDB(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	_, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "CREATE SCHEMA public")
	require.NoError(t, err)
}
