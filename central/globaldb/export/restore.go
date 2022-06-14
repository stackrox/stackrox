package export

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/backup"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/errorhelpers"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/migrations"
	"github.com/stackrox/stackrox/pkg/odirect"
	bolt "go.etcd.io/bbolt"
)

var (
	log = logging.LoggerForModule()

	restoreDir = filepath.Join(migrations.DBMountPath(), ".restore")
)

func tryRestoreBolt(r io.Reader, outDir string) error {
	odirectFlag := odirect.GetODirectFlag()
	boltFilePath := path.Join(outDir, bolthelper.DBFileName)
	boltFile, err := os.OpenFile(boltFilePath, os.O_CREATE|os.O_RDWR|odirectFlag, 0600)
	if err != nil {
		return errors.Wrap(err, "could not create bolt file")
	}
	_, err = io.Copy(boltFile, r)
	_ = boltFile.Close()

	if err != nil {
		return errors.Wrap(err, "could not write bolt file")
	}

	opts := *bolt.DefaultOptions
	opts.ReadOnly = true
	db, err := bolt.Open(boltFilePath, 0600, &opts)
	if err != nil {
		return errors.Wrap(err, "could not open bolt database")
	}
	if err := db.Close(); err != nil {
		return errors.Wrap(err, "could not close bolt database after opening")
	}

	return nil
}

func tryRestoreZip(backupFile *os.File, outPath string) error {
	stat, err := backupFile.Stat()
	if err != nil {
		return errors.Wrap(err, "could not stat file")
	}
	zipReader, err := zip.NewReader(backupFile, stat.Size())
	if err != nil {
		return errors.Wrap(err, "could not read file as ZIP archive")
	}

	hasBolt := false

	for _, f := range zipReader.File {
		if f.Name == backup.BoltFileName {
			r, err := f.Open()
			if err != nil {
				return errors.Wrapf(err, "could not open %s in ZIP archive", backup.BoltFileName)
			}
			err = tryRestoreBolt(r, outPath)
			_ = r.Close()
			if err != nil {
				return errors.Wrapf(err, "could not restore bolt DB from file %s in ZIP archive", backup.BoltFileName)
			}
			hasBolt = true
		}
	}

	if !hasBolt {
		return fmt.Errorf("bolt backup file %s not found in ZIP archive", backup.BoltFileName)
	}
	return nil
}

func removeChildren(path string) error {
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := os.RemoveAll(filepath.Join(path, f.Name())); err != nil {
			return errors.Wrapf(err, "could not remove file %s", f.Name())
		}
	}
	return nil
}

func tryRestore(backupFile *os.File, outPath string) error {
	var allErrs errorhelpers.ErrorList
	zipErr := tryRestoreZip(backupFile, outPath)
	if zipErr == nil {
		return nil
	}
	allErrs.AddStringf("treating input as ZIP file: %v", zipErr)
	if err := removeChildren(outPath); err != nil {
		return errors.Wrapf(err, "could not clean up directory %s after unsuccessful restore attempt", outPath)
	}

	boltErr := tryRestoreBolt(backupFile, outPath)
	if boltErr == nil {
		return nil
	}
	allErrs.AddStringf("treating input as bolt snapshot: %v", boltErr)
	return allErrs.ToError()
}

// Restore restores a backup from a file.
func Restore(backupFile *os.File) error {
	tempRestoreDir, err := os.MkdirTemp(migrations.DBMountPath(), ".restore-")
	if err != nil {
		return errors.Wrap(err, "could not create a temporary restore directory")
	}

	if err := tryRestore(backupFile, tempRestoreDir); err != nil {
		_ = os.RemoveAll(tempRestoreDir)
		return errors.Wrap(err, "could not restore database backup")
	}

	if err := os.Symlink(filepath.Base(tempRestoreDir), restoreDir); err != nil {
		_ = os.RemoveAll(tempRestoreDir)
		return errors.Wrap(err, "could not link temporary restore directory to canonical location")
	}

	return nil
}
