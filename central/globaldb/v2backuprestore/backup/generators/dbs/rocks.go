package dbs

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/option"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/fsutils"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/rocksdb/metrics"
	"github.com/tecbot/gorocksdb"
)

const (
	tmpPath = "rocksdb"
)

// marginOfSafety is how much more free space we want available then the current DB space used before we perform a
// backup.
var marginOfSafety = 0.5

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
	requiredBytes := float64(dbSize) * (1.0 + marginOfSafety)

	// Check tmp for space to produce a backup.
	tmpDir, err := os.MkdirTemp("", tmpPath)
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
	pvcDir, err := os.MkdirTemp(globaldb.PVCPath, tmpPath)
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

// Get the number of bytes used by files stored for the db.
func getRocksDBSize() (int64, error) {
	size, err := fileutils.DirectorySize(metrics.GetRocksDBPath(option.CentralOptions.DBPathBase))
	if err != nil {
		return 0, err
	}
	return size, nil
}
