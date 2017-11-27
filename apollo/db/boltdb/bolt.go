package boltdb

import (
	"fmt"
	"path/filepath"

	"bitbucket.org/stack-rox/apollo/apollo/db"
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

// MakeBoltDB returns an instance of the persistent BoltDB store
func MakeBoltDB(dbPath string) (db.Storage, error) {
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
	var err error
	db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte(imageBucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte(imageRuleBucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte(alertBucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	return err
}

// Close closes the database
func (b *BoltDB) Close() {
	b.DB.Close()
}

// Load is empty because Load is only used by non persistent DBs
func (b *BoltDB) Load() error {
	return nil
}
