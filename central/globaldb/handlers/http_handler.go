package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

const (
	dbFileFormat = "stackrox_2006_01_02.db"
)

// BackupDB is a handler that writes a consistent view of the database to the HTTP response.
func BackupDB(db *bolt.DB) http.Handler {
	filename := time.Now().Format(dbFileFormat)
	return serializeDB(db, filename, "")
}

// ExportDB is a handler that writes a consistent view of the database without secrets to the HTTP response.
func ExportDB(db *bolt.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		filename, exportedFilepath, compactedFilepath := generateFilePaths()

		db, err := export(db, exportedFilepath, compactedFilepath)
		if err != nil {
			handleError(w, err)
			return
		}

		serializeDB(db, filename, compactedFilepath).ServeHTTP(w, req)
	})
}

func serializeDB(db *bolt.DB, file, removalPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// This will block all other transactions until this has completed. We could use View for a hot backup
		err := db.Update(func(tx *bolt.Tx) error {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%v\"", file))
			w.Header().Set("Content-Length", strconv.Itoa(int(tx.Size())))
			_, err := tx.WriteTo(w)
			return err
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		if removalPath != "" {
			db.Close()
			if err := os.Remove(removalPath); err != nil {
				log.Error(err)
			}
		}
	})
}

func generateFilePaths() (string, string, string) {
	filename := time.Now().Format(dbFileFormat)
	exportedFilepath := filepath.Join(os.TempDir(), "exported.db")
	compactedFilepath := filepath.Join(os.TempDir(), filename)
	return filename, exportedFilepath, compactedFilepath
}

func handleError(w http.ResponseWriter, err error) {
	log.Error(err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
