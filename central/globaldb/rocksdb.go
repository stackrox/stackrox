package globaldb

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/rocksdb"
	rocksdbInstance "github.com/stackrox/rox/pkg/rocksdb/instance"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	rocksInit sync.Once

	rocksDB *rocksdb.RocksDB

	log = logging.LoggerForModule()
)

// RegisterBucket registers a bucket to have metrics pulled from it
func RegisterBucket(bucketName []byte, objType string) {
	rocksdbInstance.RegisterBucket(bucketName, objType)
}

// GetRocksDB returns the global rocksdb instance
func GetRocksDB() *rocksdb.RocksDB {
	postgres.DeprecatedCall("GetRocksDB")

	rocksInit.Do(func() {
		rocksDB = rocksdbInstance.GetRocksDB()
	})
	return rocksDB
}
