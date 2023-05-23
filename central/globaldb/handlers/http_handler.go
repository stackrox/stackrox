package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/stackrox/rox/central/globaldb/export"
	"github.com/stackrox/rox/central/systeminfo/listener"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/osutils"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/rocksdb"
	bolt "go.etcd.io/bbolt"
)

var (
	log = logging.LoggerForModule()
)

const (
	dbFileFormat      = "stackrox_db_2006_01_02_15_04_05.zip"
	restoreFileFormat = "dbrestore_2006_01_02_15_04_05"
	pgFileFormat      = "postgres_db_2006_01_02_15_04_05.sql.zip"
)

// BackupDB is a handler that writes a consistent view of the databases to the HTTP response.
func BackupDB(boltDB *bolt.DB, rocksDB *rocksdb.RocksDB, postgresDB postgres.DB, backupListener listener.BackupListener, includeCerts bool) http.Handler {
	return dumpDB(postgresDB, backupListener, includeCerts)
	return serializeDB(rocksDB, boltDB, backupListener, includeCerts)
}

func logAndWriteErrorMsg(w http.ResponseWriter, code int, t string, args ...interface{}) {
	errMsg := fmt.Sprintf(t, args...)
	log.Error(errMsg)
	http.Error(w, errMsg, code)
}

// This will EOF if we call exit at the end of a handler
func deferredRestart(ctx context.Context) {
	go func() {
		concurrency.WaitWithTimeout(ctx, 5*time.Second)
		time.Sleep(1 * time.Second)
		osutils.Restart()
	}()
}

// RestoreDB is a handler that takes in a DB and restores Central to it
func RestoreDB(boltDB *bolt.DB, rocksDB *rocksdb.RocksDB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Info("Starting DB restore ...")
		// This is the old v1 API.  No need to support that for Postgres
		logAndWriteErrorMsg(w, http.StatusInternalServerError, "api is deprecated and does not support Postgres.")
	})
}

func serializeDB(rocksDB *rocksdb.RocksDB, boltDB *bolt.DB, backupListener listener.BackupListener, includeCerts bool) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		log.Info("Starting DB backup ...")
		filename := time.Now().Format(dbFileFormat)

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

		if err := export.Backup(req.Context(), boltDB, rocksDB, backupListener, includeCerts, w); err != nil {
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "could not create database backup: %v", err)
			return
		}
		log.Info("DB backup completed")
	}
}

func dumpDB(postgresDB postgres.DB, backupListener listener.BackupListener, includeCerts bool) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		log.Info("Starting Postgres DB backup ...")
		filename := time.Now().Format(pgFileFormat)

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

		if err := export.BackupPostgres(req.Context(), postgresDB, backupListener, includeCerts, w); err != nil {
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "could not create database backup: %v", err)
			return
		}
		log.Info("Postgres DB backup completed")
	}
}
