package badgerhelper

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/migrations"
)

const (
	// BadgerDBDirName is the directory (under the mount point for data files) containing BadgerDB data.
	BadgerDBDirName = `badgerdb`
)

var (
	// DefaultBadgerPath is the default path for the DB. Exported for metrics
	DefaultBadgerPath = filepath.Join(migrations.DBMountPath, BadgerDBDirName)
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

// NewWithDefaults returns an instance of the persistent BadgerDB store instantiated at the default filesystem location.
func NewWithDefaults() (*badger.DB, error) {
	return New(DefaultBadgerPath)
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

// BucketKeyCount returns the number of objects in a "Bucket"
func BucketKeyCount(txn *badger.Txn, keyPrefix []byte) (int, error) {
	return Count(txn, dbhelper.GetBucketKey(keyPrefix, nil))
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

// CountWithBytes gets the number of keys with a specific prefix and the size in bytes of the specified prefix
// Count gets the number of keys with a specific prefix
func CountWithBytes(txn *badger.Txn, keyPrefix []byte) (int, int, error) {
	var count int
	var size int64
	opts := ForEachOptions{
		IteratorOptions: DefaultIteratorOptions(),
	}
	opts.IteratorOptions.PrefetchValues = false
	err := ForEachItemWithPrefix(txn, keyPrefix, opts, func(k []byte, item *badger.Item) error {
		count++
		size += item.KeySize() + item.ValueSize()
		return nil
	})
	return count, int(size), err
}
