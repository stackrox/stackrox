package globaldb

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/globaldb/metrics"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/set"
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

	totalConnectionQuery = `SELECT state, COUNT(datid) FROM pg_stat_activity WHERE state IS NOT NULL GROUP BY state;`

	maxConnectionQuery = `SELECT current_setting('max_connections')::int;`
)

var (
	log = logging.LoggerForModule()

	postgresOpenRetries        = 10
	postgresTimeBetweenRetries = 10 * time.Second
	postgresDB                 postgres.DB
	pgSync                     sync.Once

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

// InitializePostgres creates and returns returns a global database instance.
func InitializePostgres(ctx context.Context) postgres.DB {
	pgSync.Do(func() {
		_, dbConfig, err := pgconfig.GetPostgresConfig()
		if err != nil {
			log.Fatalf("Could not parse postgres config: %v", err)
		}

		// TODO(ROX-18005): remove this when we no longer have to worry about changing databases
		if !pgconfig.IsExternalDatabase() {
			// Get the active database name for the connection
			activeDB := pgconfig.GetActiveDB()

			// Set the connection to be the active database.
			dbConfig.ConnConfig.Database = activeDB
		}

		if err := retry.WithRetry(func() error {
			postgresDB, err = postgres.New(ctx, dbConfig)
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
			log.Warnf("Could not create pg_stat_statements extension.  Statement planning and execution stats may not be tracked: %v", err)
		}
		go startMonitoringPostgres(ctx, postgresDB, dbConfig)

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

	databases, err := pgadmin.GetAllDatabases(postgresConfig)
	if err != nil {
		log.Errorf("unable to get the databases: %v", err)
		return detailsSlice
	}

	for _, database := range databases {
		dbSize, err := pgadmin.GetDatabaseSize(postgresConfig, database)
		if err != nil {
			log.Errorf("error fetching clone size: %v", err)
			return detailsSlice
		}

		dbDetails := &stats.DatabaseDetailsStats{
			DatabaseName: database,
			DatabaseSize: int64(dbSize),
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

	totalSize, err := pgadmin.GetTotalPostgresSize(postgresConfig)
	if err != nil {
		log.Errorf("error fetching total database size: %v", err)
		return
	}
	metrics.PostgresTotalSize.Set(float64(totalSize))

	// Check Postgres remaining capacity
	if !env.ManagedCentral.BooleanSetting() && !pgconfig.IsExternalDatabase() {
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

func processConnectionCountRow(metric *prometheus.GaugeVec, rows *postgres.Rows) {
	for rows.Next() {
		var (
			databaseName    string
			connectionCount int
		)
		if err := rows.Scan(&databaseName, &connectionCount); err != nil {
			log.Errorf("error scanning row for connection data: %v", err)
			return
		}

		databaseLabel := prometheus.Labels{"database": databaseName}
		metric.With(databaseLabel).Set(float64(connectionCount))
	}
}

func startMonitoringPostgres(ctx context.Context, db postgres.DB, postgresConfig *postgres.Config) {
	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()
	for range t.C {
		_ = CollectPostgresStats(ctx, db)
		CollectPostgresDatabaseStats(postgresConfig)
		CollectPostgresConnectionStats(ctx, db)
	}
}
