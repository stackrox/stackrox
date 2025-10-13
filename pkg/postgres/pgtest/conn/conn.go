package conn

import (
	"fmt"
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"k8s.io/utils/env"
)

// GetConnectionStringWithDatabaseName returns a connection string with the passed database
func GetConnectionStringWithDatabaseName(t testing.TB, database string) string {
	return fmt.Sprintf("%s database=%s", GetConnectionString(t), database)
}

// GetConnectionString returns a connection string for integration testing with Postgres w/o database name.
func GetConnectionString(_ testing.TB) string {
	user := os.Getenv("USER")
	if _, ok := os.LookupEnv("CI"); ok {
		user = "postgres"
	}
	pass := env.GetString("POSTGRES_PASSWORD", "")
	host := env.GetString("POSTGRES_HOST", "localhost")
	port := env.GetString("POSTGRES_PORT", "5432")
	src := fmt.Sprintf("host=%s port=%s user=%s sslmode=disable statement_timeout=600000 client_encoding=UTF8", host, port, user)
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
			Logger:                                   logger.Discard,
			QueryFields:                              true,
		},
	)
	require.NoError(t, err, "failed to connect to connect with gorm db")
	return gormDB
}

// CloseGormDB closes connection to a Gorm DB.
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
