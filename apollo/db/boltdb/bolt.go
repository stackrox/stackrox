package boltdb

import (
	"fmt"
	"path/filepath"

	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/boltdb/bolt"
)

var (
	log = logging.New("db/bolt")
)

// var so this can be modified in tests
var defaultPoliciesPath = `/data/policies`

// BoltDB returns an instantiation of the storage interface. Exported for test purposes
type BoltDB struct {
	*bolt.DB
}

// New returns an instance of the persistent BoltDB store
func New(dbPath string) (*BoltDB, error) {
	db, err := bolt.Open(filepath.Join(dbPath, "apollo.db"), 0600, nil)
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
	return b.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(alertBucket)); err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(benchmarkBucket)); err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(benchmarkResultBucket)); err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(clusterBucket)); err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(deploymentBucket)); err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(imageBucket)); err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(imagePolicyBucket)); err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(notifierBucket)); err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(registryBucket)); err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(scannerBucket)); err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
}

// Close closes the database
func (b *BoltDB) Close() {
	b.DB.Close()
}

// Load is empty because Load is only used by non persistent DBs
func (b *BoltDB) Load() error {
	return nil
}
