package rockshelper

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/tecbot/gorocksdb"
)

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

// ReadFromRocksDB return unmarshalled proto object read from rocksDB for given prefix and id.
func ReadFromRocksDB(db *gorocksdb.DB, opts *gorocksdb.ReadOptions, msg proto.Message, prefix []byte, id []byte) (proto.Message, bool, error) {
	key := rocksdbmigration.GetPrefixedKey(prefix, id)
	slice, err := db.Get(opts, key)
	if err != nil {
		return nil, false, errors.Wrapf(err, "getting key %s", key)
	}
	defer slice.Free()
	if !slice.Exists() {
		return nil, false, nil
	}
	if err := proto.Unmarshal(slice.Data(), msg); err != nil {
		return nil, false, errors.Wrapf(err, "deserializing object with key %s", key)
	}
	return msg, true, nil
}
