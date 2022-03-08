package pgtest

import (
	"fmt"
	"os"
	"testing"

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
	return fmt.Sprintf("host=%s port=5432 database=%s user=%s password=%s sslmode=disable statement_timeout=600000 pool_min_conns=1 pool_max_conns=90", host, database, user, pass)
}
