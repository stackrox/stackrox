package export

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/pkg/bolthelper"
)

func scrubSecretsAndCompact(dbDumpFile string) (string, error) {
	defer os.Remove(dbDumpFile)

	oldDB, err := bolt.Open(dbDumpFile, 0600, bolt.DefaultOptions)
	oldDB.NoSync = true
	if err != nil {
		return "", fmt.Errorf("could not open database dump: %v", err)
	}

	if err := scrubSensitiveData(oldDB); err != nil {
		return "", fmt.Errorf("could not scrub secrets from database: %v", err)
	}

	compactedDBFile, err := ioutil.TempFile("", "bolt-backup-compacted-")
	if err != nil {
		return "", fmt.Errorf("could not create temporary file for bolt backup: %v", err)
	}
	compactedDBFileName := compactedDBFile.Name()
	if err := compactedDBFile.Close(); err != nil {
		return "", fmt.Errorf("could not close compacted database file: %v", err)
	}
	os.Remove(compactedDBFileName)

	db, err := bolt.Open(compactedDBFileName, 0600, bolt.DefaultOptions)
	if err != nil {
		return "", fmt.Errorf("could not create a new bolt database for compaction: %v", err)
	}
	db.NoSync = true

	if err := bolthelper.Compact(db, oldDB); err != nil {
		return "", fmt.Errorf("could not compact database: %v", err)
	}

	if err := oldDB.Close(); err != nil {
		log.Errorf("Could not close old database: %v", err)
	}
	if err := db.Close(); err != nil {
		// Try remove in a best-effort fashion
		os.Remove(compactedDBFileName)
		return "", fmt.Errorf("could not close compacted database: %v", err)
	}

	return compactedDBFileName, nil
}

func backupBolt(db *bolt.DB, out io.Writer, scrubSecrets bool) error {
	tempFile, err := ioutil.TempFile("", "bolt-backup-")
	if err != nil {
		return fmt.Errorf("could not create temporary file for bolt backup: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	err = db.View(func(tx *bolt.Tx) error {
		_, err := tx.WriteTo(out)
		return err
	})
	if err != nil {
		return fmt.Errorf("could not dump bolt database: %v", err)
	}

	_, err = tempFile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("could not rewind to beginning of file: %v", err)
	}

	dbFileReader := io.ReadCloser(tempFile)
	if scrubSecrets {
		tempFileName := tempFile.Name()
		if err := tempFile.Close(); err != nil {
			return fmt.Errorf("could not close database dump file: %v", err)
		}

		compactedTempFilePath, err := scrubSecretsAndCompact(tempFileName)
		if err != nil {
			return fmt.Errorf("could not compact database: %v", err)
		}

		compactedTempFile, err := os.Open(compactedTempFilePath)
		if err != nil {
			return fmt.Errorf("could not open compacted database file: %v", err)
		}

		dbFileReader = compactedTempFile
	}

	defer dbFileReader.Close()

	_, err = io.Copy(out, dbFileReader)
	return err
}

func backupBadger(db *badger.DB, out io.Writer) error {
	tempFile, err := ioutil.TempFile("", "badger-backup-")
	if err != nil {
		return fmt.Errorf("could not create temporary file for badger backup: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = db.Backup(tempFile, 0)
	if err != nil {
		return fmt.Errorf("could not create badger backup: %v", err)
	}

	_, err = tempFile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("could not rewind to beginning of file: %v", err)
	}

	_, err = io.Copy(out, tempFile)
	return err
}

// Backup backs up the given databases (optionally removing secrets) and writes a ZIP archive to the given writer.
func Backup(boltDB *bolt.DB, badgerDB *badger.DB, out io.Writer, scrubSecrets bool) error {
	zipWriter := zip.NewWriter(out)
	boltWriter, err := zipWriter.Create(boltFileName)
	if err != nil {
		return err
	}
	if err := backupBolt(boltDB, boltWriter, scrubSecrets); err != nil {
		return fmt.Errorf("backing up bolt: %v", err)
	}
	badgerWriter, err := zipWriter.Create(badgerFileName)
	if err != nil {
		return err
	}
	if err := backupBadger(badgerDB, badgerWriter); err != nil {
		return fmt.Errorf("backing up badger: %v", err)
	}
	return zipWriter.Close()
}
