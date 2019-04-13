package export

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/utils"
)

func scrubSecretsAndCompact(dbDumpFile string) (string, error) {
	defer func() {
		_ = os.Remove(dbDumpFile)
	}()

	oldDB, err := bolt.Open(dbDumpFile, 0600, bolt.DefaultOptions)
	oldDB.NoSync = true
	if err != nil {
		return "", errors.Wrap(err, "could not open database dump")
	}

	if err := scrubSensitiveData(oldDB); err != nil {
		return "", errors.Wrap(err, "could not scrub secrets from database")
	}

	compactedDBFile, err := ioutil.TempFile("", "bolt-backup-compacted-")
	if err != nil {
		return "", errors.Wrap(err, "could not create temporary file for bolt backup")
	}
	compactedDBFileName := compactedDBFile.Name()
	if err := compactedDBFile.Close(); err != nil {
		return "", errors.Wrap(err, "could not close compacted database file")
	}
	if err := os.Remove(compactedDBFileName); err != nil {
		return "", errors.Wrapf(err, "could not remove compacted file %s", compactedDBFileName)
	}

	db, err := bolt.Open(compactedDBFileName, 0600, bolt.DefaultOptions)
	if err != nil {
		return "", errors.Wrap(err, "could not create a new bolt database for compaction")
	}
	db.NoSync = true

	if err := bolthelper.Compact(db, oldDB); err != nil {
		return "", errors.Wrap(err, "could not compact database")
	}

	if err := oldDB.Close(); err != nil {
		log.Errorf("Could not close old database: %v", err)
	}
	if err := db.Close(); err != nil {
		// Try remove in a best-effort fashion
		_ = os.Remove(compactedDBFileName)
		return "", errors.Wrap(err, "could not close compacted database")
	}

	return compactedDBFileName, nil
}

func backupBolt(db *bolt.DB, out io.Writer, scrubSecrets bool) error {
	tempFile, err := ioutil.TempFile("", "bolt-backup-")
	if err != nil {
		return errors.Wrap(err, "could not create temporary file for bolt backup")
	}
	defer func() {
		_ = os.Remove(tempFile.Name())
	}()
	defer utils.IgnoreError(tempFile.Close)

	err = db.View(func(tx *bolt.Tx) error {
		_, err := tx.WriteTo(out)
		return err
	})
	if err != nil {
		return errors.Wrap(err, "could not dump bolt database")
	}

	_, err = tempFile.Seek(0, 0)
	if err != nil {
		return errors.Wrap(err, "could not rewind to beginning of file")
	}

	dbFileReader := io.ReadCloser(tempFile)
	if scrubSecrets {
		tempFileName := tempFile.Name()
		if err := tempFile.Close(); err != nil {
			return errors.Wrap(err, "could not close database dump file")
		}

		compactedTempFilePath, err := scrubSecretsAndCompact(tempFileName)
		if err != nil {
			return errors.Wrap(err, "could not compact database")
		}

		compactedTempFile, err := os.Open(compactedTempFilePath)
		if err != nil {
			return errors.Wrap(err, "could not open compacted database file")
		}

		dbFileReader = compactedTempFile
	}

	defer utils.IgnoreError(dbFileReader.Close)

	_, err = io.Copy(out, dbFileReader)
	return err
}

func backupBadger(db *badger.DB, out io.Writer) error {
	tempFile, err := ioutil.TempFile("", "badger-backup-")
	if err != nil {
		return errors.Wrap(err, "could not create temporary file for badger backup")
	}
	defer func() {
		_ = os.Remove(tempFile.Name())
	}()
	defer utils.IgnoreError(tempFile.Close)

	_, err = db.Backup(tempFile, 0)
	if err != nil {
		return errors.Wrap(err, "could not create badger backup")
	}

	_, err = tempFile.Seek(0, 0)
	if err != nil {
		return errors.Wrap(err, "could not rewind to beginning of file")
	}

	_, err = io.Copy(out, tempFile)
	return err
}

// Backup backs up the given databases (optionally removing secrets) and writes a ZIP archive to the given writer.
func Backup(boltDB *bolt.DB, badgerDB *badger.DB, out io.Writer, scrubSecrets bool) error {
	zipWriter := zip.NewWriter(out)
	defer utils.IgnoreError(zipWriter.Close)
	boltWriter, err := zipWriter.Create(boltFileName)
	if err != nil {
		return err
	}
	if err := backupBolt(boltDB, boltWriter, scrubSecrets); err != nil {
		return errors.Wrap(err, "backing up bolt")
	}
	badgerWriter, err := zipWriter.Create(badgerFileName)
	if err != nil {
		return err
	}
	if err := backupBadger(badgerDB, badgerWriter); err != nil {
		return errors.Wrap(err, "backing up badger")
	}
	return zipWriter.Close()
}
