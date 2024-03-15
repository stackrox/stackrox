package rockshelper

import (
	"path/filepath"

	"github.com/DataDog/gorocksdb"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/option"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	// rocksDBDirName it the name of the RocksDB directory on the PVC
	rocksDBDirName = `rocksdb`
)

var (
	rocksInit sync.Once

	rocksDB *rocksdb.RocksDB
)

// GetRocksDB returns the global rocksdb instance
func GetRocksDB() *rocksdb.RocksDB {
	rocksInit.Do(func() {
		db, err := rocksdb.New(filepath.Join(option.MigratorOptions.DBPathBase, rocksDBDirName))
		if err != nil {
			panic(err)
		}
		rocksDB = db
	})
	return rocksDB
}

// ReadFromRocksDB return unmarshalled proto object read from rocksDB for given prefix and id.
func ReadFromRocksDB(db *gorocksdb.DB, opts *gorocksdb.ReadOptions, msg protocompat.Message, prefix []byte, id []byte) (protocompat.Message, bool, error) {
	key := rocksdbmigration.GetPrefixedKey(prefix, id)
	slice, err := db.Get(opts, key)
	if err != nil {
		return nil, false, errors.Wrapf(err, "getting key %s", key)
	}
	defer slice.Free()
	if !slice.Exists() {
		return nil, false, nil
	}
	if err := protocompat.Unmarshal(slice.Data(), msg); err != nil {
		return nil, false, errors.Wrapf(err, "deserializing object with key %s", key)
	}
	return msg, true, nil
}
