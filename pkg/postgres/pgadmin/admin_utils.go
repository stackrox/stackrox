package pgadmin

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

const (
	// AdminDB - name of admin database
	AdminDB = "postgres"

	// EmptyDB - name of an empty database (automatically created by postgres)
	EmptyDB = "template0"

	// PostgresQueryTimeout - timeout time for query
	PostgresQueryTimeout = 5 * time.Second

	getCloneStmt = "SELECT datname FROM pg_catalog.pg_database WHERE datname ~ '^%s_.*'"

	// terminateConnectionStmt - terminates connections to the specified database
	terminateConnectionStmt = "SELECT pg_terminate_backend(pg_stat_activity.pid) " +
		"FROM pg_stat_activity " +
		"WHERE datname = $1 " +
		"AND pid <> pg_backend_pid();"

	// totalSizeQuery - get the size taken up by all databases within the Postgres installation.
	// NOTE:  This only works from an account that has total access.  Something to be mindful of
	// in the future if we head into a multi-tenant database situation.
	totalSizeStmt = "SELECT sum(pg_catalog.pg_database_size(datname)) FROM pg_catalog.pg_database"

	// databaseSizeStmt - gets the size of a specific database within Postgres
	databaseSizeStmt = "SELECT pg_catalog.pg_database_size($1)"

	analyzeTimeout = 5 * time.Minute
)

// DropDB - drops a database.
func DropDB(sourceMap map[string]string, adminConfig *postgres.Config, databaseName string) error {
	if pgconfig.IsExternalDatabase() {
		utils.Should(errors.New("unexpected call, should not try to drop a database from an external Postgres."))
	}

	// Set the options for pg_dump from the connection config
	options := []string{
		"-f",
		"--if-exists",
		databaseName,
	}

	// Get the common DB connection info
	options = append(options, GetConnectionOptions(adminConfig)...)

	cmd := exec.Command("dropdb", options...)

	SetPostgresCmdEnv(cmd, sourceMap, adminConfig)
	err := ExecutePostgresCmd(cmd)
	if err != nil {
		log.Errorf("Unable to drop database %s", databaseName)
		return err
	}

	return nil
}

// CreateDB - creates a database from template with the given database name
func CreateDB(sourceMap map[string]string, adminConfig *postgres.Config, dbTemplate, dbName string) error {
	if pgconfig.IsExternalDatabase() {
		utils.Should(errors.New("unexpected call, should not try to create a database from an external Postgres."))
	}

	log.Infof("CreateDB %q", dbName)

	// Set the options for pg_dump from the connection config
	options := []string{
		"-T",
		dbTemplate,
		dbName,
	}

	// Get the common DB connection info
	options = append(options, GetConnectionOptions(adminConfig)...)

	cmd := exec.Command("createdb", options...)

	SetPostgresCmdEnv(cmd, sourceMap, adminConfig)

	// Terminate connections to the source database.  You cannot copy from a database if
	// there are open connections to it.
	err := TerminateConnection(adminConfig, dbTemplate)
	if err != nil {
		return err
	}

	log.Infof("%q has been created", dbName)
	return ExecutePostgresCmd(cmd)
}

// CheckIfDBExists - checks to see if a restore database exists
func CheckIfDBExists(postgresConfig *postgres.Config, dbName string) (bool, error) {
	log.Debugf("CheckIfDBExists - %q", dbName)

	// Connect to different database for admin functions
	connectPool, err := GetAdminPool(postgresConfig)
	if err != nil {
		return false, err
	}
	// Close the admin connection pool
	defer connectPool.Close()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), PostgresQueryTimeout)
	defer cancel()

	existsStmt := "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_database WHERE datname = $1)"

	row := connectPool.QueryRow(ctx, existsStmt, dbName)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, err
	}

	log.Debugf("%q database exists => %t", dbName, exists)
	return exists, nil
}

// GetDatabaseClones - returns list of database clones based off base database
func GetDatabaseClones(postgresConfig *postgres.Config) ([]string, error) {
	log.Debug("GetDatabaseClones")

	// Connect to different database for admin functions
	connectPool, err := GetAdminPool(postgresConfig)
	if err != nil {
		return nil, err
	}
	// Close the admin connection pool
	defer connectPool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), PostgresQueryTimeout)
	defer cancel()

	cloneStmt := fmt.Sprintf(getCloneStmt, config.GetConfig().CentralDB.DatabaseName)

	rows, err := connectPool.Query(ctx, cloneStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clones []string
	for rows.Next() {
		var cloneName string
		if err := rows.Scan(&cloneName); err != nil {
			return nil, err
		}

		clones = append(clones, cloneName)
	}

	log.Debugf("database clones => %s", clones)

	return clones, nil
}

