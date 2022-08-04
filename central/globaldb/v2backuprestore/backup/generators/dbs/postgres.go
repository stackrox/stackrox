package dbs

import (
	"context"
	"io"
	"os/exec"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
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
	sourceMap, config, err := pgconfig.GetPostgresConfig()
	if err != nil {
		log.Fatalf("Could not parse postgres config: %v", err)
		return err
	}

	// Set the options for pg_dump from the connection config
	options := []string{
		"-d",
		pgconfig.GetActiveDB(),
		"-Fc", // Custom format, compressed hopefully supports stdin to restore
		"-v",
	}

	// Get the common DB connection info
	options = append(options, pgadmin.GetConnectionOptions(config)...)

	cmd := exec.Command("pg_dump", options...)

	// Set the stdout of the command to be the output writer.
	cmd.Stdout = out

	pgadmin.SetPostgresCmdEnv(cmd, sourceMap, config)

	return pgadmin.ExecutePostgresCmd(cmd)
}
