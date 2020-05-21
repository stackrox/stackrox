package gatherers

import (
	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

type badgerGatherer struct {
	badger *badger.DB
}

func newBadgerGatherer(badger *badger.DB) *badgerGatherer {
	return &badgerGatherer{
		badger: badger,
	}
}

func getBadgerSize() (int64, error) {
	size, err := fileutils.DirectorySize(badgerhelper.DefaultBadgerPath)
	if err != nil {
		return 0, err
	}
	return size, nil
}

// Gather returns telemetry information about the Badger database used by this central
func (d *badgerGatherer) Gather() *data.DatabaseStats {
	errorList := errorhelpers.NewErrorList("")
	sizeInBytes, err := getBadgerSize()
	errorList.AddError(err)

	bucketStats, bucketErrors := d.getBadgerBucketStats()
	errorList.AddErrors(bucketErrors...)

	dbStats := &data.DatabaseStats{
		Type: "badger",
		// Can't get the path from the DB object, we don't track the actual path.  Just use the default for now.
		Path:      badgerhelper.DefaultBadgerPath,
		UsedBytes: sizeInBytes,
		Buckets:   bucketStats,
		Errors:    errorList.ErrorStrings(),
	}
	return dbStats
}

func (d *badgerGatherer) getBadgerBucketStats() ([]*data.BucketStats, []error) {
	var errList []error
	prefixCardinality, prefixBytes, err := badgerhelper.GetBadgerMetrics()
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
