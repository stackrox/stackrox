package globaldb

import (
	"time"

	"github.com/stackrox/rox/central/globaldb/metrics"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/rocksdb"
	rocksMetrics "github.com/stackrox/rox/pkg/rocksdb/metrics"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	rocksInit sync.Once
	rocksDB   *rocksdb.RocksDB
)

// GetRocksDB returns the global rocksdb instance
func GetRocksDB() *rocksdb.RocksDB {
	if !env.RocksDB.BooleanSetting() {
		return nil
	}
	rocksInit.Do(func() {
		db, err := rocksdb.New(rocksMetrics.RocksDBPath)
		if err != nil {
			panic(err)
		}
		rocksDB = db
		go startMonitoringRocksDB(rocksDB)
	})
	return rocksDB
}

func startMonitoringRocksDB(db *rocksdb.RocksDB) {
	ticker := time.NewTicker(gatherFrequency)
	for range ticker.C {
		for _, bucket := range registeredBuckets {
			rocksMetrics.UpdateRocksDBPrefixSizeMetric(db, bucket.badgerPrefix, bucket.prefixString, bucket.objType)
		}

		size, err := fileutils.DirectorySize(rocksMetrics.RocksDBPath)
		if err != nil {
			log.Errorf("error getting rocksdb directory size: %v", err)
			return
		}
		metrics.RocksDBSize.Set(float64(size))
	}
}
