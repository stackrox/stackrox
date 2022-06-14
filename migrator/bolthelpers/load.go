package bolthelpers

import (
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/option"
	bolt "go.etcd.io/bbolt"
)

const (
	dbFileName    = "stackrox.db"
	dbOpenTimeout = 2 * time.Minute
)

// Path returns the path to the Bolt DB
func Path() string {
	return filepath.Join(option.MigratorOptions.DBPathBase, dbFileName)
}

// Load loads an instance of Bolt from disk.
// It returns nil and no error if no DB is found in the expected location; the caller MUST check for this.
func Load() (*bolt.DB, error) {
	dbPath := Path()
	_, err := os.Stat(dbPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "couldn't stat file")
	}
	return newBolt(dbPath)
}

func newBolt(dbPath string) (*bolt.DB, error) {
	opts := *bolt.DefaultOptions
	opts.Timeout = dbOpenTimeout
	db, err := bolt.Open(dbPath, 0600, &opts)
	if err != nil {
		return nil, errors.Wrap(err, "bolt open failed")
	}

	return db, nil
}

// NewTemp creates a new DB, but places it in the host temporary directory.
func NewTemp(dbPath string) (*bolt.DB, error) {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, err
	}
	return newBolt(filepath.Join(tmpDir, dbPath))
}
