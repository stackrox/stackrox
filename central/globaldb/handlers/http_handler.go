package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/stackrox/rox/central/globaldb/export"
	"github.com/stackrox/rox/central/systeminfo/listener"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	log = logging.LoggerForModule()
)

const (
	pgFileFormat = "postgres_db_2006_01_02_15_04_05.sql.zip"
)

// BackupDB is a handler that writes a consistent view of the databases to the HTTP response.
func BackupDB(postgresDB postgres.DB, backupListener listener.BackupListener, includeCerts bool) http.Handler {
	return dumpDB(postgresDB, backupListener, includeCerts)
}

func logAndWriteErrorMsg(w http.ResponseWriter, code int, t string, args ...interface{}) {
	errMsg := fmt.Sprintf(t, args...)
	log.Error(errMsg)
	http.Error(w, errMsg, code)
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
