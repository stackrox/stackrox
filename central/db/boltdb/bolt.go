package boltdb

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/boltdb/bolt"
)

var (
	log = logging.New("db/bolt")
)

// var so this can be modified in tests
var (
	defaultBenchmarksPath = `/data/benchmarks`
)

// BoltDB returns an instantiation of the storage interface. Exported for test purposes
type BoltDB struct {
	*bolt.DB
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
		DB: db,
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
		dbPath = filepath.Join(dbPath, "mitigate.db")
	}

	db, err := New(dbPath)
	if err != nil {
		return db, err
	}

	if err := db.loadDefaults(); err != nil {
		log.Errorf("unable to load defaults: %s", err)
		db.Close()
		return nil, err
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
		imageBucket,
		policyBucket,
		notifierBucket,
		registryBucket,
		scannerBucket,
		scanMetadataBucket,
		scansToCheckBucket,
		serviceIdentityBucket,
	}
	return b.Update(func(tx *bolt.Tx) error {
		for _, b := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(b)); err != nil {
				return fmt.Errorf("create bucket: %s", err)
			}
		}
		return nil
	})
}

// Close closes the database
func (b *BoltDB) Close() {
	b.DB.Close()
}

// BackupHandler writes a consistent view of the database to the HTTP response
func (b *BoltDB) BackupHandler() http.Handler {
	filename := time.Now().Format("mitigate_2006_01_02.db")
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// This will block all other transactions until this has completed. We could use View for a hot backup
		err := b.Update(func(tx *bolt.Tx) error {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%v\"", filename))
			w.Header().Set("Content-Length", strconv.Itoa(int(tx.Size())))
			_, err := tx.WriteTo(w)
			return err
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
