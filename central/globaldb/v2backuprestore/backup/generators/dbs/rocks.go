package dbs

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/option"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/rocksdb/metrics"
	"github.com/tecbot/gorocksdb"
)

const (
	tmpPath = "rocksdb"
)

// NewRocksBackup returns a generator for RocksDB backups.
// We take in the path that holds the DB as well so that we can estimate the db's size with statfs_t.
func NewRocksBackup(db *rocksdb.RocksDB) *RocksBackup {
	return &RocksBackup{
		db: db,
	}
}

// RocksBackup is an implementation of a DirectoryGenerator which writes a backup of RocksDB to the input path.
type RocksBackup struct {
	db *rocksdb.RocksDB
}

// WriteDirectory writes a backup of RocksDB to the input path.
func (rgen *RocksBackup) WriteDirectory(ctx context.Context) (string, error) {
	if err := rgen.db.IncRocksDBInProgressOps(); err != nil {
		return "", err
	}
	defer rgen.db.DecRocksDBInProgressOps()

	path, err := findScratchPath()
	if err != nil {
		return "", errors.Wrap(err, "could not find space sufficient for backup generation")
	}

	// Generate the backup files in the directory.
	backupEngine, err := gorocksdb.OpenBackupEngine(rocksdb.GetRocksDBOptions(), path)
	if err != nil {
		return "", errors.Wrap(err, "error initializing backup process")
	}
	defer backupEngine.Close()

	// Check DB size vs. availability.
	err = backupEngine.CreateNewBackup(rgen.db.DB)
	if err != nil {
		return "", errors.Wrap(err, "error generating backup directory")
	}
	return path, nil
}

func findScratchPath() (string, error) {
	dbSize, err := getRocksDBSize()
	if err != nil {
		return "", err
	}

	return findTmpPath(dbSize, tmpPath)
}

// Get the number of bytes used by files stored for the db.
func getRocksDBSize() (int64, error) {
	size, err := fileutils.DirectorySize(metrics.GetRocksDBPath(option.CentralOptions.DBPathBase))
	if err != nil {
		return 0, err
	}
	return size, nil
}
