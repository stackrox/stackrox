package badgerhelper

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
)

const (
	// BadgerDBDirName is the directory (under the mount point for data files) containing BadgerDB data.
	BadgerDBDirName = `badgerdb`
)

// GetDefaultOptions for BadgerDB
func GetDefaultOptions(path string) badger.Options {
	return badger.DefaultOptions(path).
		WithDir(path).
		WithTruncate(true).
		// These options keep the DB size small at the cost of doing more aggressive compaction
		// They are an adjustment on this issue and the related comments: https://github.com/dgraph-io/badger/issues/718
		WithNumLevelZeroTables(2).
		WithNumLevelZeroTablesStall(5)
}

// New returns an instance of the persistent BadgerDB store
func New(path string) (*badger.DB, error) {
	if stat, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0700)
		if err != nil {
			return nil, errors.Wrapf(err, "error creating badger path %s", path)
		}
	} else if err != nil {
		return nil, err
	} else if !stat.IsDir() {
		return nil, fmt.Errorf("badger path %s is not a directory", path)
	}

	options := GetDefaultOptions(path).WithLogger(nullLogger{})
	return badger.Open(options)
}

// NewTemp creates a new DB, but places it in the host temporary directory.
func NewTemp(name string) (*badger.DB, string, error) {
	tmpDir, err := ioutil.TempDir("", fmt.Sprintf("badgerdb-%s", strings.Replace(name, "/", "_", -1)))
	if err != nil {
		return nil, "", err
	}
	db, err := New(tmpDir)
	return db, tmpDir, err
}

// Count gets the number of keys with a specific prefix
func Count(txn *badger.Txn, keyPrefix []byte) (int, error) {
	var count int
	err := ForEachOverKeySet(txn, keyPrefix, ForEachOptions{}, func(_ []byte) error {
		count++
		return nil
	})
	return count, err
}
