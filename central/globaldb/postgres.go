package globaldb

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/globaldb/metrics"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	dbPasswordFile = "/run/secrets/stackrox.io/db-password/password"

	tableQuery = `WITH RECURSIVE pg_inherit(inhrelid, inhparent) AS
    (select inhrelid, inhparent
    FROM pg_inherits
    UNION
    SELECT child.inhrelid, parent.inhparent
    FROM pg_inherit child, pg_inherits parent
    WHERE child.inhparent = parent.inhrelid),
pg_inherit_short AS (SELECT * FROM pg_inherit WHERE inhparent NOT IN (SELECT inhrelid FROM pg_inherit))
SELECT TABLE_NAME
    , row_estimate
    , total_bytes AS total
    , index_bytes AS INDEX
    , toast_bytes AS toast
    , table_bytes AS TABLE
  FROM (
    SELECT *, total_bytes-index_bytes-COALESCE(toast_bytes,0) AS table_bytes
    FROM (
         SELECT c.oid
              , relname AS TABLE_NAME
              , SUM(c.reltuples) OVER (partition BY parent) AS row_estimate
              , SUM(pg_total_relation_size(c.oid)) OVER (partition BY parent) AS total_bytes
              , SUM(pg_indexes_size(c.oid)) OVER (partition BY parent) AS index_bytes
              , SUM(pg_total_relation_size(reltoastrelid)) OVER (partition BY parent) AS toast_bytes
              , parent
          FROM (
                SELECT pg_class.oid
                    , reltuples
                    , relname
                    , relnamespace
                    , pg_class.reltoastrelid
                    , COALESCE(inhparent, pg_class.oid) parent
                FROM pg_class
                    LEFT JOIN pg_inherit_short ON inhrelid = oid
                WHERE relkind IN ('r', 'p')
             ) c
             LEFT JOIN pg_namespace n ON n.oid = c.relnamespace WHERE nspname = 'public'
  ) a
  WHERE oid = parent
) a;`
)

var (
	postgresOpenRetries        = 10
	postgresTimeBetweenRetries = 10 * time.Second
	postgresDB                 *pgxpool.Pool
	pgSync                     sync.Once

	// PostgresQueryTimeout - Postgres query timeout value
	PostgresQueryTimeout = 10 * time.Second
)

// GetPostgres returns a global database instance
func GetPostgres() *pgxpool.Pool {
	pgSync.Do(func() {
		_, config, err := GetPostgresConfig()
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
		go startMonitoringPostgres(postgresDB)

	})
	return postgresDB
}

// GetPostgresConfig - gets the configuration used to connect to Postgres
func GetPostgresConfig() (map[string]string, *pgxpool.Config, error) {
	centralConfig := config.GetConfig()
	password, err := os.ReadFile(dbPasswordFile)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "pgsql: could not load password file %q", dbPasswordFile)
	}
	source := fmt.Sprintf("%s password=%s", centralConfig.CentralDB.Source, password)

	config, err := pgxpool.ParseConfig(source)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Could not parse postgres config")
	}

	sourceMap, err := ParseSource(source)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Could not parse postgres source")
	}

	return sourceMap, config, nil
}

// ParseSource - parses source string into a map for easier use
func ParseSource(source string) (map[string]string, error) {
	if source == "" {
		return nil, errors.New("source string is empty")
	}

	sourceSlice := strings.Fields(source)
	sourceMap := make(map[string]string)
	for _, pair := range sourceSlice {
		// Due to the possibility that the password could potentially have an = we
		// need to ensure that we get the entire password
		key, value := stringutils.Split2(pair, "=")

		sourceMap[key] = strings.TrimSpace(value)
	}

	return sourceMap, nil
}

func collectPostgresStats(db *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(context.Background(), PostgresQueryTimeout)
	defer cancel()
	row, err := db.Query(ctx, tableQuery)
	if err != nil {
		log.Errorf("error fetching object counts: %v", err)
		return
	}

	defer row.Close()
	for row.Next() {
		var (
			tableName   string
			rowEstimate int
			totalSize   int
			indexSize   int
			toastSize   int
			tableSize   int
		)
		if err := row.Scan(&tableName, &rowEstimate, &totalSize, &indexSize, &toastSize, &tableSize); err != nil {
			log.Errorf("error scanning row: %v", err)
			return
		}

		tableLabel := prometheus.Labels{"Table": tableName}
		metrics.PostgresTableCounts.With(tableLabel).Set(float64(rowEstimate))
		metrics.PostgresTableTotalSize.With(tableLabel).Set(float64(totalSize))
		metrics.PostgresIndexSize.With(tableLabel).Set(float64(indexSize))
		metrics.PostgresToastSize.With(tableLabel).Set(float64(toastSize))
		metrics.PostgresTableDataSize.With(tableLabel).Set(float64(tableSize))
	}
}

func startMonitoringPostgres(db *pgxpool.Pool) {
	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()
	for range t.C {
		collectPostgresStats(db)
	}
}
