package globaldb

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/retry"
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
	source := "host=central-db.stackrox port=5432 user=postgres sslmode=disable statement_timeout=600000 pool_min_conns=90 pool_max_conns=90"
	// 		source := "host=localhost port=5432 database=postgres user=connorgorman sslmode=disable statement_timeout=600000 pool_min_conns=90 pool_max_conns=90"
	pgInitialize(source)
	return pgDB
}

func pgInitialize(source string) {
	pgInit.Do(func() {
		config, err := pgxpool.ParseConfig(source)
		if err != nil {
			panic(err)
		}
		err = retry.WithRetry(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			pool, err := pgxpool.ConnectConfig(ctx, config)
			if err != nil {
				return retry.MakeRetryable(err)
			}

			conn, err := pool.Acquire(context.Background())
			if err != nil {
				return retry.MakeRetryable(err)
			}
			defer conn.Release()
			t := time.Now()
			if err := conn.Conn().Ping(context.Background()); err != nil {
				return retry.MakeRetryable(err)
			}
			fmt.Println("Ping Nanos: ", time.Since(t).Nanoseconds())
			pgDB = pool
			return nil
		}, retry.Tries(20), retry.BetweenAttempts(func(_ int) {
			time.Sleep(3 * time.Second)
		}))
		if err != nil {
			//panic(err)
		}
		go startMonitoringPostgresDB(pgDB)
	})
}

func startMonitoringPostgresDB(db *pgxpool.Pool) {
	ticker := time.NewTicker(pgGatherFreq)
	for range ticker.C {
		for _, registeredTable := range registeredTables {
			var count int
			row := db.QueryRow(context.Background(), "SELECT reltuples::bigint AS estimate FROM pg_class WHERE relname=$1", registeredTable.table)
			if err := row.Scan(&count); err != nil {
				log.Errorf("error scanning count row for table %s: %v", registeredTable.table, err)
				continue
			}
			//log.Infof("table %s has %d objects", registeredTable.table, count)
		}

		rows, err := db.Query(context.Background(), "select total_exec_time, mean_exec_time, calls, substr(query, 1, 300) from pg_stat_statements where calls > 5 order by total_exec_time desc limit 50 ;")
		if err != nil {
			//log.Errorf("Error getting pg stat statements: %v", err)
			continue
		}
		for rows.Next() {
			var totalExecTime, meanExecTime float64
			var calls int
			var query string
			if err := rows.Scan(&totalExecTime, &meanExecTime, &calls, &query); err != nil {
				log.Errorf("error scanning pg_stat_statement: %v", err)
			}
			//log.Infof("Stat: total=%0.2f mean=%0.2f calls=%d query=%s", totalExecTime, meanExecTime, calls, query)
		}

	}
}
