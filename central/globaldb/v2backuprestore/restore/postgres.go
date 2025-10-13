package restore

import (
	"io"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
)

const (
	restoreDB = "central_restore"
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
	err = pgadmin.CreateDB(sourceMap, dbConfig, pgadmin.EmptyDB, restoreDB)
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

func runRestoreStream(fileReader io.Reader, sourceMap map[string]string, config *postgres.Config, restoreDB string) error {
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

func getRestoreDBName() string {
	return restoreDB
}
