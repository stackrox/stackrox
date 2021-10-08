package globaldb

import (
	"time"

	"github.com/stackrox/rox/central/globaldb/metrics"
	"github.com/stackrox/rox/central/option"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/rocksdb"
	rocksMetrics "github.com/stackrox/rox/pkg/rocksdb/metrics"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	rocksInit sync.Once
	rocksDB   *rocksdb.RocksDB

	registeredBuckets []registeredBucket

	registeredTables []registeredTable

	log = logging.LoggerForModule()
)

type registeredTable struct {
	table, objType string
}

type registeredBucket struct {
	prefix       []byte
	prefixString string
	objType      string
}

// RegisterBucket registers a bucket to have metrics pulled from it
func RegisterBucket(bucketName []byte, objType string) {
	registeredBuckets = append(registeredBuckets, registeredBucket{
		prefixString: string(bucketName),
		prefix:       bucketName,
		objType:      objType,
	})
}

func RegisterTable(table string, objType string) {
	registeredTables = append(registeredTables, registeredTable{
		table: table,
		objType: objType,
	})
}

// GetRocksDB returns the global rocksdb instance
func GetRocksDB() *rocksdb.RocksDB {
	rocksInit.Do(func() {
		db, err := rocksdb.New(rocksMetrics.GetRocksDBPath(option.CentralOptions.DBPathBase))
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
			rocksMetrics.UpdateRocksDBPrefixSizeMetric(db, bucket.prefix, bucket.prefixString, bucket.objType)
		}

		size, err := fileutils.DirectorySize(rocksMetrics.GetRocksDBPath(option.CentralOptions.DBPathBase))
		if err != nil {
			log.Errorf("error getting rocksdb directory size: %v", err)
			return
		}
		metrics.RocksDBSize.Set(float64(size))
	}
}
