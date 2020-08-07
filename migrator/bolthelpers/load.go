package bolthelpers

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/migrations"
	bolt "go.etcd.io/bbolt"
)

const (
	dbFileName = "stackrox.db"
)

// Path returns the path to the Bolt DB
func Path() string {
	return filepath.Join(migrations.DBMountPath, dbFileName)
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
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, errors.Wrap(err, "bolt open failed")
	}

	return db, nil
}

// NewTemp creates a new DB, but places it in the host temporary directory.
func NewTemp(dbPath string) (*bolt.DB, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	return newBolt(filepath.Join(tmpDir, dbPath))
}
