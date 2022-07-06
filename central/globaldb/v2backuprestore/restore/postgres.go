package restore

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
)

const (
	// restoreSuffix - suffix for the restore database
	restoreSuffix = "_restore"
)

var (
	log = logging.LoggerForModule()
)

// LoadRestoreStream a Postgres database from a dump
func LoadRestoreStream(fileReader io.Reader) error {
	log.Info("Starting Postgres Restore")

	sourceMap, dbConfig, err := pgconfig.GetPostgresConfig()
	if err != nil {
		return errors.Wrap(err, "Could not parse postgres config")
	}

	// Build the restore database name
	restoreDB := getRestoreDBName()

	// Now recreate the DB
	err = pgadmin.CreateDB(sourceMap, dbConfig, pgadmin.AdminDB, restoreDB)
	if err != nil {
		return errors.Wrap(err, "Could not create restore database")
	}

	// Execute the restore on the temporary restore database
	err = runRestoreStream(fileReader, sourceMap, dbConfig, restoreDB)
	if err != nil {
		return errors.Wrap(err, "Could not load the Postgres backup")
	}

	log.Info("Postgres Restore Complete")
	return nil
}

func runRestoreStream(fileReader io.Reader, sourceMap map[string]string, config *pgxpool.Config, restoreDB string) error {
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
func SwitchToRestoredDB(sourceMap map[string]string, dbConfig *pgxpool.Config) error {
	log.Info("Switching to restored database")

	// Connect to different database for admin functions
	connectPool := pgadmin.AdminPool(dbConfig)
	// Close the admin connection pool
	defer connectPool.Close()

	// Restore succeeded to the separate DB, so we need to drop the original in order to rename
	// the new one.
	err := pgadmin.DropDB(sourceMap, dbConfig, pgconfig.GetActiveDB())
	if err != nil {
		log.Errorf("Could not drop the DB: %v", err)
		return err
	}

	// rename central_restore to postgres
	err = pgadmin.RenameDB(connectPool, getRestoreDBName(), pgconfig.GetActiveDB())
	if err != nil {
		log.Errorf("Could not rename the DB: %v", err)
		return err
	}

	return nil
}

// CheckIfRestoreDBExists - checks to see if a restore database exists
func CheckIfRestoreDBExists(dbConfig *pgxpool.Config) bool {
	return pgadmin.CheckIfDBExists(dbConfig, getRestoreDBName())
}

func getRestoreDBName() string {
	// Build the active database name for the connection
	return fmt.Sprintf("%s%s", config.GetConfig().CentralDB.RootDatabaseName, restoreSuffix)
}
