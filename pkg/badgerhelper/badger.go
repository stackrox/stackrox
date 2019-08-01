package badgerhelper

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/migrations"
)

const (
	// BadgerDBDirName is the directory (under the mount point for data files) containing BadgerDB data.
	BadgerDBDirName = `badgerdb`
)

var (
	separator = []byte("\x00")
)

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

	options := badger.DefaultOptions
	options.ValueDir = path
	options.Dir = path
	options.Logger = nullLogger{}
	options.Truncate = true
	return badger.Open(options)
}

// NewWithDefaults returns an instance of the persistent BadgerDB store instantiated at the default filesystem location.
func NewWithDefaults() (*badger.DB, error) {
	return New(filepath.Join(migrations.DBMountPath, BadgerDBDirName))
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

// DeletePrefixRange deletes all keys with a matching prefix from the DB.
func DeletePrefixRange(txn *badger.Txn, keyPrefix []byte) error {
	itOpts := badger.DefaultIteratorOptions
	itOpts.PrefetchValues = false
	itOpts.Prefix = keyPrefix

	it := txn.NewIterator(itOpts)
	defer it.Close()
	for it.Seek(keyPrefix); it.ValidForPrefix(keyPrefix); it.Next() {
		if err := txn.Delete(it.Item().Key()); err != nil {
			return err
		}
	}
	return nil
}

// Count gets the number of keys with a specific prefix
func Count(txn *badger.Txn, keyPrefix []byte) int {
	itOpts := badger.DefaultIteratorOptions
	itOpts.PrefetchValues = false
	itOpts.Prefix = keyPrefix

	var count int
	it := txn.NewIterator(itOpts)
	defer it.Close()
	for it.Seek(keyPrefix); it.ValidForPrefix(keyPrefix); it.Next() {
		count++
	}
	return count
}

// GetBucketKey returns a key which combines the prefix and the id with a separator
func GetBucketKey(prefix []byte, id []byte) []byte {
	result := make([]byte, 0, len(prefix)+len(separator)+len(id))
	result = append(result, prefix...)
	result = append(result, separator...)
	result = append(result, id...)
	return result
}
