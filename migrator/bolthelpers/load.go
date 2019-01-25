package bolthelpers

import (
	"fmt"
	"os"
	"path/filepath"

	bolt "github.com/etcd-io/bbolt"
)

const (
	dbMountPath = "/var/lib/stackrox"
	dbFileName  = "stackrox.db"
)

// New returns an instance of the persistent BoltDB store
func New() (*bolt.DB, error) {
	dbPath := filepath.Join(dbMountPath, dbFileName)
	_, err := os.Stat(dbPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("couldn't stat file: %v", err)
	}
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("bolt open failed: %v", err)
	}

	return db, nil
}
