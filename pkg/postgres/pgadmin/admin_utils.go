package pgadmin

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
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
)

// DropDB - drops a database.
func DropDB(sourceMap map[string]string, adminConfig *pgxpool.Config, databaseName string) error {
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
func CreateDB(sourceMap map[string]string, adminConfig *pgxpool.Config, dbTemplate, dbName string) error {
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

// RenameDB - renames a database
func RenameDB(adminPool *pgxpool.Pool, originalDB, newDB string) error {
	log.Debugf("Renaming database %q to %q", originalDB, newDB)
	ctx, cancel := context.WithTimeout(context.Background(), PostgresQueryTimeout)
	defer cancel()

	// Terminate connections to the source database.  You cannot move a database if
	// there are open connections to it.
	err := TerminateConnection(adminPool.Config(), originalDB)
	if err != nil {
		return err
	}

	sqlStmt := fmt.Sprintf("ALTER DATABASE %s RENAME TO %s", originalDB, newDB)

	_, err = adminPool.Exec(ctx, sqlStmt)

	return err
}

// CheckIfDBExists - checks to see if a restore database exists
func CheckIfDBExists(postgresConfig *pgxpool.Config, dbName string) bool {
	log.Debugf("CheckIfDBExists - %q", dbName)

	// Connect to different database for admin functions
	connectPool := GetAdminPool(postgresConfig)
	// Close the admin connection pool
	defer connectPool.Close()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), PostgresQueryTimeout)
	defer cancel()

	existsStmt := "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_database WHERE datname = $1)"

	row := connectPool.QueryRow(ctx, existsStmt, dbName)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false
	}

	log.Debugf("%q database exists => %t", dbName, exists)
	return exists
}

// GetDatabaseClones - returns list of database clones based off base database
func GetDatabaseClones(postgresConfig *pgxpool.Config) []string {
	log.Debug("GetDatabaseClones")

	// Connect to different database for admin functions
	connectPool := GetAdminPool(postgresConfig)
	// Close the admin connection pool
	defer connectPool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), PostgresQueryTimeout)
	defer cancel()

	cloneStmt := fmt.Sprintf(getCloneStmt, config.GetConfig().CentralDB.DatabaseName)

	rows, err := connectPool.Query(ctx, cloneStmt)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var clones []string
	for rows.Next() {
		var cloneName string
		if err := rows.Scan(&cloneName); err != nil {
			return nil
		}

		clones = append(clones, cloneName)
	}

	log.Debugf("database clones => %s", clones)

	return clones
}

// AnalyzeDatabase - runs ANALYZE on the database named dbName
func AnalyzeDatabase(config *pgxpool.Config, dbName string) error {
	log.Debugf("Analyze - %q", dbName)

	// Connect to different database for admin functions
	connectPool := GetClonePool(config, dbName)
	// Close the admin connection pool
	defer connectPool.Close()

	_, err := connectPool.Exec(context.Background(), "ANALYZE")

	log.Debug("Anaylze done")
	return err
}

// TerminateConnection - terminates connections to the specified database
func TerminateConnection(config *pgxpool.Config, dbName string) error {
	log.Debugf("TerminateConnection - %q", dbName)

	// Connect to different database for admin functions
	connectPool := GetAdminPool(config)
	// Close the admin connection pool
	defer connectPool.Close()

	_, err := connectPool.Exec(context.Background(), terminateConnectionStmt, dbName)

	log.Debug("TerminateConnection done")
	return err
}

// GetAdminPool - returns a pool to connect to the admin database.
// This is useful for renaming databases such as a restore to active.
// THIS POOL SHOULD BE CLOSED ONCE ITS PURPOSE HAS BEEN FULFILLED.
func GetAdminPool(postgresConfig *pgxpool.Config) *pgxpool.Pool {
	// Clone config to connect to template DB
	tempConfig := postgresConfig.Copy()

	// Need to connect on a static DB so we can rename the used DBs.
	tempConfig.ConnConfig.Database = AdminDB

	postgresDB := getPool(tempConfig)

	log.Debugf("Got connection pool for database %q", AdminDB)
	return postgresDB
}

// GetClonePool - returns a connection pool for the specified database clone.
// THIS POOL SHOULD BE CLOSED ONCE ITS PURPOSE HAS BEEN FULFILLED.
func GetClonePool(postgresConfig *pgxpool.Config, clone string) *pgxpool.Pool {
	log.Debugf("GetClonePool -- %q", clone)

	// Clone config to connect to template DB
	tempConfig := postgresConfig.Copy()

	// Need to connect on a static DB so we can rename the used DBs.
	tempConfig.ConnConfig.Database = clone

	postgresDB := getPool(tempConfig)

	log.Debugf("Got connection pool for database %q", clone)

	return postgresDB
}

func getPool(postgresConfig *pgxpool.Config) *pgxpool.Pool {
	var err error
	var postgresDB *pgxpool.Pool

	err = pgutils.Retry(func() error {
		postgresDB, err = pgxpool.NewWithConfig(context.Background(), postgresConfig)
		return err
	})
	if err != nil {
		log.Fatalf("Timed out trying to open database: %v", err)
	}

	return postgresDB
}