// AnalyzeDatabase - runs ANALYZE on the database named dbName
func AnalyzeDatabase(config *postgres.Config, dbName string) error {
	log.Debugf("Analyze - %q", dbName)

	// Connect to different database for admin functions
	connectPool, err := GetClonePool(config, dbName)
	if err != nil {
		return err
	}
	// Close the admin connection pool
	defer connectPool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), analyzeTimeout)
	defer cancel()
	_, err = connectPool.Exec(ctx, "ANALYZE")

	log.Debug("Analyze done")
	return err
}

// TerminateConnection - terminates connections to the specified database
func TerminateConnection(config *postgres.Config, dbName string) error {
	if pgconfig.IsExternalDatabase() {
		utils.Should(errors.New("unexpected call, should not try to terminate connections from an external Postgres."))
	}

	log.Debugf("TerminateConnection - %q", dbName)

	// Connect to different database for admin functions
	connectPool, err := GetAdminPool(config)
	if err != nil {
		return err
	}
	// Close the admin connection pool
	defer connectPool.Close()

	_, err = connectPool.Exec(context.Background(), terminateConnectionStmt, dbName)

	log.Debug("TerminateConnection done")
	return err
}

// GetAdminPool - returns a pool to connect to the admin database.
// This is useful for renaming databases such as a restore to active.
// THIS POOL SHOULD BE CLOSED ONCE ITS PURPOSE HAS BEEN FULFILLED.
func GetAdminPool(postgresConfig *postgres.Config) (postgres.DB, error) {
	// Clone config to connect to template DB
	tempConfig := postgresConfig.Copy()

	if !pgconfig.IsExternalDatabase() {
		// Need to connect on a static DB so we can rename the used DBs.
		tempConfig.ConnConfig.Database = AdminDB
	}

	postgresDB, err := GetPool(tempConfig)
	if err != nil {
		return nil, err
	}

	log.Debugf("Got connection pool for database %q", AdminDB)
	return postgresDB, nil
}

// GetClonePool - returns a connection pool for the specified database clone.
// THIS POOL SHOULD BE CLOSED ONCE ITS PURPOSE HAS BEEN FULFILLED.
func GetClonePool(postgresConfig *postgres.Config, clone string) (postgres.DB, error) {
	log.Debugf("GetClonePool -- %q", clone)

	// Clone config to connect to template DB
	tempConfig := postgresConfig.Copy()

	if !pgconfig.IsExternalDatabase() {
		// Need to connect on a static DB so we can rename the used DBs.
		tempConfig.ConnConfig.Database = clone
	}

	postgresDB, err := GetPool(tempConfig)
	if err != nil {
		return nil, err
	}

	log.Debugf("Got connection pool for database %q", clone)

	return postgresDB, nil
}

// GetPool - returns a connection pool for the given config
// THIS POOL SHOULD BE CLOSED ONCE ITS PURPOSE HAS BEEN FULFILLED.
func GetPool(postgresConfig *postgres.Config) (postgres.DB, error) {
	var err error
	var postgresDB postgres.DB

	err = pgutils.Retry(func() error {
		postgresDB, err = postgres.New(context.Background(), postgresConfig)
		return err
	})
	if err != nil {
		return nil, err
	}

	return postgresDB, nil
}

