package restore

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
)

const (
	restoreDB      = "central_restore"
	connectDB      = "template1"
	createTemplate = "template0"
)

var (
	log = logging.LoggerForModule()

	postgresOpenRetries        = 10
	postgresTimeBetweenRetries = 10 * time.Second
	postgresDB                 *pgxpool.Pool
)

// dropDB - drops a database.  This is so we can restore to a new database then flip the name
// of the restored database to the original for consistency
func dropDB(sourceMap map[string]string, config *pgxpool.Config) error {
	// Set the options for pg_dump from the connection config
	options := []string{
		"-U",
		config.ConnConfig.User,
		"-h",
		config.ConnConfig.Host,
		"-p",
		strconv.FormatUint(uint64(config.ConnConfig.Port), 10),
		"-f",
		"--if-exists",
		config.ConnConfig.Database,
	}

	cmd := exec.Command("dropdb", options...)

	common.SetPostgresCmdEnv(cmd, sourceMap, config)
	err := common.ExecutePostgresCmd(cmd)
	if err != nil {
		return err
	}

	return nil
}

// createDB - creates a temporary database from template for restoring
func createDB(sourceMap map[string]string, config *pgxpool.Config) error {
	// Set the options for pg_dump from the connection config
	options := []string{
		"-U",
		config.ConnConfig.User,
		"-h",
		config.ConnConfig.Host,
		"-p",
		strconv.FormatUint(uint64(config.ConnConfig.Port), 10),
		"-T",
		createTemplate,
		restoreDB,
	}

	cmd := exec.Command("createdb", options...)

	common.SetPostgresCmdEnv(cmd, sourceMap, config)
	err := common.ExecutePostgresCmd(cmd)
	if err != nil {
		return err
	}

	return nil
}

func runRestore(dumpPath string, sourceMap map[string]string, config *pgxpool.Config) error {
	// Set the options for pg_dump from the connection config
	options := []string{
		"-U",
		config.ConnConfig.User,
		"-h",
		config.ConnConfig.Host,
		"-p",
		strconv.FormatUint(uint64(config.ConnConfig.Port), 10),
		"-d",
		restoreDB,
		"--no-owner",
		//"-C",
		"--clean",
		"--if-exists",
		"--exit-on-error",
		"-Fd",
		"-j",
		"5",
		"-vvv",
		dumpPath,
	}

	cmd := exec.Command("pg_restore", options...)

	common.SetPostgresCmdEnv(cmd, sourceMap, config)
	err := common.ExecutePostgresCmd(cmd)
	if err != nil {
		return err
	}

	return nil
}

// LoadRestore a Postgres database from a dump
func LoadRestore(dumpPath string) error {
	log.Info("Starting Postgres Restore")
	sourceMap, config, err := globaldb.GetPostgresConfig()
	if err != nil {
		log.Fatalf("Could not parse postgres config: %v", err)
		return err
	}

	// Now recreate the DB
	err = createDB(sourceMap, config)
	if err != nil {
		log.Fatalf("Could not create restore database: %v", err)
		return err
	}

	// Execute the restore on the temporary restore database
	err = runRestore(dumpPath, sourceMap, config)
	if err != nil {
		log.Errorf("Could load the Postgres backup: %v", err)
		return err
	}

	log.Info("Postgres Restore Complete")
	return nil
}

func renameRestoreDB(connectPool *pgxpool.Pool) error {
	sqlStmt := fmt.Sprintf("ALTER DATABASE %s RENAME TO %s", restoreDB, "postgres")

	_, err := connectPool.Exec(context.TODO(), sqlStmt)
	if err != nil {
		log.Errorf("Could rename the DB: %v", err)
	}

	return err
}

func SwitchToRestoredDB() error {
	log.Info("Switching to restored database")
	sourceMap, config, err := globaldb.GetPostgresConfig()
	if err != nil {
		log.Fatalf("Could not parse postgres config: %v", err)
		return err
	}

	// connect on template1
	connectPool := adminPool(config)

	// Restore succeeded to the separate DB, so we need to drop the original in order to rename
	// the new one.
	err = dropDB(sourceMap, config)
	if err != nil {
		log.Errorf("Could drop the DB: %v", err)
		return err
	}

	// rename central_restore to postgres
	err = renameRestoreDB(connectPool)
	if err != nil {
		return err
	}

	// Close the connection pool on template1 to support changing name.
	connectPool.Close()

	return nil
}

func adminPool(config *pgxpool.Config) *pgxpool.Pool {
	var err error

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

func CheckIfRestoreDBExists() bool {
	log.Info("CheckIfRestoreDBExists")
	_, config, err := globaldb.GetPostgresConfig()
	if err != nil {
		log.Fatalf("Could not parse postgres config: %v", err)
		return false
	}
	// connect on template1
	connectPool := adminPool(config)

	existsStmt := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_database WHERE datname = $1)")

	row := connectPool.QueryRow(context.Background(), existsStmt, restoreDB)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false
	}

	log.Infof("Restore database exists => %t", exists)
	return exists

}
