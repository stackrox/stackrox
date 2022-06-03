package pgtest

import (
	"fmt"
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"k8s.io/utils/env"
)

// GetConnectionString returns a connection string for integration testing with Postgres
func GetConnectionString(_ *testing.T) string {
	user := os.Getenv("USER")
	if _, ok := os.LookupEnv("CI"); ok {
		user = "postgres"
	}
	pass := env.GetString("POSTGRES_PASSWORD", "")
	database := env.GetString("POSTGRES_DB", "postgres")
	host := env.GetString("POSTGRES_HOST", "localhost")
	return fmt.Sprintf("host=%s port=5432 database=%s user=%s password=%s sslmode=disable statement_timeout=600000", host, database, user, pass)
}

// OpenGormDB opens a Gorm DB to the Postgres DB
func OpenGormDB(t *testing.T, source string) *gorm.DB {
	gormDB, err := gorm.Open(postgres.Open(source), &gorm.Config{NamingStrategy: pgutils.NamingStrategy})
	require.NoError(t, err, "failed to connect to connect with gorm db")
	return gormDB
}

// CloseGormDB closes connection to a Gorm DB
func CloseGormDB(t *testing.T, db *gorm.DB) {
	if db == nil {
		return
	}
	genericDB, err := db.DB()
	require.NoError(t, err)
	if err == nil {
		_ = genericDB.Close()
	}
}
