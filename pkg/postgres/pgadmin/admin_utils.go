package pgadmin

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
)

var (
	log = logging.LoggerForModule()

	postgresQueryTimeout = 5 * time.Second
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

	getCloneStmt = "SELECT datname FROM pg_catalog.pg_database WHERE datname ~ '^%s_.*'"
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

	log.Infof("%q has been created", dbName)
	return ExecutePostgresCmd(cmd)
}

// RenameDB - renames a database
func RenameDB(adminPool *pgxpool.Pool, originalDB, newDB string) error {
	log.Debugf("Renaming database %q to %q", originalDB, newDB)
	ctx, cancel := context.WithTimeout(context.Background(), postgresQueryTimeout)
	defer cancel()

	sqlStmt := fmt.Sprintf("ALTER DATABASE %s RENAME TO %s", originalDB, newDB)

	_, err := adminPool.Exec(ctx, sqlStmt)

	return err
}

// CheckIfDBExists - checks to see if a restore database exists
func CheckIfDBExists(pgConfig *pgxpool.Config, dbName string) bool {
	log.Debugf("CheckIfDBExists - %q", dbName)

	// Connect to different database for admin functions
	connectPool := GetAdminPool(pgConfig)
	// Close the admin connection pool
	defer connectPool.Close()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), postgresQueryTimeout)
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
func GetDatabaseClones(pgConfig *pgxpool.Config) []string {
	log.Debug("GetDatabaseClones")

	// Connect to different database for admin functions
	connectPool := GetAdminPool(pgConfig)
	// Close the admin connection pool
	defer connectPool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), postgresQueryTimeout)
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

// GetAdminPool - returns a pool to connect to the admin database.
// This is useful for renaming databases such as a restore to active.
// THIS POOL SHOULD BE CLOSED ONCE ITS PURPOSE HAS BEEN FULFILLED.
func GetAdminPool(pgConfig *pgxpool.Config) *pgxpool.Pool {
	// Clone config to connect to template DB
	tempConfig := pgConfig.Copy()

	// Need to connect on a static DB so we can rename the used DBs.
	tempConfig.ConnConfig.Database = AdminDB

	postgresDB := getPool(tempConfig)

	log.Debugf("Got connection pool for database %q", AdminDB)
	return postgresDB
}

// GetClonePool - returns a connection pool for the specified database clone.
// THIS POOL SHOULD BE CLOSED ONCE ITS PURPOSE HAS BEEN FULFILLED.
func GetClonePool(pgConfig *pgxpool.Config, clone string) *pgxpool.Pool {
	log.Debugf("GetClonePool -- %q", clone)

	// Clone config to connect to template DB
	tempConfig := pgConfig.Copy()

	// Need to connect on a static DB so we can rename the used DBs.
	tempConfig.ConnConfig.Database = clone

	postgresDB := getPool(tempConfig)

	log.Debugf("Got connection pool for database %q", clone)
	return postgresDB
}

func getPool(pgConfig *pgxpool.Config) *pgxpool.Pool {
	var err error
	var postgresDB *pgxpool.Pool

	if err := retry.WithRetry(func() error {
		postgresDB, err = pgxpool.ConnectConfig(context.Background(), pgConfig)
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
