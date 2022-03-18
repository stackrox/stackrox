package globaldb

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	dbPasswordFile = "/run/secrets/stackrox.io/db-password/password"
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
	})
	return postgresDB
}
