package globaldb

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	pgInit sync.Once
	pgDB   *sql.DB

	registeredTables []registeredTable

	pgGatherFreq = 1 * time.Minute
)

type registeredTable struct {
	table, objType string
}

func RegisterTable(table string, objType string) {
	registeredTables = append(registeredTables, registeredTable{
		table: table,
		objType: objType,
	})
}

// GetPostgresDB returns the global postgres instance
func GetPostgresDB() *sql.DB {
	pgInit.Do(func() {
		source := "host=central-db.stackrox port=5432 user=postgres sslmode=disable statement_timeout=60000"
		db, err := sql.Open("postgres", source)
		if err != nil {
			panic(err)
		}

		if err := db.Ping(); err != nil {
			panic(err)
		}
		pgDB = db
		go startMonitoringPostgresDB(pgDB)
	})
	return pgDB
}

func startMonitoringPostgresDB(db *sql.DB) {
	ticker := time.NewTicker(pgGatherFreq)
	for range ticker.C {
		for _, registeredTable := range registeredTables {
			row := db.QueryRow("select count(*) from "+ registeredTable.table)
			if err := row.Err(); err != nil {
				log.Errorf("error getting size of table %s: %v", registeredTable.table, err)
				continue
			}
			var count int
			if err := row.Scan(&count); err != nil {
				log.Errorf("error scanning count row for table %s: %v", registeredTable.table, err)
				continue
			}
			log.Infof("table %s has %d objects", registeredTable.table, count)
		}
	}
}
