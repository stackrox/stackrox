package globaldb

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// NewBoltDB returns an instance of the persistent BoltDB store
func NewBoltDB(path string) (*bolt.DB, error) {
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

	return db, nil
}

// New returns an instance of the persistent BoltDB store
func New(path string) (*bolt.DB, error) {
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

	return db, nil
}

// NewWithDefaults returns an instance of the persistent BoltDB store with default values loaded.
func NewWithDefaults(dbPath string) (*bolt.DB, error) {
	if filepath.Ext(dbPath) != ".db" {
		dbPath = filepath.Join(dbPath, "prevent.db")
	}

	db, err := New(dbPath)
	if err != nil {
		return db, err
	}

	return db, nil
}
