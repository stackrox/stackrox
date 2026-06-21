package globaldb

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/globaldb/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/administration/events/stream"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	pgStats "github.com/stackrox/rox/pkg/postgres/stats"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	stats "github.com/stackrox/rox/pkg/telemetry/data"
)

const (
	tableQuery = `
WITH
    -- all partitioned tables
    partition_roots AS (
        SELECT oid FROM pg_class
        WHERE relkind = 'p'
    ),

    -- all partitions of some partitioned table
    partitions AS (
        SELECT c.oid, (pg_partition_tree(c.oid)).relid
        FROM pg_class c
        WHERE c.relkind = 'r'
    ),

    -- rest of the relations
    non_partitioned AS (
        SELECT oid FROM pg_class
        where
            relkind = 'r' AND
            oid NOT IN (SELECT oid FROM partitions)
    )

-- Select size information about partitions summarized by partitioned table
SELECT
    c.relname AS table_name,
    pts.*,
    COALESCE(pts.total_bytes - pts.indexes_bytes - pts.toast_bytes, 0) as table_bytes
    FROM partition_roots p
    LEFT JOIN LATERAL (
        SELECT
            -- Estimated number of live rows
            SUM(c.reltuples) AS rows_estimate,
            -- Total disk space used by the table, including all indexes and TOAST
            SUM(pg_total_relation_size(relid)) AS total_bytes,
            -- Total disk space used by indexes attached to the table
            SUM(pg_indexes_size(relid)) AS indexes_bytes,
            -- Total disk space used by the TOAST table
            SUM(COALESCE(pg_total_relation_size(c.reltoastrelid), 0)) AS toast_bytes
        FROM pg_partition_tree(p.oid)
        LEFT JOIN pg_class c ON relid = c.oid
    ) pts ON TRUE
    LEFT JOIN pg_class c ON p.oid = c.oid
    LEFT JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE
        n.nspname NOT IN ('pg_catalog', 'information_schema')
UNION
-- Select size information about the rest of relations
SELECT
        c.relname AS table_name,
        -- Estimated number of live rows
        c.reltuples AS rows_estimate,
        -- Total disk space used by the table, including all indexes and TOAST
        pg_total_relation_size(np.oid) AS total_bytes,
        -- Total disk space used by indexes attached to the table
        pg_indexes_size(np.oid) AS indexes_bytes,
        -- Total disk space used by the TOAST table
        COALESCE(pg_total_relation_size(c.reltoastrelid), 0) AS toast_bytes,
        COALESCE(pg_total_relation_size(np.oid) - pg_indexes_size(np.oid) - pg_total_relation_size(c.reltoastrelid), 0) as table_bytes
    FROM non_partitioned np
    LEFT JOIN pg_class c ON np.oid = c.oid;
`

	versionQuery = `SHOW server_version;`

	totalConnectionQuery = `SELECT state, COUNT(datid) FROM pg_stat_activity WHERE state IS NOT NULL GROUP BY state;`

	maxConnectionQuery = `SELECT current_setting('max_connections')::int;`

	invalidIndexQuery = `SELECT s.relname, s.indexrelname
		FROM pg_stat_user_indexes s
		JOIN pg_index ix ON ix.indexrelid = s.indexrelid
		WHERE ix.indisvalid = false`

	pgStatStatementsMax = 1000
)

var (
	log = logging.LoggerForModule()

	postgresDB postgres.DB
	pgSync     sync.Once

	// PostgresQueryTimeout - Postgres query timeout value
	PostgresQueryTimeout = 10 * time.Second

	loggedCapacityCalculationError = false

	// writtenStates tracks the states written for connections in order to reset them to zero
	// when they no longer exist
	writtenStates = set.NewStringSet()
)

// GetPostgres returns a global database instance. It should be called after InitializePostgres
func GetPostgres() postgres.DB {
	return postgresDB
}

// SetPostgresTest sets a global database instance. It should be used in tests only.
func SetPostgresTest(t *testing.T, db postgres.DB) postgres.DB {
	t.Log("Initializing Postgres... ")
	postgresDB = db
	return postgresDB
}

// InitializePostgres creates and returns a global database instance.
// The pool is created with lazy connections (MinConns=0) so this returns
// instantly without waiting for the database to be available. The actual
// TCP+TLS connection is established on first query.
func InitializePostgres(ctx context.Context) postgres.DB {
	pgSync.Do(func() {
		_, dbConfig, err := pgconfig.GetPostgresConfig()
		if err != nil {
			log.Fatalf("Could not parse postgres config: %v", err)
		}

		if !pgconfig.IsExternalDatabase() {
			activeDB := pgconfig.GetActiveDB()
			dbConfig.ConnConfig.Database = activeDB
		}

		// Create pool with lazy connections — no network I/O here.
		// Connections are established on first query, which allows the
		// gRPC server to bind its socket before the DB is reachable.
		dbConfig.MinConns = 0
		postgresDB, err = postgres.New(ctx, dbConfig)
		if err != nil {
			log.Fatalf("Could not create postgres pool: %v", err)
		}

		go func() {
			if _, err := postgresDB.Exec(ctx, "create extension if not exists pg_stat_statements"); err != nil {
				log.Warnf("Could not create pg_stat_statements extension: %v", err)
			}
			startMonitoringPostgres(ctx, postgresDB, dbConfig)
		}()
	})
	return postgresDB
}

