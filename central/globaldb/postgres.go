package globaldb

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/globaldb/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sync"
	stats "github.com/stackrox/rox/pkg/telemetry/data"
)

const (
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
              , SUM(COALESCE(pg_total_relation_size(reltoastrelid), 0)) OVER (partition BY parent) AS toast_bytes
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

	versionQuery = `SHOW server_version;`
)

var (
	postgresOpenRetries        = 10
	postgresTimeBetweenRetries = 10 * time.Second
	postgresDB                 *pgxpool.Pool
	pgSync                     sync.Once

	// PostgresQueryTimeout - Postgres query timeout value
	PostgresQueryTimeout = 10 * time.Second
)

// GetPostgres returns a global database instance. It should be called after InitializePostgres
func GetPostgres() *pgxpool.Pool {
	return postgresDB
}

// GetPostgresTest returns a global database instance. It should be used in tests only.
func GetPostgresTest(t *testing.T) *pgxpool.Pool {
	t.Log("Initializing Postgres...")
	return InitializePostgres(context.Background())
}

// InitializePostgres creates and returns returns a global database instance.
func InitializePostgres(ctx context.Context) *pgxpool.Pool {
	pgSync.Do(func() {
		_, dbConfig, err := pgconfig.GetPostgresConfig()
		if err != nil {
			log.Fatalf("Could not parse postgres config: %v", err)
		}

		// Get the active database name for the connection
		activeDB := pgconfig.GetActiveDB()

		// Set the connection to be the active database.
		dbConfig.ConnConfig.Database = activeDB

		if err := retry.WithRetry(func() error {
			postgresDB, err = pgxpool.ConnectConfig(ctx, dbConfig)
			return err
		}, retry.Tries(postgresOpenRetries), retry.BetweenAttempts(func(attempt int) {
			time.Sleep(postgresTimeBetweenRetries)
		}), retry.OnFailedAttempts(func(err error) {
			log.Errorf("open database: %v", err)
		})); err != nil {
			log.Fatalf("Timed out trying to open database: %v", err)
		}

		_, err = postgresDB.Exec(ctx, "create extension if not exists pg_stat_statements")
		if err != nil {
			log.Errorf("Could not create pg_stat_statements extension: %v", err)
		}
		go startMonitoringPostgres(ctx, postgresDB, dbConfig)

	})
	return postgresDB
}

// GetPostgresVersion -- return version of the database
func GetPostgresVersion(ctx context.Context, db *pgxpool.Pool) string {
	ctx, cancel := context.WithTimeout(ctx, PostgresQueryTimeout)
	defer cancel()

	row := db.QueryRow(ctx, versionQuery)
	var version string
	if err := row.Scan(&version); err != nil {
		log.Errorf("error fetching database version: %v", err)
		return ""
	}
	return version
}

// CollectPostgresStats -- collect table level stats for Postgres
func CollectPostgresStats(ctx context.Context, db *pgxpool.Pool) *stats.DatabaseStats {
	ctx, cancel := context.WithTimeout(ctx, PostgresQueryTimeout)
	defer cancel()

	dbStats := &stats.DatabaseStats{}

	if err := db.Ping(ctx); err != nil {
		metrics.PostgresConnected.Set(float64(0))
		dbStats.DatabaseAvailable = false
		log.Errorf("not connected to Postgres: %v", err)
		return nil
	} else {
		metrics.PostgresConnected.Set(float64(1))
		dbStats.DatabaseAvailable = true
	}

	row, err := db.Query(ctx, tableQuery)
	if err != nil {
		log.Errorf("error fetching object counts: %v", err)
		return nil
	}

	statsSlice := make([]*stats.TableStats, 0)

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
			log.Errorf("error scanning row for table %s: %v", tableName, err)
			return nil
		}

		tableLabel := prometheus.Labels{"Table": tableName}
		metrics.PostgresTableCounts.With(tableLabel).Set(float64(rowEstimate))
		metrics.PostgresTableTotalSize.With(tableLabel).Set(float64(totalSize))
		metrics.PostgresIndexSize.With(tableLabel).Set(float64(indexSize))
		metrics.PostgresToastSize.With(tableLabel).Set(float64(toastSize))
		metrics.PostgresTableDataSize.With(tableLabel).Set(float64(tableSize))

		tableStat := &stats.TableStats{
			Name:      tableName,
			RowCount:  int64(rowEstimate),
			TableSize: int64(tableSize),
			IndexSize: int64(indexSize),
			ToastSize: int64(toastSize),
		}

		statsSlice = append(statsSlice, tableStat)
	}

	dbStats.Tables = statsSlice
	return dbStats
}

// CollectPostgresDatabaseStats -- collect database level stats for Postgres
func CollectPostgresDatabaseStats(ctx context.Context, postgresConfig *pgxpool.Config) {
	ctx, cancel := context.WithTimeout(ctx, PostgresQueryTimeout)
	defer cancel()

	clones := pgadmin.GetDatabaseClones(postgresConfig)

	for _, clone := range clones {
		cloneSize, err := pgadmin.GetDatabaseSize(postgresConfig, clone)
		if err != nil {
			log.Errorf("error fetching clone size: %v", err)
			return
		}

		cloneLabel := prometheus.Labels{"Clone": clone}
		metrics.PostgresDBSize.With(cloneLabel).Set(float64(cloneSize))
	}
}

func startMonitoringPostgres(ctx context.Context, db *pgxpool.Pool, postgresConfig *pgxpool.Config) {
	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()
	for range t.C {
		_ = CollectPostgresStats(ctx, db)
		CollectPostgresDatabaseStats(ctx, postgresConfig)
	}
}
