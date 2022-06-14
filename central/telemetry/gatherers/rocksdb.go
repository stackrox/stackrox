package gatherers

import (
	"github.com/stackrox/rox/central/option"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/rocksdb/metrics"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

type rocksdbGatherer struct {
	db *rocksdb.RocksDB
}

func newRocksDBGatherer(db *rocksdb.RocksDB) *rocksdbGatherer {
	return &rocksdbGatherer{
		db: db,
	}
}

// Gather returns telemetry information about the RocksDB database used by this central
func (d *rocksdbGatherer) Gather() *data.DatabaseStats {
	errorList := errorhelpers.NewErrorList("rocksdb telemetry gather")
	sizeInBytes, err := getRocksDBSize()
	errorList.AddError(err)

	bucketStats, bucketErrors := d.getRocksDBBucketStats()
	errorList.AddErrors(bucketErrors...)

	dbStats := &data.DatabaseStats{
		Type: "rocksdb",
		// Can't get the path from the DB object, we don't track the actual path.  Just use the default for now.
		Path:      metrics.GetRocksDBPath(option.CentralOptions.DBPathBase),
		UsedBytes: sizeInBytes,
		Buckets:   bucketStats,
		Errors:    errorList.ErrorStrings(),
	}
	return dbStats
}

func (d *rocksdbGatherer) getRocksDBBucketStats() ([]*data.BucketStats, []error) {
	var errList []error
	prefixCardinality, prefixBytes, err := metrics.GetRocksDBMetrics()
	if err != nil {
		errList = append(errList, err)
	}
	if len(prefixCardinality) == 0 {
		return nil, nil
	}

	stats, errs := getBucketStats(prefixCardinality, prefixBytes)
	errList = append(errList, errs...)
	return stats, errList
}

// Get the number of bytes used by files stored for the db.
func getRocksDBSize() (int64, error) {
	size, err := fileutils.DirectorySize(metrics.GetRocksDBPath(option.CentralOptions.DBPathBase))
	if err != nil {
		return 0, err
	}
	return size, nil
}
