package dbs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/fsutils"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	dumpTmpPath    = "pg_backup"
	dbPasswordFile = "/run/secrets/stackrox.io/db-password/password"

	sizeBufferMargin = 0.5
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
	log.Infof("SHREWS -- %s", source)

	config, err := pgxpool.ParseConfig(source)
	if err != nil {
		log.Fatalf("Could not parse postgres config: %v", err)
		return "", err
	}

	backupPath, err := bu.findScratchPath(ctx)
	if err != nil {
		return "", err
	}

	log.Infof("SHREWS -- backup path -- %s", backupPath)

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
	}

	cmd := exec.Command("/usr/bin/pg_dump", options...)
	cmd.Env = os.Environ()

	// TODO:  Try to parse source to set these.
	// Setup the environment variables specific to the command
	cmd.Env = append(cmd.Env, "PGSSLMODE=verify-full")
	cmd.Env = append(cmd.Env, "PGSSLROOTCERT=/run/secrets/stackrox.io/certs/ca.pem")
	//pwEnv := "PGPASSWORD=" + fmt.Sprintf("%s", pw)
	pwEnv := "PGPASSWORD=" + fmt.Sprintf("%s", config.ConnConfig.Password)
	cmd.Env = append(cmd.Env, pwEnv)

	log.Info(cmd.String())
	log.Info(cmd.Env)

	// Direct the command stdout to the destination io.Writer
	//cmd.Stdout = out

	// Run the command
	cmd.Start()
	err = cmd.Wait()

	if exitError, ok := err.(*exec.ExitError); ok {
		log.Info(err)
		log.Info(exitError)
		return "", err
	}

	log.Infof("SHREWS -- Config => %s", config)
	log.Info("Performed Dump")
	return backupPath, nil
}

func (bu *PostgresBackup) findScratchPath(ctx context.Context) (string, error) {
	dbSize, err := bu.getPostgresSize(ctx)
	log.Infof("SHREWS => %d", dbSize)
	if err != nil {
		return "", err
	}
	requiredBytes := float64(dbSize) * (1.0 + sizeBufferMargin)

	// Check tmp for space to produce a backup.
	tmpDir, err := os.MkdirTemp("", dumpTmpPath)
	if err != nil {
		return "", err
	}
	tmpBytesAvailable, err := fsutils.AvailableBytesIn(tmpDir)
	if err != nil {
		return "", errors.Wrapf(err, "unable to calculates size of %s", tmpDir)
	}
	if float64(tmpBytesAvailable) > requiredBytes {
		return tmpDir, nil
	}

	// If there isn't enough space there, try using PVC to create it.
	pvcDir, err := os.MkdirTemp(globaldb.PVCPath, dumpTmpPath)
	if err != nil {
		return "", err
	}
	pvcBytesAvailable, err := fsutils.AvailableBytesIn(pvcDir)
	if err != nil {
		return "", errors.Wrapf(err, "unable to calculates size of %s", pvcDir)
	}
	if float64(pvcBytesAvailable) > requiredBytes {
		return pvcDir, nil
	}

	// If neither had enough space, return an error.
	return "", errors.Errorf("required %f bytes of space, found %f bytes in %s and %f bytes on PVC, cannot backup", requiredBytes, float64(tmpBytesAvailable), os.TempDir(), float64(pvcBytesAvailable))
}
