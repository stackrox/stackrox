package handlers

import (
	"fmt"
	"io"
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
	dbFileFormat = "stackrox_2006_01_02_15_04_05.db"

	tempUploadName = "stackrox_temp.db"
)

// BackupDB is a handler that writes a consistent view of the database to the HTTP response.
func BackupDB(db *bolt.DB) http.Handler {
	return serializeDB(db, "")
}

// ExportDB is a handler that writes a consistent view of the database without secrets to the HTTP response.
func ExportDB(db *bolt.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		exportedFilepath := filepath.Join(filepath.Dir(db.Path()), "exported.db")
		compactedFilepath := filepath.Join(filepath.Dir(db.Path()), "compacted.db")

		db, err := export(db, exportedFilepath, compactedFilepath)
		if err != nil {
			handleError(w, err)
			return
		}

		serializeDB(db, compactedFilepath).ServeHTTP(w, req)
	})
}

func undoRestore(dst, src string) {
	if err := os.Rename(src, dst); err != nil {
		log.Errorf("Error undoing restore: %v", err)
	}
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
func RestoreDB(db *bolt.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		dbPath := db.Path()

		dir := filepath.Dir(dbPath)
		tempUploadPath := filepath.Join(dir, tempUploadName)

		// Write the new file and try and open it, returning an error if unsuccessful
		file, err := os.Create(tempUploadPath)
		if err != nil {
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "creating db %q: %v", dbPath, err)
			return
		}
		if _, err := io.Copy(file, req.Body); err != nil {
			os.Remove(tempUploadPath)
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "error copying added file: %v", err)
			return
		}

		if _, err := bolt.Open(tempUploadPath, 0600, nil); err != nil {
			os.Remove(tempUploadPath)
			logAndWriteErrorMsg(w, http.StatusBadRequest, "unable to open file to restore: %v", err)
			return
		}

		// Now that we have verified the uploaded DB, close the current DB
		// Do two renames and bounce Central
		defer deferredExit(1)
		if err := db.Close(); err != nil {
			log.Errorf("unable to close DB: %v", err)
			return
		}

		// Backup the DB to a new path
		replacementPath := filepath.Join(filepath.Dir(dbPath), "replaced_"+filepath.Base(dbPath))
		if err := os.Rename(dbPath, replacementPath); err != nil {
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "unable to rename %q: %v", dbPath, err)
			return
		}

		if err := os.Rename(tempUploadPath, dbPath); err != nil {
			undoRestore(dbPath, replacementPath)
			logAndWriteErrorMsg(w, http.StatusInternalServerError, "unable to rename uploaded file %q: %v", tempUploadPath, err)
			return
		}
		log.Infof("Bouncing Central to pick up newly imported DB")
		deferredExit(0)
	})
}

func serializeDB(db *bolt.DB, removalPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		filename := time.Now().Format(dbFileFormat)
		// This will block all other transactions until this has completed. We could use View for a hot backup
		err := db.Update(func(tx *bolt.Tx) error {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%v\"", filename))
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

func handleError(w http.ResponseWriter, err error) {
	log.Error(err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
