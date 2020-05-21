package rocksdb

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/tecbot/gorocksdb"
)

// NewTemp creates a new DB, but places it in the host temporary directory.
func NewTemp(name string) (*gorocksdb.DB, string, error) {
	tmpDir, err := ioutil.TempDir("", fmt.Sprintf("rocksdb-%s", strings.Replace(name, "/", "_", -1)))
	if err != nil {
		return nil, "", err
	}
	db, err := New(tmpDir)
	return db, tmpDir, err
}

// New creates a new RocksDB at the specified path
func New(path string) (*gorocksdb.DB, error) {
	return gorocksdb.OpenDb(GetRocksDBOptions(), path)
}

// GetRocksDBOptions returns the options used to open RocksDB.
func GetRocksDBOptions() *gorocksdb.Options {
	opts := gorocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCompression(gorocksdb.LZ4Compression)
	opts.SetUseFsync(true)
	return opts
}