// getAvailablePostgresCapacity - retrieves the capacity for Postgres
func getAvailablePostgresCapacity(postgresConfig *pgxpool.Config) (int64, error) {
	// When using managed services, Postgres space is not a concern at this time.
	if env.ManagedCentral.BooleanSetting() {
		utils.Should(errors.New("unexpected call, cannot yet determine managed capacity.  Calculation is an estimate based on suggested size."))

		// Cannot get managed services capacity via Postgres.  Assume size for now.
		return pgconfig.GetPostgresCapacity(), nil
	}

	// Connect to database for admin functions
	connectPool := GetAdminPool(postgresConfig)
	// Close the admin connection pool
	defer connectPool.Close()

	// Wrap in a transaction
	ctx, cancel := context.WithTimeout(context.Background(), PostgresQueryTimeout)
	defer cancel()
	conn, err := connectPool.Acquire(ctx)
	if err != nil {
		log.Error(err)
		return 0, err
	}
	defer conn.Release()

	// Start a transaction
	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Error(err)
		return 0, err
	}

	// COPY needs some place to write the data.  This table will be deleted when the transaction ends.
	_, err = tx.Exec(ctx, "CREATE TEMP TABLE IF NOT EXISTS tmp_sys_df (content text) ON COMMIT DROP;")
	if err != nil {
		log.Errorf("Unable to create tmp table: %v", err)
		if err := tx.Rollback(ctx); err != nil {
			return 0, err
		}
		return 0, err
	}

	// COPY can execute a system level program and stream the results to a table.
	_, err = tx.Exec(ctx, "COPY tmp_sys_df FROM PROGRAM 'df -kP $PGDATA | tail'")
	if err != nil {
		log.Errorf("Unable to copy to tmp table: %v", err)
		if err := tx.Rollback(ctx); err != nil {
			return 0, err
		}
		return 0, err
	}

	var rawCapacityInfo []string
	rows, err := tx.Query(ctx, "SELECT content FROM tmp_sys_df;")
	if err != nil {
		log.Errorf("Unable to read tmp table: %v", err)
		if err := tx.Rollback(ctx); err != nil {
			return 0, err
		}
		return 0, err
	}

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
	if len(rawCapacityInfo) != 2 {
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

	return availableCapacity, nil
}

// GetRemainingCapacity - retrieves the amount of space left in Postgres
func GetRemainingCapacity(postgresConfig *pgxpool.Config) (int64, error) {
	// When using managed services, Postgres space is not a concern at this time.
	if env.ManagedCentral.BooleanSetting() {
		utils.Should(errors.New("unexpected call, cannot yet determine managed capacity.  Calculation is an estimate based on suggested size."))

		// Cannot get managed services capacity via Postgres.  Assume size for now.
		return pgconfig.GetPostgresCapacity(), nil
	}

	// Connect to database for admin functions
	connectPool := GetAdminPool(postgresConfig)
	// Close the admin connection pool
	defer connectPool.Close()

	sizeUsed, err := GetTotalPostgresSize(postgresConfig)
	if err != nil {
		return 0, err
	}

	capacity, err := getAvailablePostgresCapacity(postgresConfig)
	if err != nil {
		log.Error(err)
		// If we cannot calculate the capacity, assume it based on the recommended starting
		// capacity
		return pgconfig.GetPostgresCapacity() - sizeUsed, err
	}

	log.Infof("remaining capacity = %d", capacity)
	return capacity, nil
}

// GetDatabaseSize - retrieves the size of the database specified by dbName
func GetDatabaseSize(postgresConfig *pgxpool.Config, dbName string) (int64, error) {
	log.Debugf("GetDatabaseSize -- %q", dbName)

	// Connect to different database for admin functions
	connectPool := GetAdminPool(postgresConfig)
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

	log.Debugf("%q size = %d.", dbName, dbSize)
	return dbSize, nil
}

// GetTotalPostgresSize - retrieves the total size of all Postgres databases
func GetTotalPostgresSize(postgresConfig *pgxpool.Config) (int64, error) {
	// Connect to database for admin functions
	connectPool := GetAdminPool(postgresConfig)
	// Close the admin connection pool
	defer connectPool.Close()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), PostgresQueryTimeout)
	defer cancel()

	row := connectPool.QueryRow(ctx, totalSizeStmt)
	var sizeUsed int64
	err := row.Scan(&sizeUsed)
	if err != nil {
		return 0, err
	}

	return sizeUsed, nil
}

// GetAllDatabases - returns list of databases in Postgres
func GetAllDatabases(postgresConfig *pgxpool.Config) []string {
	// Connect to different database for admin functions
	connectPool := GetAdminPool(postgresConfig)
	// Close the admin connection pool
	defer connectPool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), PostgresQueryTimeout)
	defer cancel()

	rows, err := connectPool.Query(ctx, "SELECT datname FROM pg_catalog.pg_database")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var clones []string
	for rows.Next() {
		var cloneName string
		if err := rows.Scan(&cloneName); err != nil {
			return nil
		}

		clones = append(clones, cloneName)
	}

	return clones
}
