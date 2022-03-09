package globaldb

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	registeredTables = make(map[string]registeredTable)

	postgresDB *pgxpool.Pool
	pgSync     sync.Once
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
		source := "host=localhost port=5432 database=postgres user=postgres sslmode=disable statement_timeout=600000 pool_min_conns=1 pool_max_conns=90"
		config, err := pgxpool.ParseConfig(source)
		if err != nil {
			panic(err)
		}
		postgresDB, err = pgxpool.ConnectConfig(context.Background(), config)
		if err != nil {
			panic(err)
		}
	})
	return postgresDB
}
