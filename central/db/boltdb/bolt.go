package boltdb

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"bitbucket.org/stack-rox/apollo/central/ranking"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/boltdb/bolt"
)

var (
	log = logging.LoggerForModule()
)

// BoltDB returns an instantiation of the storage interface. Exported for test purposes
type BoltDB struct {
	*bolt.DB
	ranker *ranking.Ranker
}

// New returns an instance of the persistent BoltDB store
func New(path string) (*BoltDB, error) {
	dirPath := filepath.Dir(path)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, 0600)
		if err != nil {
			return nil, fmt.Errorf("Error creating db path %v: %+v", dirPath, err)
		}
	} else if err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	b := &BoltDB{
		DB:     db,
		ranker: ranking.NewRanker(),
	}

	if err := b.initializeTables(); err != nil {
		log.Errorf("unable to initialize buckets: %s", err)
		b.Close()
		return nil, err
	}

	return b, nil
}

// NewWithDefaults returns an instance of the persistent BoltDB store with default values loaded.
func NewWithDefaults(dbPath string) (*BoltDB, error) {
	if filepath.Ext(dbPath) != ".db" {
		dbPath = filepath.Join(dbPath, "prevent.db")
	}

	db, err := New(dbPath)
	if err != nil {
		return db, err
	}

	return db, nil
}

func (b *BoltDB) initializeTables() error {
	var buckets = []string{
		alertBucket,
		authProviderBucket,
		benchmarksToScansBucket,
		benchmarkBucket,
		benchmarkScheduleBucket,
		benchmarkTriggerBucket,
		checkResultsBucket,
		clusterBucket,
		clusterStatusBucket,
		deploymentBucket,
		deploymentGraveyard,
		deploymentEventBucket,
		dnrIntegrationBucket,
		imageIntegrationBucket,
		imageBucket,
		logsBucket,
		multiplierBucket,
		notifierBucket,
		policyBucket,
		scanMetadataBucket,
		scansToCheckBucket,
		serviceIdentityBucket,
	}
	return b.Update(func(tx *bolt.Tx) error {
		for _, b := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(b)); err != nil {
				return fmt.Errorf("create bucket: %s", err)
			}
			if err := createUniqueKeyBucket(tx, b); err != nil {
				return fmt.Errorf("create bucket: %s", err)
			}
		}
		return nil
	})
}

// Close closes the database
func (b *BoltDB) Close() {
	if err := b.DB.Close(); err != nil {
		log.Errorf("unable to close bolt db: %s", err)
	}
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

// BackupHandler writes a consistent view of the database to the HTTP response
func (b *BoltDB) BackupHandler() http.Handler {
	filename := time.Now().Format("prevent_2006_01_02.db")
	return serializeDB(b.DB, filename, "")
}

func handleError(w http.ResponseWriter, err error) {
	log.Error(err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// ExportHandler writes a consistent view of the database without secrets to the HTTP response
func (b *BoltDB) ExportHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		exportedFilepath := filepath.Join(os.TempDir(), "exported.db")
		defer os.Remove(exportedFilepath)

		// This will block all other transactions until this has completed. We could use View for a hot backup
		err := b.Update(func(tx *bolt.Tx) error {
			return tx.CopyFile(exportedFilepath, 0600)
		})
		if err != nil {
			handleError(w, err)
			return
		}

		exportDB, err := New(exportedFilepath)
		if err != nil {
			handleError(w, err)
			return
		}
		defer exportDB.Close()

		notifiers, err := exportDB.GetNotifiers(&v1.GetNotifiersRequest{})
		if err != nil {
			handleError(w, err)
			return
		}
		for _, n := range notifiers {
			n.Config = nil
			if err := exportDB.UpdateNotifier(n); err != nil {
				handleError(w, err)
				return
			}
		}
		integrations, err := exportDB.GetImageIntegrations(&v1.GetImageIntegrationsRequest{})
		if err != nil {
			handleError(w, err)
			return
		}
		for _, d := range integrations {
			d.Config = nil
			if err := exportDB.UpdateImageIntegration(d); err != nil {
				handleError(w, err)
				return
			}
		}
		if err := exportDB.Sync(); err != nil {
			handleError(w, err)
			return
		}

		filename := time.Now().Format("prevent_2006_01_02.db")
		compactedFilepath := filepath.Join(os.TempDir(), filename)

		// Create completely clean DB and compact to it, wiping the secrets from cached memory
		newDB, err := bolt.Open(compactedFilepath, 0600, nil)
		if err != nil {
			handleError(w, err)
			return
		}
		if err := bolthelper.Compact(newDB, exportDB.DB); err != nil {
			handleError(w, err)
			return
		}
		// Close the databases
		exportDB.Close()
		serializeDB(newDB, filename, compactedFilepath).ServeHTTP(w, req)
	})
}
