package gatherers

import (
	"github.com/stackrox/stackrox/pkg/errorhelpers"
	"github.com/stackrox/stackrox/pkg/fsutils"
	"github.com/stackrox/stackrox/pkg/migrations"
	"github.com/stackrox/stackrox/pkg/telemetry/data"
)

type databaseGatherer struct {
	bolt  *boltGatherer
	bleve *bleveGatherer
	rocks *rocksdbGatherer
}

func newDatabaseGatherer(rocks *rocksdbGatherer, bolt *boltGatherer, bleve *bleveGatherer) *databaseGatherer {
	return &databaseGatherer{
		bolt:  bolt,
		bleve: bleve,
		rocks: rocks,
	}
}

// Gather returns a list of stats about all the databases this Central is using
func (d *databaseGatherer) Gather() *data.StorageInfo {
	errList := errorhelpers.NewErrorList("")
	capacity, used, err := fsutils.DiskStatsIn(migrations.DBMountPath())
	errList.AddError(err)

	storageInfo := &data.StorageInfo{
		DiskCapacityBytes: int64(capacity),
		DiskUsedBytes:     int64(used),
		StorageType:       "unknown", // TODO: Figure out how to determine storage type (pvc etc.)
		Databases: []*data.DatabaseStats{
			d.bolt.Gather(),
		},
		Errors: errList.ErrorStrings(),
	}

	storageInfo.Databases = append(storageInfo.Databases, d.rocks.Gather())
	databaseStats := d.bleve.Gather()
	storageInfo.Databases = append(storageInfo.Databases, databaseStats...)

	return storageInfo
}
