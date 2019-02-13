package export

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
)

var (
	log = logging.LoggerForModule()

	restoreDir = filepath.Join(migrations.DBMountPath, ".restore")
)

func tryRestoreBolt(r io.Reader, outDir string) error {
	boltFilePath := path.Join(outDir, bolthelper.DBFileName)
	boltFile, err := os.OpenFile(boltFilePath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("could not create bolt file: %v", err)
	}
	_, err = io.Copy(boltFile, r)
	boltFile.Close()

	if err != nil {
		return fmt.Errorf("could not write bolt file: %v", err)
	}

	opts := *bolt.DefaultOptions
	opts.ReadOnly = true
	db, err := bolt.Open(boltFilePath, 0600, &opts)
	if err != nil {
		return fmt.Errorf("could not open bolt database: %v", err)
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("could not close bolt database after opening: %v", err)
	}

	return nil
}

func tryRestoreBadger(r io.Reader, outDir string) error {
	badgerDirPath := path.Join(outDir, badgerhelper.BadgerDBDirName)
	if err := os.Mkdir(badgerDirPath, 0600); err != nil {
		return fmt.Errorf("could not create badger database directory: %v", err)
	}

	db, err := badgerhelper.New(badgerDirPath)
	if err != nil {
		return fmt.Errorf("could not create new badger DB in empty dir: %v", err)
	}

	if err := db.Load(r); err != nil {
		return fmt.Errorf("could not load badger DB backup: %v", err)
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("could not close badger DB after loading: %v", err)
	}

	return nil
}

func tryRestoreZip(backupFile *os.File, outPath string) error {
	stat, err := backupFile.Stat()
	if err != nil {
		return fmt.Errorf("could not stat file: %v", err)
	}
	zipReader, err := zip.NewReader(backupFile, stat.Size())
	if err != nil {
		return fmt.Errorf("could not read file as ZIP archive: %v", err)
	}

	hasBolt := false
	hasBadger := false

	for _, f := range zipReader.File {
		if f.Name == boltFileName {
			r, err := f.Open()
			if err != nil {
				return fmt.Errorf("could not open %s in ZIP archive: %v", boltFileName, err)
			}
			err = tryRestoreBolt(r, outPath)
			r.Close()
			if err != nil {
				return fmt.Errorf("could not restore bolt DB from file %s in ZIP archive: %v", boltFileName, err)
			}
			hasBolt = true
		} else if f.Name == badgerFileName {
			r, err := f.Open()
			if err != nil {
				return fmt.Errorf("could not open %s in ZIP archive: %v", badgerFileName, err)
			}
			err = tryRestoreBadger(r, outPath)
			r.Close()
			if err != nil {
				return fmt.Errorf("could not restore badger DB from file %s in ZIP archive: %v", badgerFileName, err)
			}
			hasBadger = true
		}
	}

	if !hasBolt {
		return fmt.Errorf("bolt backup file %s not found in ZIP archive", boltFileName)
	}
	if !hasBadger {
		return fmt.Errorf("badger backup file %s not found in ZIP archive", badgerFileName)
	}
	return nil
}

func removeChildren(path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := os.RemoveAll(filepath.Join(path, f.Name())); err != nil {
			return fmt.Errorf("could not remove file %s: %v", f.Name(), err)
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
		return fmt.Errorf("could not clean up directory %s after unsuccessful restore attempt: %v", outPath, err)
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
	tempRestoreDir, err := ioutil.TempDir(migrations.DBMountPath, ".restore-")
	if err != nil {
		return fmt.Errorf("could not create a temporary restore directory: %v", err)
	}

	if err := tryRestore(backupFile, tempRestoreDir); err != nil {
		os.RemoveAll(tempRestoreDir)
		return fmt.Errorf("could not restore database backup: %v", err)
	}

	if err := os.Rename(tempRestoreDir, restoreDir); err != nil {
		os.RemoveAll(tempRestoreDir)
		return fmt.Errorf("could not rename temporary restore directory to canonical location: %v", err)
	}

	return nil
}