// GetPostgresVersion -- return version of the database
func GetPostgresVersion(ctx context.Context, db postgres.DB) string {
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
func CollectPostgresStats(ctx context.Context, db postgres.DB) *stats.DatabaseStats {
	ctx, cancel := context.WithTimeout(ctx, PostgresQueryTimeout)
	defer cancel()

	dbStats := &stats.DatabaseStats{}

	if err := db.Ping(ctx); err != nil {
		metrics.PostgresConnected.Set(float64(0))
		dbStats.DatabaseAvailable = false
		log.Errorf("not connected to Postgres: %v", err)
		return dbStats
	}

	metrics.PostgresConnected.Set(float64(1))
	dbStats.DatabaseAvailable = true

	rows, err := db.Query(ctx, tableQuery)
	if err != nil {
		log.Errorf("error fetching object counts: %v", err)
		return dbStats
	}
	defer rows.Close()

	statsSlice := make([]*stats.TableStats, 0)

	for rows.Next() {
		var (
			tableName   string
			rowEstimate int
			totalSize   int
			indexSize   int
			toastSize   int
			tableSize   int
		)
		if err := rows.Scan(&tableName, &rowEstimate, &totalSize, &indexSize, &toastSize, &tableSize); err != nil {
			log.Errorf("error scanning row for table %s: %v", tableName, err)
			return nil
		}

		tableLabel := prometheus.Labels{"table": tableName}
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

	if err := rows.Err(); err != nil {
		log.Errorf("error getting complete table statistic information: %v", err)
	}

	dbStats.Tables = statsSlice
	return dbStats
}

// CollectPostgresDatabaseSizes -- collect database sizing stats for Postgres
func CollectPostgresDatabaseSizes(postgresConfig *postgres.Config) []*stats.DatabaseDetailsStats {
	detailsSlice := make([]*stats.DatabaseDetailsStats, 0)
	var databases []string
	var err error

	if !env.ManagedCentral.BooleanSetting() && !pgconfig.IsExternalDatabase() {
		databases, err = pgadmin.GetAllDatabases(postgresConfig)
		if err != nil {
			log.Errorf("unable to get the databases: %v", err)
			return detailsSlice
		}
	} else {
		databases = append(databases, postgresConfig.ConnConfig.Database)
	}

	for _, database := range databases {
		dbSize, err := pgadmin.GetDatabaseSize(postgresConfig, database)
		if err != nil {
			log.Errorf("error fetching clone size: %v", err)
			return detailsSlice
		}

		dbDetails := &stats.DatabaseDetailsStats{
			DatabaseName: database,
			DatabaseSize: dbSize,
		}
		detailsSlice = append(detailsSlice, dbDetails)
	}

	return detailsSlice
}

// CollectPostgresDatabaseStats -- collect database level stats for Postgres
func CollectPostgresDatabaseStats(postgresConfig *postgres.Config) {
	dbStats := CollectPostgresDatabaseSizes(postgresConfig)

	for _, dbStat := range dbStats {
		databaseLabel := prometheus.Labels{"database": dbStat.DatabaseName}
		metrics.PostgresDBSize.With(databaseLabel).Set(float64(dbStat.DatabaseSize))
	}

	if !env.ManagedCentral.BooleanSetting() && !pgconfig.IsExternalDatabase() {
		totalSize, err := pgadmin.GetTotalPostgresSize(postgresConfig)
		if err != nil {
			log.Errorf("error fetching total database size: %v", err)
			return
		}
		metrics.PostgresTotalSize.Set(float64(totalSize))

		// Check Postgres remaining capacity
		availableDBBytes, err := pgadmin.GetRemainingCapacity(postgresConfig)
		if err != nil {
			if !loggedCapacityCalculationError {
				log.Errorf("error fetching remaining database storage: %v", err)
				loggedCapacityCalculationError = true
			}
			return
		}

		metrics.PostgresRemainingCapacity.Set(float64(availableDBBytes))
	}
}

// CollectPostgresTupleStats -- collect tuple stats for Postgres
func CollectPostgresTupleStats(ctx context.Context, db postgres.DB) {
	tupleStats := pgStats.GetPGTupleStats(ctx, db, pgStatStatementsMax)
	if tupleStats == nil {
		return
	}

	for _, tuple := range tupleStats.Tuples {
		tableLabel := prometheus.Labels{"table": tuple.Table}
		metrics.PostgresTableLiveTuples.With(tableLabel).Set(float64(tuple.NumLiveTuples))
		metrics.PostgresTableDeadTuples.With(tableLabel).Set(float64(tuple.NumDeadTuples))
	}
}

// CollectPostgresConnectionStats -- collect connection stats for Postgres
func CollectPostgresConnectionStats(ctx context.Context, db postgres.DB) {
	// Get the total connections by database
	getTotalConnections(ctx, db)

	// Get the max connections for Postgres
	getMaxConnections(ctx, db)
}

// getTotalConnections -- gets the total connections by database
func getTotalConnections(ctx context.Context, db postgres.DB) {
	ctx, cancel := context.WithTimeout(ctx, PostgresQueryTimeout)
	defer cancel()

	rows, err := db.Query(ctx, totalConnectionQuery)
	if err != nil {
		log.Errorf("error fetching total connection information: %v", err)
		return
	}

	defer rows.Close()

	currentStates := set.NewStringSet()
	for rows.Next() {
		var (
			state           string
			connectionCount int
		)
		if err := rows.Scan(&state, &connectionCount); err != nil {
			log.Errorf("error scanning row for connection data: %v", err)
			return
		}

		currentStates.Add(state)
		stateLabel := prometheus.Labels{"state": state}
		metrics.PostgresTotalConnections.With(stateLabel).Set(float64(connectionCount))
	}
	// Set metric for states that no longer exist to 0
	for state := range writtenStates.Difference(currentStates) {
		stateLabel := prometheus.Labels{"state": state}
		metrics.PostgresTotalConnections.With(stateLabel).Set(0)
	}
	writtenStates = currentStates
}

// getMaxConnections -- gets maximum number of connections to Postgres server
func getMaxConnections(ctx context.Context, db postgres.DB) {
	ctx, cancel := context.WithTimeout(ctx, PostgresQueryTimeout)
	defer cancel()

	row := db.QueryRow(ctx, maxConnectionQuery)
	var connectionCount int
	if err := row.Scan(&connectionCount); err != nil {
		log.Errorf("error fetching max connection information: %v", err)
		return
	}

	metrics.PostgresMaximumConnections.Set(float64(connectionCount))
}

func startMonitoringPostgres(ctx context.Context, db postgres.DB, postgresConfig *postgres.Config) {
	go func() {
		t := time.NewTicker(1 * time.Hour)
		defer t.Stop()
		CollectPostgresIndexStats(ctx, db)
		for range t.C {
			CollectPostgresIndexStats(ctx, db)
		}
	}()

	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()
	for range t.C {
		_ = CollectPostgresStats(ctx, db)
		CollectPostgresDatabaseStats(postgresConfig)
		CollectPostgresConnectionStats(ctx, db)
		CollectPostgresTupleStats(ctx, db)
	}
}

// CollectPostgresIndexStats checks for invalid indexes, exports a Prometheus metric,
// and produces an administration event if any are found.
func CollectPostgresIndexStats(ctx context.Context, db postgres.DB) {
	ctx, cancel := context.WithTimeout(ctx, PostgresQueryTimeout)
	defer cancel()

	rows, err := db.Query(ctx, invalidIndexQuery)
	if err != nil {
		log.Errorf("error checking for invalid indexes: %v", err)
		metrics.PostgresInvalidIndexes.Reset()
		return
	}
	defer rows.Close()

	type invalidIndex struct {
		tableName string
		indexName string
	}
	var invalidIndexes []invalidIndex

	for rows.Next() {
		var idx invalidIndex
		if err := rows.Scan(&idx.tableName, &idx.indexName); err != nil {
			log.Errorf("error scanning row for invalid index data: %v", err)
			return
		}
		invalidIndexes = append(invalidIndexes, idx)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("error reading invalid index rows: %v", err)
	}

	metrics.PostgresInvalidIndexes.Reset()
	for _, idx := range invalidIndexes {
		metrics.PostgresInvalidIndexes.With(prometheus.Labels{
			"index_name": idx.indexName,
			"table_name": idx.tableName,
		}).Set(1)
	}

	if len(invalidIndexes) > 0 {
		var details strings.Builder
		for i, idx := range invalidIndexes {
			if i > 0 {
				details.WriteString(", ")
			}
			fmt.Fprintf(&details, "%s (table: %s)", idx.indexName, idx.tableName)
		}

		stream.Singleton().Produce(&events.AdministrationEvent{
			Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
			Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING,
			Domain:       events.DefaultDomain,
			Message:      fmt.Sprintf("Detected %d invalid PostgreSQL index(es): %s", len(invalidIndexes), details.String()),
			ResourceType: "Database",
			ResourceName: "central-db",
			Hint:         "Invalid indexes can degrade query performance and may indicate a failed index creation or migration. To fix, run: REINDEX INDEX CONCURRENTLY <index_name>; for each invalid index. If the problem persists, contact Red Hat support.",
		})
	}
}
