package globaldb

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	pgInit sync.Once
	pgDB   *pgxpool.Pool

	registeredTables []registeredTable

	pgGatherFreq = 1 * time.Minute
)

type registeredTable struct {
	table, objType string
}

func RegisterTable(table string, objType string) {
	registeredTables = append(registeredTables, registeredTable{
		table:   table,
		objType: objType,
	})
}

// GetPostgresDB returns the global postgres instance
func GetPostgresDB() *pgxpool.Pool {
	pgInit.Do(func() {
		source := "host=central-db.stackrox port=5432 user=postgres sslmode=disable statement_timeout=60000 pool_min_conns=90 pool_max_conns=90"

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		config, err := pgxpool.ParseConfig(source)
		if err != nil {
			panic(err)
		}
		pool, err := pgxpool.ConnectConfig(ctx, config)
		if err != nil {
			panic(err)
		}

		conn, err := pool.Acquire(context.Background())
		if err != nil {
			panic(err)
		}
		defer conn.Release()
		t := time.Now()
		conn.Conn().Ping(context.Background())
		fmt.Println("Ping Nanos: ", time.Since(t).Nanoseconds())
		pgDB = pool

		go startMonitoringPostgresDB(pool)
	})
	return pgDB
}

func startMonitoringPostgresDB(db *pgxpool.Pool) {
	ticker := time.NewTicker(pgGatherFreq)
	for range ticker.C {
		for _, registeredTable := range registeredTables {
			var count int
			row := db.QueryRow(context.Background(), "select count(*) from "+registeredTable.table)
			if err := row.Scan(&count); err != nil {
				log.Errorf("error scanning count row for table %s: %v", registeredTable.table, err)
				continue
			}
			log.Infof("table %s has %d objects", registeredTable.table, count)
		}
	}
}
