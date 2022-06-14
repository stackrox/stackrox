package gatherers

import (
	"os"

	"github.com/stackrox/stackrox/pkg/errorhelpers"
	"github.com/stackrox/stackrox/pkg/telemetry/data"
	"go.etcd.io/bbolt"
)

type boltGatherer struct {
	bolt *bbolt.DB
}

func newBoltGatherer(bolt *bbolt.DB) *boltGatherer {
	return &boltGatherer{
		bolt: bolt,
	}
}

// Gather returns telemetry information about the Bolt database used by this Central
func (b *boltGatherer) Gather() *data.DatabaseStats {
	errorList := errorhelpers.NewErrorList("")

	sizeInBytes, err := b.getDbSize()
	errorList.AddError(err)

	boltBuckets, err := b.getBoltBucketStats()
	errorList.AddError(err)

	dbStats := &data.DatabaseStats{
		Type:      "bolt",
		Path:      b.bolt.Path(),
		UsedBytes: sizeInBytes,
		Buckets:   boltBuckets,
		Errors:    errorList.ErrorStrings(),
	}
	return dbStats
}

func (b *boltGatherer) getDbSize() (int64, error) {
	dbFile, err := os.Open(b.bolt.Path())
	if err != nil {
		return 0, err
	}
	info, err := dbFile.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (b *boltGatherer) getBoltBucketStats() ([]*data.BucketStats, error) {
	var bucketStats []*data.BucketStats
	err := b.bolt.View(func(tx *bbolt.Tx) error {
		return tx.ForEach(func(bucket []byte, _ *bbolt.Bucket) error {
			stats := tx.Bucket(bucket).Stats()
			bucketStats = append(bucketStats, &data.BucketStats{
				Name:        string(bucket),
				UsedBytes:   int64(stats.BranchAlloc + stats.LeafAlloc),
				Cardinality: stats.KeyN,
			})
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return bucketStats, nil
}
