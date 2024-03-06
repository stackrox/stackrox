package gatherers

import (
	"context"

	"github.com/stackrox/rox/pkg/fsutils"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

type databaseGatherer struct {
	postgres *postgresGatherer
}

func newDatabaseGatherer(postgres *postgresGatherer) *databaseGatherer {
	return &databaseGatherer{
		postgres: postgres,
	}
}

// Gather returns a list of stats about all the databases this Central is using
func (d *databaseGatherer) Gather(ctx context.Context) *data.StorageInfo {
	capacity, used, err := fsutils.DiskStatsIn(migrations.DBMountPath())
	var errStrings []string
	if err != nil {
		errStrings = []string{err.Error()}
	}

	storageInfo := &data.StorageInfo{
		DiskCapacityBytes: int64(capacity),
		DiskUsedBytes:     int64(used),
		StorageType:       "unknown", // TODO: Figure out how to determine storage type (pvc etc.)
		Databases:         []*data.DatabaseStats{},
		Errors:            errStrings,
	}

	storageInfo.Databases = append(storageInfo.Databases, d.postgres.Gather(ctx))
	return storageInfo
}
