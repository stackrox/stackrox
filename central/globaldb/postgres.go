package globaldb

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	dbPasswordFile = "/run/secrets/stackrox.io/db-password/password"
)

var (
	registeredTables = make(map[string]*walker.Schema)

	postgresOpenRetries        = 10
	postgresTimeBetweenRetries = 10 * time.Second
	postgresDB                 *pgxpool.Pool
	pgSync                     sync.Once
)

// RegisterTable maps a table to an object type for the purposes of metrics gathering
func RegisterTable(schema *walker.Schema) {
	if _, ok := registeredTables[schema.Table]; ok {
		log.Fatalf("table %q is already registered for %s", schema.Table, schema.Type)
		return
	}
	registeredTables[schema.Table] = schema
}

// GetPostgres returns a global database instance
func GetPostgres() *pgxpool.Pool {
	pgSync.Do(func() {
		centralConfig := config.GetConfig()
		password, err := os.ReadFile(dbPasswordFile)
		if err != nil {
			log.Fatalf("pgsql: could not load password file %q: %v", dbPasswordFile, err)
			return
		}
		source := fmt.Sprintf("%s password=%s", centralConfig.CentralDB.Source, password)

		config, err := pgxpool.ParseConfig(source)
		if err != nil {
			log.Fatalf("Could not parse postgres config: %v", err)
		}

		if err := retry.WithRetry(func() error {
			postgresDB, err = pgxpool.ConnectConfig(context.Background(), config)
			return err
		}, retry.Tries(postgresOpenRetries), retry.BetweenAttempts(func(attempt int) {
			time.Sleep(postgresTimeBetweenRetries)
		}), retry.OnFailedAttempts(func(err error) {
			log.Errorf("open database: %v", err)
		})); err != nil {
			log.Fatalf("Timed out trying to open database: %v", err)
		}

		_, err = postgresDB.Exec(context.TODO(), "create extension if not exists pg_stat_statements")
		if err != nil {
			log.Errorf("Could not create pg_stat_statements extension: %v", err)
		}

	})
	return postgresDB
}

// GetSchemaForTable return the schema registered for specified table name.
func GetSchemaForTable(tableName string) *walker.Schema {
	return registeredTables[tableName]
}

// GetAllRegisteredSchemas returns all registered schemas.
func GetAllRegisteredSchemas() map[string]*walker.Schema {
	ret := make(map[string]*walker.Schema)
	for k, v := range registeredTables {
		ret[k] = v
	}
	return ret
}
