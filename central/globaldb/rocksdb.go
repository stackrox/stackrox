package globaldb

import (
	"time"

	"github.com/stackrox/rox/central/globaldb/metrics"
	"github.com/stackrox/rox/central/option"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/rocksdb"
	rocksdbInstance "github.com/stackrox/rox/pkg/rocksdb/instance"
	rocksMetrics "github.com/stackrox/rox/pkg/rocksdb/metrics"
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
	postgres.LogCallerOnPostgres("GetRocksDB")
	rocksInit.Do(func() {
		rocksDB = rocksdbInstance.GetRocksDB()
		go startMonitoringRocksDB(rocksDB)
	})
	return rocksDB
}

func startMonitoringRocksDB(db *rocksdb.RocksDB) {
	ticker := time.NewTicker(gatherFrequency)
	for range ticker.C {
		rocksdbInstance.WalkBucket(
			func(prefix []byte, prefixString string, objType string) {
				rocksMetrics.UpdateRocksDBPrefixSizeMetric(GetRocksDB(), prefix, prefixString, objType)
			},
		)
		size, err := fileutils.DirectorySize(rocksMetrics.GetRocksDBPath(option.CentralOptions.DBPathBase))
		if err != nil {
			log.Errorf("error getting rocksdb directory size: %v", err)
			return
		}
		metrics.RocksDBSize.Set(float64(size))
	}
}
