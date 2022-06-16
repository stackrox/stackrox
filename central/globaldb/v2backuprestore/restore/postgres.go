package restore

import (
	"context"
	"io"
	"os/exec"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
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

// LoadRestoreStream a Postgres database from a dump
func LoadRestoreStream(fileReader io.Reader) error {
	log.Info("Starting Postgres Restore")

	sourceMap, config, err := globaldb.GetPostgresConfig()
	if err != nil {
		return errors.Wrap(err, "Could not parse postgres config")
	}

	// Now recreate the DB
	err = pgadmin.CreateDB(sourceMap, config, createTemplate, restoreDB)
	if err != nil {
		return errors.Wrap(err, "Could not create restore database")
	}

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
		restoreDB,
		"--no-owner",
		"--clean",
		"--if-exists",
		"--exit-on-error",
		"-Fc",
		"-vvv",
		"--single-transaction",
	}

	// Get the common DB connection info
	options = append(options, pgadmin.GetConnectionOptions(config)...)

	cmd := exec.Command("pg_restore", options...)

	// Set stdin to be the incoming reader
	cmd.Stdin = fileReader

	pgadmin.SetPostgresCmdEnv(cmd, sourceMap, config)
	err := pgadmin.ExecutePostgresCmd(cmd)

	if err != nil {
		// Clean up the restore DB since the restore failed
		_ = pgadmin.DropDB(sourceMap, config, restoreDB)
		return errors.Wrap(err, "Unable to restore the postgres dump")
	}

	return nil
}

// SwitchToRestoredDB - switches the restore DB to be the active DB
func SwitchToRestoredDB(sourceMap map[string]string, config *pgxpool.Config) error {
	log.Info("Switching to restored database")

	// Connect to different database for admin functions
	connectPool := adminPool(config)
	// Close the admin connection pool
	defer connectPool.Close()

	// Restore succeeded to the separate DB, so we need to drop the original in order to rename
	// the new one.
	err := pgadmin.DropDB(sourceMap, config, config.ConnConfig.Database)
	if err != nil {
		log.Errorf("Could not drop the DB: %v", err)
		return err
	}

	// rename central_restore to postgres
	err = pgadmin.RenameDB(connectPool, restoreDB, config.ConnConfig.Database)
	if err != nil {
		log.Errorf("Could not rename the DB: %v", err)
		return err
	}

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
	ctx, cancel := context.WithTimeout(context.Background(), globaldb.PostgresQueryTimeout)
	defer cancel()

	// Connect to different database for admin functions
	connectPool := adminPool(config)
	// Close the admin connection pool
	defer connectPool.Close()

	existsStmt := "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_database WHERE datname = $1)"

	row := connectPool.QueryRow(ctx, existsStmt, restoreDB)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false
	}

	log.Infof("Restore database exists => %t", exists)
	return exists
}
