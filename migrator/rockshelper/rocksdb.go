package rockshelper

import "github.com/tecbot/gorocksdb"

const (
	// RocksDBDirName it the name of the RocksDB directory on the PVC
	rocksDBDirName = `rocksdb`
	// RocksDBPath is the full directory path on the PVC
	rocksDBPath = "/var/lib/stackrox/" + rocksDBDirName
)

// New returns a new RocksDB
func New() (*gorocksdb.DB, error) {
	opts := gorocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCompression(gorocksdb.LZ4Compression)
	return gorocksdb.OpenDb(opts, rocksDBPath)
}
