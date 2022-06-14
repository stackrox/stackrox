package dbs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/stackrox/pkg/config"
	"github.com/stackrox/stackrox/pkg/logging"
)

const (
	dumpTmpPath    = "pg_backup"
	dbPasswordFile = "/run/secrets/stackrox.io/db-password/password"
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

// WriteDirectory writes a backup of RocksDB to the input path.
func (bu *PostgresBackup) WriteDirectory(ctx context.Context) (string, error) {
	centralConfig := config.GetConfig()
	password, err := os.ReadFile(dbPasswordFile)
	if err != nil {
		log.Fatalf("pgsql: could not load password file %q: %v", dbPasswordFile, err)
		return "", err
	}
	source := fmt.Sprintf("%s password=%s", centralConfig.CentralDB.Source, password)
	sourceMap := bu.parseSource(source)

	config, err := pgxpool.ParseConfig(source)
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
		"-U",
		config.ConnConfig.User,
		"-h",
		config.ConnConfig.Host,
		"-p",
		strconv.FormatUint(uint64(config.ConnConfig.Port), 10),
		"-d",
		config.ConnConfig.Database,
		"-Fd",
		"-f",
		backupPath,
		"-j",
		"5",
	}

	cmd := exec.Command("pg_dump", options...)
	cmd.Env = os.Environ()

	if _, found := sourceMap["sslmode"]; found {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGSSLMODE=%s", sourceMap["sslmode"]))
	}
	if _, found := sourceMap["sslrootcert"]; found {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGSSLROOTCERT=%s", sourceMap["sslrootcert"]))
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", config.ConnConfig.Password))

	// Run the command
	err = cmd.Start()
	if err != nil {
		return "", err
	}
	err = cmd.Wait()

	if exitError, ok := err.(*exec.ExitError); ok {
		log.Error(exitError)
		return "", err
	}

	return backupPath, nil
}

func (bu *PostgresBackup) findScratchPath(ctx context.Context) (string, error) {
	dbSize, err := bu.getPostgresSize(ctx)
	if err != nil {
		return "", err
	}

	return findTmpPath(dbSize, dumpTmpPath)
}

func (bu *PostgresBackup) parseSource(source string) map[string]string {
	sourceSlice := strings.Split(source, " ")
	sourceMap := make(map[string]string)
	for _, pair := range sourceSlice {
		configSetting := strings.Split(pair, "=")
		sourceMap[configSetting[0]] = configSetting[1]
	}

	return sourceMap
}
