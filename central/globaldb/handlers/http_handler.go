package handlers

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/globaldb/export"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

const (
	dbFileFormat = "stackrox_db_2006_01_02_15_04_05.zip"
)

// BackupDB is a handler that writes a consistent view of the databases to the HTTP response.
func BackupDB(boltDB *bolt.DB, badgerDB *badger.DB) http.Handler {
	return serializeDB(boltDB, badgerDB, false)
}

// ExportDB is a handler that writes a consistent view of the databases without secrets to the HTTP response.
func ExportDB(boltDB *bolt.DB, badgerDB *badger.DB) http.Handler {
	return serializeDB(boltDB, badgerDB, true)
}

func logAndWriteErrorMsg(w http.ResponseWriter, code int, t string, args ...interface{}) {
	errMsg := fmt.Sprintf(t, args...)
	log.Error(errMsg)
	http.Error(w, errMsg, code)
}

// This will EOF if we call exit at the end of a handler
func deferredExit(code int) {
	go func() {
		time.Sleep(50 * time.Millisecond)
		os.Exit(code)
	}()
}

// RestoreDB is a handler that takes in a DB and restores Central to it
func RestoreDB(boltDB *bolt.DB, badgerDB *badger.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		tempFile, err := ioutil.TempFile("", "dbrestore-")
		if err != nil {
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "could not create temporary file for DB upload: %v", err)
			return
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		if _, err := io.Copy(tempFile, req.Body); err != nil {
			req.Body.Close()
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "error storing upload in temporary location: %v", err)
			return
		}
		req.Body.Close()

		if _, err := tempFile.Seek(0, 0); err != nil {
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "could not rewind to beginning of temporary file: %v", err)
			return
		}

		if err := export.Restore(tempFile); err != nil {
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "could not restore database backup: %v", err)
			return
		}

		// Now that we have verified the uploaded DB, close the current DB
		// Do two renames and bounce Central
		defer deferredExit(1)

		if err := boltDB.Close(); err != nil {
			log.Errorf("unable to close bolt DB: %v", err)
		}
		if err := badgerDB.Close(); err != nil {
			log.Errorf("unable to close badger DB: %v", err)
		}

		log.Infof("Bouncing Central to pick up newly imported DB")
		deferredExit(0)
	})
}

func serializeDB(boltDB *bolt.DB, badgerDB *badger.DB, scrubSecrets bool) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		filename := time.Now().Format(dbFileFormat)

		tempFile, err := ioutil.TempFile("", "db-backup-")
		if err != nil {
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "could not create temporary ZIP file for database export: %v", err)
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		if err := export.Backup(boltDB, badgerDB, tempFile, scrubSecrets); err != nil {
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "could not create database backup: %v", err)
			return
		}

		size, err := tempFile.Seek(0, 1)
		if err != nil {
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "could not determine size of database backup: %v", err)
			return
		}

		_, err = tempFile.Seek(0, 0)
		if err != nil {
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "could not rewind to beginning of backup file: %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		w.Header().Set("Content-Length", strconv.Itoa(int(size)))

		if _, err := io.Copy(w, tempFile); err != nil {
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "could not copy to output stream: %v", err)
			return
		}
	}
}
