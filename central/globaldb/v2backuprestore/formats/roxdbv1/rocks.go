//go:build amd64

package roxdbv1

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/pkg/rocksdb"
	pkgTar "github.com/stackrox/rox/pkg/tar"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/tecbot/gorocksdb"
)

const (
	rocksDBPath = "rocksdb"
	scratchPath = "rocksdbScratch"
)

func restoreRocksDB(ctx common.RestoreFileContext, fileReader io.Reader, _ int64) error {
	absDirPath, err := ctx.Mkdir(rocksDBPath, 0700)
	if err != nil {
		return errors.Wrap(err, "could not create rocksdb database directory")
	}

	tmpDir, err := os.MkdirTemp("", scratchPath)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(func() error { return os.RemoveAll(tmpDir) })

	err = pkgTar.ToPath(tmpDir, fileReader)
	if err != nil {
		return errors.Wrap(err, "unable to untar rocksdb backup to scratch path")
	}

	// Generate the backup files in the directory.
	backupEngine, err := gorocksdb.OpenBackupEngine(rocksdb.GetRocksDBOptions(), tmpDir)
	if err != nil {
		return errors.Wrap(err, "error initializing backup process")
	}
	defer backupEngine.Close()

	// Check DB size vs. availability.
	err = backupEngine.RestoreDBFromLatestBackup(absDirPath, absDirPath, gorocksdb.NewRestoreOptions())
	if err != nil {
		return errors.Wrap(err, "error generating backup directory")
	}

	ctx.CheckAsync(func(_ common.RestoreProcessContext) error { return validateRocksDB(absDirPath) })
	return os.RemoveAll(tmpDir)
}

func validateRocksDB(dbPath string) error {
	rocksDB, err := rocksdb.New(dbPath)
	if err != nil {
		return errors.Wrap(err, "unable to open rocksdb path after restore")
	}
	rocksDB.Close()
	return nil
}
