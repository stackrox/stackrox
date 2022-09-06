package pgadmin

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/retry"
)

var (
	log = logging.LoggerForModule()
)

const (
	// AdminDB - name of admin database
	AdminDB = "postgres"

	// EmptyDB - name of an empty database (automatically created by postgres)
	EmptyDB = "template0"

	// postgresOpenRetries - number of retries when trying to open a connection
	postgresOpenRetries = 10

	// postgresTimeBetweenRetries - time to wait between retries
	postgresTimeBetweenRetries = 10 * time.Second

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

	if err := retry.WithRetry(func() error {
		postgresDB, err = pgxpool.ConnectConfig(context.Background(), postgresConfig)
		return err
	}, retry.Tries(postgresOpenRetries), retry.BetweenAttempts(func(attempt int) {
		time.Sleep(postgresTimeBetweenRetries)
	}), retry.OnFailedAttempts(func(err error) {
		log.Errorf("open database: %v", err)
	})); err != nil {
		log.Fatalf("Timed out trying to open database: %v", err)
	}

	return postgresDB
}

// GetRemainingCapacity - retrieves the amount of space left in Postgres
func GetRemainingCapacity(postgresConfig *pgxpool.Config) (int64, error) {
	// Connect to different database for admin functions
	connectPool := GetAdminPool(postgresConfig)
	// Close the admin connection pool
	defer connectPool.Close()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), PostgresQueryTimeout)
	defer cancel()

	row := connectPool.QueryRow(ctx, totalSizeStmt)
	var sizeUsed int64
	if err := row.Scan(&sizeUsed); err != nil {
		return 0, err
	}

	log.Infof("size used = %d.  size left = %d", sizeUsed, pgconfig.GetPostgresCapacity()-sizeUsed)
	return pgconfig.GetPostgresCapacity() - sizeUsed, nil
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

	log.Infof("%q size = %d.", dbName, dbSize)
	return dbSize, nil
}