// getAvailablePostgresCapacity - retrieves the capacity for Postgres
func getAvailablePostgresCapacity(postgresConfig *postgres.Config) (int64, error) {
	// When using managed services, Postgres space is not a concern at this time.
	if env.ManagedCentral.BooleanSetting() || pgconfig.IsExternalDatabase() {
		utils.Should(errors.New("unexpected call, cannot yet determine managed capacity.  Calculation is an estimate based on suggested size."))

		// Cannot get managed services capacity via Postgres.  Assume size for now.
		return pgconfig.GetPostgresCapacity(), nil
	}

	// Connect to database for admin functions
	connectPool, err := GetAdminPool(postgresConfig)
	if err != nil {
		return 0, err
	}
	// Close the admin connection pool
	defer connectPool.Close()

	// Wrap in a transaction
	ctx, cancel := context.WithTimeout(context.Background(), PostgresQueryTimeout)
	defer cancel()
	conn, err := connectPool.Acquire(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Release()

	// Start a transaction
	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, err
	}

	// COPY needs some place to write the data.  This table will be deleted when the transaction ends.
	_, err = tx.Exec(ctx, "CREATE TEMP TABLE IF NOT EXISTS tmp_sys_df (content text) ON COMMIT DROP;")
	if err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return 0, err
		}
		return 0, errors.Wrap(err, "Unable to create tmp table")
	}

	// COPY can execute a system level program and stream the results to a table.
	_, err = tx.Exec(ctx, "COPY tmp_sys_df FROM PROGRAM 'df -kP $PGDATA | tail'")
	if err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return 0, err
		}
		return 0, errors.Wrap(err, "Unable to copy to tmp table")
	}

	var rawCapacityInfo []string
	rows, err := tx.Query(ctx, "SELECT content FROM tmp_sys_df;")
	if err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return 0, err
		}
		return 0, errors.Wrap(err, "Unable to read tmp table")
	}
	defer rows.Close()

	for rows.Next() {
		var info string
		if err := rows.Scan(&info); err != nil {
			if err := tx.Rollback(ctx); err != nil {
				return 0, err
			}
			return 0, err
		}

		rawCapacityInfo = append(rawCapacityInfo, info)
	}

	// We should only get the header row and the row for the size of $PGDATA.  If we
	// get more than that, then $PGDATA is not defined
	if len(rawCapacityInfo) < 2 {
		if err := tx.Rollback(ctx); err != nil {
			return 0, err
		}
		return 0, errors.New("unable to determine postgres volume capacity")
	}

	// Get the volume size data
	capacityFields := strings.Fields(rawCapacityInfo[1])

	// Get the available blocks
	capacityBlocks, err := strconv.ParseInt(capacityFields[3], 10, 64)
	if err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return 0, err
		}
		return 0, errors.New("unable to determine capacity")
	}

	// Calculate the capacity based on blocks and block size.
	availableCapacity := capacityBlocks * 1024

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return availableCapacity, rows.Err()
}

// GetRemainingCapacity - retrieves the amount of space left in Postgres
func GetRemainingCapacity(postgresConfig *postgres.Config) (int64, error) {
	// When using managed services, Postgres space is not a concern at this time.
	if env.ManagedCentral.BooleanSetting() || pgconfig.IsExternalDatabase() {
		utils.Should(errors.New("unexpected call, cannot yet determine managed capacity.  Calculation is an estimate based on suggested size."))

		// Cannot get managed services capacity via Postgres.  Assume size for now.
		return pgconfig.GetPostgresCapacity(), nil
	}

	sizeUsed, err := GetTotalPostgresSize(postgresConfig)
	if err != nil {
		return 0, err
	}

	capacity, err := getAvailablePostgresCapacity(postgresConfig)
	if err != nil {
		// If we cannot calculate the capacity, assume it based on the recommended starting
		// capacity
		return pgconfig.GetPostgresCapacity() - sizeUsed, err
	}

	return capacity, nil
}

// GetDatabaseSize - retrieves the size of the database specified by dbName
func GetDatabaseSize(postgresConfig *postgres.Config, dbName string) (int64, error) {
	log.Debugf("GetDatabaseSize -- %q", dbName)

	// Connect to different database for admin functions
	connectPool, err := GetAdminPool(postgresConfig)
	if err != nil {
		return 0, err
	}
	// Close the admin connection pool
	defer connectPool.Close()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), PostgresQueryTimeout)
	defer cancel()

	row := connectPool.QueryRow(ctx, databaseSizeStmt, dbName)
	var dbSize int64
	if err := row.Scan(&dbSize); err != nil {
		return 0, err
	}

	log.Debugf("%q size = %d", dbName, dbSize)
	return dbSize, nil
}

// GetTotalPostgresSize - retrieves the total size of all Postgres databases
func GetTotalPostgresSize(postgresConfig *postgres.Config) (int64, error) {
	// Connect to database for admin functions
	connectPool, err := GetAdminPool(postgresConfig)
	if err != nil {
		return 0, err
	}
	// Close the admin connection pool
	defer connectPool.Close()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), PostgresQueryTimeout)
	defer cancel()

	row := connectPool.QueryRow(ctx, totalSizeStmt)
	var sizeUsed int64
	err = row.Scan(&sizeUsed)
	if err != nil {
		return 0, err
	}

	return sizeUsed, nil
}

// GetAllDatabases - returns list of databases in Postgres
func GetAllDatabases(postgresConfig *postgres.Config) ([]string, error) {
	// Connect to different database for admin functions
	connectPool, err := GetAdminPool(postgresConfig)
	if err != nil {
		return nil, err
	}
	// Close the admin connection pool
	defer connectPool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), PostgresQueryTimeout)
	defer cancel()

	rows, err := connectPool.Query(ctx, "SELECT datname FROM pg_catalog.pg_database")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clones []string
	for rows.Next() {
		var cloneName string
		if err := rows.Scan(&cloneName); err != nil {
			return nil, err
		}

		clones = append(clones, cloneName)
	}

	return clones, nil
}
