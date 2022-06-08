package dbs

import (
	"context"
	"io"
	"os/exec"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	dumpTmpPath = "pg_backup"
)

var (
	log = logging.LoggerForModule()
)

// NewPostgresBackup returns a generator for Postgres backups.
// We take in the connection to connect to the DB
func NewPostgresBackup(db *pgxpool.Pool) *PostgresBackup {
	return &PostgresBackup{
		db: db,
	}
}

// PostgresBackup is an implementation of a postgres connection pool
type PostgresBackup struct {
	db *pgxpool.Pool
}

// getPostgresSize Method to calculate size
func (bu *PostgresBackup) getPostgresSize(ctx context.Context) (int64, error) {
	row := bu.db.QueryRow(ctx, "SELECT pg_database_size('postgres')")
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// WriteTo writes a backup of BoltDB to the input io.Writer.
func (bu *PostgresBackup) WriteTo(ctx context.Context, out io.Writer) error {
	sourceMap, config, err := globaldb.GetPostgresConfig()
	if err != nil {
		log.Fatalf("Could not parse postgres config: %v", err)
		return err
	}

	//backupPath, err := bu.findScratchPath(ctx)
	//if err != nil {
	//	return err
	//}

	//tempfile := backupPath + "/" + backup.PostgresFileName

	// Set the options for pg_dump from the connection config
	options := []string{
		"-d",
		config.ConnConfig.Database,
		"-Fc", // Custom format, compressed hopefully supports stdin to restore
		//"-Fd", // Custom format.  Compressed files written to a directory.
		//"-f",
		//tempfile,
		//"-j", // Allows for work to be spread across jobs
		//"5",  // The number of jobs
	}

	// Get the common DB connection info
	options = append(options, common.GetConnectionOptions(config)...)

	cmd := exec.Command("pg_dump", options...)

	// Get a pipe to the commands standard in
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	// Copy the data into the commands stdin
	log.Info("SHREWS -- runRestoreStream -- copying data to the stdin")
	//_, err = io.Copy(stdin, fileReader)
	//if err != nil {
	//	log.Fatal(err)
	//}

	go func() {
		defer stdout.Close()
		_, err = io.Copy(out, stdout)
		//io.WriteString(stdin, "values written to stdin are passed to cmd's standard input")
	}()

	common.SetPostgresCmdEnv(cmd, sourceMap, config)

	err = common.ExecutePostgresCmd(cmd)
	if err != nil {
		return err
	}

	//f, err := os.OpenFile(tempfile, os.O_CREATE|os.O_RDWR, os.FileMode(os.O_RDONLY))
	//dbFileReader := io.ReadCloser(f)
	//defer utils.IgnoreError(dbFileReader.Close)
	//
	//_, err = io.Copy(out, dbFileReader)
	//if err != nil {
	//	return errors.Wrap(err, "could not copy bolt backup file")
	//}
	return nil
}

// WriteDirectory writes a backup of Postgres to the input path.
func (bu *PostgresBackup) WriteDirectory(ctx context.Context) (string, error) {
	sourceMap, config, err := globaldb.GetPostgresConfig()
	if err != nil {
		log.Fatalf("Could not parse postgres config: %v", err)
		return "", err
	}

	backupPath, err := bu.findScratchPath(ctx)
	if err != nil {
		return "", err
	}

	// Set the options for pg_dump from the connection config
	options := []string{
		"-d",
		config.ConnConfig.Database,
		"-Fc", // Custom format, compressed hopefully supports stdin to restore
		//"-Fd", // Custom format.  Compressed files written to a directory.
		"-f",
		backupPath,
		//"-j", // Allows for work to be spread across jobs
		//"5",  // The number of jobs
	}

	// Get the common DB connection info
	options = append(options, common.GetConnectionOptions(config)...)

	cmd := exec.Command("pg_dump", options...)

	common.SetPostgresCmdEnv(cmd, sourceMap, config)

	err = common.ExecutePostgresCmd(cmd)
	if err != nil {
		return "", err
	}

	return backupPath, nil
}

func (bu *PostgresBackup) findScratchPath(ctx context.Context) (string, error) {
	dbSize, err := bu.getPostgresSize(ctx)
	if err != nil {
		return "", err
	}

	return common.FindTmpPath(dbSize, dumpTmpPath)
}
