package restore

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
)

const (
	// restoreDB - temporary database to apply the postgres dump
	restoreDB = "central_restore"
	// createTemplate - template DB to base the temporary DB off of.
	createTemplate = "template0"
	// connectDB - database we can connect to in order to perform the rename
	// TODO: ROX-11272 We should reserve a database for admin type functionality
	// such as what we are doing with a restore.  This will need to be updated
	// with that change.
	connectDB = "template1"
)

var (
	log = logging.LoggerForModule()

	postgresOpenRetries        = 10
	postgresTimeBetweenRetries = 10 * time.Second
)

// dropDB - drops a database.  This is so we can restore to a new database then flip the name
// of the restored database to the original for consistency
func dropDB(sourceMap map[string]string, config *pgxpool.Config, databaseName string) error {
	// Set the options for pg_dump from the connection config
	options := []string{
		"-f",
		"--if-exists",
		databaseName,
	}

	// Get the common DB connection info
	options = append(options, common.GetConnectionOptions(config)...)

	cmd := exec.Command("dropdb", options...)

	common.SetPostgresCmdEnv(cmd, sourceMap, config)
	err := common.ExecutePostgresCmd(cmd)
	if err != nil {
		log.Errorf("Unable to drop database %s", databaseName)
		return err
	}

	return nil
}

// createDB - creates a temporary database from template for restoring
func createDB(sourceMap map[string]string, config *pgxpool.Config) error {
	// Set the options for pg_dump from the connection config
	options := []string{
		"-T",
		createTemplate,
		restoreDB,
	}

	// Get the common DB connection info
	options = append(options, common.GetConnectionOptions(config)...)

	cmd := exec.Command("createdb", options...)

	common.SetPostgresCmdEnv(cmd, sourceMap, config)

	return common.ExecutePostgresCmd(cmd)
}

// LoadRestoreStream a Postgres database from a dump
func LoadRestoreStream(fileReader io.Reader) error {
	log.Info("Starting Postgres Restore")

	// Close the connection pool so data is not in flight while trying to restore.
	globaldb.ClosePostgresPool()

	sourceMap, config, err := globaldb.GetPostgresConfig()
	if err != nil {
		return errors.Wrap(err, "Could not parse postgres config")
	}

	//// Now recreate the DB
	//err = createDB(sourceMap, config)
	//if err != nil {
	//	return errors.Wrap(err, "Could not create restore database")
	//}

	// Execute the restore on the temporary restore database
	err = runRestoreStream(fileReader, sourceMap, config)
	if err != nil {
		return errors.Wrap(err, "Could not load the Postgres backup")
	}

	log.Info("Postgres Restore Complete")
	return nil
}

func runRestoreStream(fileReader io.Reader, sourceMap map[string]string, config *pgxpool.Config) error {
	// Set the options for pg_dump from the connection config
	options := []string{
		"-d",
		config.ConnConfig.Database,
		//restoreDB,
		"--no-owner",
		"--clean",
		"--if-exists",
		"--exit-on-error",
		"-Fc",
		"-vvv",
		"--single-transaction",
	}

	// Get the common DB connection info
	options = append(options, common.GetConnectionOptions(config)...)

	cmd := exec.Command("pg_restore", options...)

	// Get a pipe to the commands standard in
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "Unable to get pipe to feed restore command")
	}

	// Copy the data into the commands stdin
	go func() {
		defer stdin.Close()
		_, err = io.Copy(stdin, fileReader)
	}()

	common.SetPostgresCmdEnv(cmd, sourceMap, config)
	err = common.ExecutePostgresCmd(cmd)

	if err != nil {
		// Clean up the restore DB since the restore failed
		_ = dropDB(sourceMap, config, restoreDB)
		return errors.Wrap(err, "Unable to restore the postgres dump")
	}

	return nil
}

func renameRestoreDB(connectPool *pgxpool.Pool, updatedDB, primaryDB string) error {
	log.Info("Flipping the restored DB to the active DB")
	sqlStmt := fmt.Sprintf("ALTER DATABASE %s RENAME TO %s", updatedDB, primaryDB)

	_, err := connectPool.Exec(context.TODO(), sqlStmt)

	return err
}

// SwitchToRestoredDB - switches the restore DB to be the active DB
func SwitchToRestoredDB(sourceMap map[string]string, config *pgxpool.Config) error {
	log.Info("Switching to restored database")

	// Connect to different database for admin functions
	connectPool := adminPool(config)

	// Restore succeeded to the separate DB, so we need to drop the original in order to rename
	// the new one.
	err := dropDB(sourceMap, config, config.ConnConfig.Database)
	if err != nil {
		log.Errorf("Could not drop the DB: %v", err)
		return err
	}

	// rename central_restore to postgres
	err = renameRestoreDB(connectPool, restoreDB, config.ConnConfig.Database)
	if err != nil {
		log.Errorf("Could not rename the DB: %v", err)
		return err
	}

	// Close the admin connection pool
	connectPool.Close()

	return nil
}

func adminPool(config *pgxpool.Config) *pgxpool.Pool {
	var err error
	var postgresDB *pgxpool.Pool

	// Clone config to connect to template DB
	tempConfig := config.Copy()

	// Need to connect on a static DB so we can rename the used DBs.
	tempConfig.ConnConfig.Database = connectDB

	if err := retry.WithRetry(func() error {
		postgresDB, err = pgxpool.ConnectConfig(context.Background(), tempConfig)
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

// CheckIfRestoreDBExists - checks to see if a restore database exists
func CheckIfRestoreDBExists(config *pgxpool.Config) bool {
	log.Info("CheckIfRestoreDBExists")

	// Connect to different database for admin functions
	connectPool := adminPool(config)

	existsStmt := "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_database WHERE datname = $1)"

	row := connectPool.QueryRow(context.Background(), existsStmt, restoreDB)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false
	}

	// Close the admin connection pool
	connectPool.Close()

	log.Infof("Restore database exists => %t", exists)
	return exists

}
