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
	database := env.GetString("POSTGRES_DB", "postgres")
	host := env.GetString("POSTGRES_HOST", "localhost")
	return fmt.Sprintf("host=%s port=5432 database=%s user=%s sslmode=disable statement_timeout=600000 pool_min_conns=90 pool_max_conns=90", host, database, user)
}
