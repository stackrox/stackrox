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
	if err := initializeTables(db); err != nil {
		return nil, err
	}
	return &BoltDB{
		db,
	}, nil
}

func initializeTables(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(alertBucket)); err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(benchmarkBucket)); err != nil {
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
