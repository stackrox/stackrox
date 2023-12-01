package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/stackrox/rox/central/certs/export"
	"github.com/stackrox/rox/central/systeminfo/listener"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	certFileFormat = "cert_backup_2006_01_02_15_04_05.zip"
)

var (
	log = logging.LoggerForModule()
)

// BackupCerts is a handler that writes a consistent view of the certs to the HTTP response.
func BackupCerts(backupListener listener.BackupListener) http.Handler {
	return dumpCerts(backupListener)
}

func logAndWriteErrorMsg(w http.ResponseWriter, code int, t string, args ...interface{}) {
	errMsg := fmt.Sprintf(t, args...)
	log.Error(errMsg)
	http.Error(w, errMsg, code)
}

func dumpCerts(backupListener listener.BackupListener) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		log.Info("Starting cert backup ...")
		filename := time.Now().Format(certFileFormat)

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

		if err := export.BackupCerts(req.Context(), backupListener, w); err != nil {
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "could not create cert backup: %v", err)
			return
		}
		log.Info("Cert backup completed")
	}
}
