package gatherers

import (
	"fmt"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/telemetry/data"
	"golang.org/x/sys/unix"
)

type databaseGatherer struct {
	badger *badgerGatherer
	bolt   *boltGatherer
	bleve  *bleveGatherer
}

func newDatabaseGatherer(badger *badgerGatherer, bolt *boltGatherer, bleve *bleveGatherer) *databaseGatherer {
	return &databaseGatherer{
		badger: badger,
		bolt:   bolt,
		bleve:  bleve,
	}
}

// Gather returns a list of stats about all the databases this Central is using
func (d *databaseGatherer) Gather() *data.StorageInfo {
	errList := errorhelpers.NewErrorList("")
	capacity, used, err := getDiskStats(migrations.DBMountPath)
	errList.AddError(err)

	storageInfo := &data.StorageInfo{
		DiskCapacityBytes: capacity,
		DiskUsedBytes:     used,
		StorageType:       "unknown", // TODO: Figure out how to determine storage type (pvc etc.)
		Databases: []*data.DatabaseStats{
			d.badger.Gather(),
			d.bolt.Gather(),
			d.bleve.Gather(),
		},
		Errors: errList.ErrorStrings(),
	}
	return storageInfo
}

func getDiskStats(path string) (int64, int64, error) {
	var diskStats unix.Statfs_t
	err := unix.Statfs(path, &diskStats)
	retries := 0
	for err == unix.EINTR && retries < 3 {
		retries++
		err = unix.Statfs(path, &diskStats)
	}

	if err != nil {
		return 0, 0, fmt.Errorf("failed to get Central disk stats: %s", err.Error())
	}
	capacity := diskStats.Blocks * uint64(diskStats.Bsize)
	used := (diskStats.Blocks - diskStats.Bavail) * uint64(diskStats.Bsize)
	return int64(capacity), int64(used), nil
}
