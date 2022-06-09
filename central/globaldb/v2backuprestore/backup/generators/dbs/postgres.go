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

// PostgresBackup is an implementation of a StreamGenerator which writes a backup of PostgresDB to the input io.Writer.
type PostgresBackup struct {
	db *pgxpool.Pool
}

// WriteTo writes a backup of Postgres to the writer
func (bu *PostgresBackup) WriteTo(ctx context.Context, out io.Writer) error {
	sourceMap, config, err := globaldb.GetPostgresConfig()
	if err != nil {
		log.Fatalf("Could not parse postgres config: %v", err)
		return err
	}

	// Set the options for pg_dump from the connection config
	options := []string{
		"-d",
		config.ConnConfig.Database,
		"-Fc", // Custom format, compressed hopefully supports stdin to restore
		"-vvv",
	}

	// Get the common DB connection info
	options = append(options, common.GetConnectionOptions(config)...)

	cmd := exec.Command("pg_dump", options...)

	// Get a pipe to the commands standard out
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	// Copy the data from stdout to the outgoing writer.
	go func() {
		defer stdout.Close()
		_, err = io.Copy(out, stdout)
	}()

	common.SetPostgresCmdEnv(cmd, sourceMap, config)

	err = common.ExecutePostgresCmd(cmd)
	if err != nil {
		return err
	}

	return nil
}
