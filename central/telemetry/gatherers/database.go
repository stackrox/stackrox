package gatherers

import (
	"context"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/fsutils"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

type databaseGatherer struct {
	bolt     *boltGatherer
	bleve    *bleveGatherer
	rocks    *rocksdbGatherer
	postgres *postgresGatherer
}

func newDatabaseGatherer(rocks *rocksdbGatherer, bolt *boltGatherer, bleve *bleveGatherer, postgres *postgresGatherer) *databaseGatherer {
	return &databaseGatherer{
		bolt:     bolt,
		bleve:    bleve,
		rocks:    rocks,
		postgres: postgres,
	}
}

// Gather returns a list of stats about all the databases this Central is using
func (d *databaseGatherer) Gather(ctx context.Context) *data.StorageInfo {
	errList := errorhelpers.NewErrorList("")
	capacity, used, err := fsutils.DiskStatsIn(migrations.DBMountPath())
	errList.AddError(err)

	storageInfo := &data.StorageInfo{
		DiskCapacityBytes: int64(capacity),
		DiskUsedBytes:     int64(used),
		StorageType:       "unknown", // TODO: Figure out how to determine storage type (pvc etc.)
		Databases:         []*data.DatabaseStats{},
		Errors:            errList.ErrorStrings(),
	}

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		storageInfo.Databases = append(storageInfo.Databases, d.postgres.Gather(ctx))
	} else {
		storageInfo.Databases = append(storageInfo.Databases, d.bolt.Gather())
		storageInfo.Databases = append(storageInfo.Databases, d.rocks.Gather())
		databaseStats := d.bleve.Gather()
		storageInfo.Databases = append(storageInfo.Databases, databaseStats...)
	}

	return storageInfo
}
