package globaldb

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	registeredTables = make(map[string]registeredTable)

	postgresOpenRetries        = 10
	postgresTimeBetweenRetries = 10 * time.Second
	postgresDB                 *pgxpool.Pool
	pgSync                     sync.Once
)

type registeredTable struct {
	table, objType string
}

// RegisterTable maps a table to an object type for the purposes of metrics gathering
func RegisterTable(table string, objType string) {
	tableToRegister := registeredTable{
		table:   table,
		objType: objType,
	}

	if registered, ok := registeredTables[table]; ok {
		if registered != tableToRegister {
			log.Fatalf("table %q is already mapped to %q", table, registered.objType)
		}
		return
	}

	registeredTables[table] = tableToRegister
}

// GetPostgres returns a global database instance
func GetPostgres() *pgxpool.Pool {
	pgSync.Do(func() {
		source := "host=central-db.stackrox port=5432 database=postgres user=postgres password=SpTasCsKwRorXdFor$Now8 sslmode=disable statement_timeout=600000 pool_min_conns=1 pool_max_conns=90"
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
	})
	return postgresDB
}
