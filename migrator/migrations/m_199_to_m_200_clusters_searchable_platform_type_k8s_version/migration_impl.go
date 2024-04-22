package m199tom200

import (
	"context"

	"github.com/pkg/errors"
	newSchema "github.com/stackrox/rox/migrator/migrations/m_199_to_m_200_clusters_searchable_platform_type_k8s_version/schema/new"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TODO(dont-merge): generate/write and import any store required for the migration (skip any unnecessary step):
//  - create a schema subdirectory
//  - create a schema/old subdirectory
//  - create a schema/new subdirectory
//  - create a stores subdirectory
//  - create a stores/previous subdirectory
//  - create a stores/updated subdirectory
//  - copy the old schemas from pkg/postgres/schema to schema/old
//  - copy the old stores from their location in central to appropriate subdirectories in stores/previous
//  - generate the new schemas in pkg/postgres/schema and the new stores where they belong
//  - copy the newly generated schemas from pkg/postgres/schema to schema/new
//  - remove the calls to GetSchemaForTable and to RegisterTable from the copied schema files
//  - remove the xxxTableName constant from the copied schema files
//  - copy the newly generated stores from their location in central to appropriate subdirectories in stores/updated
//  - remove any unused function from the copied store files (the minimum for the public API should contain Walk, UpsertMany, DeleteMany)
//  - remove the scoped access control code from the copied store files
//  - remove the metrics collection code from the copied store files

// TODO(dont-merge): Determine if this change breaks a previous releases database.
// If so increment the `MinimumSupportedDBVersionSeqNum` to the `CurrentDBVersionSeqNum` of the release immediately
// following the release that cannot tolerate the change in pkg/migrations/internal/fallback_seq_num.go.
//
// For example, in 4.2 a column `column_v2` is added to replace the `column_v1` column in 4.1.
// All the code from 4.2 onward will not reference `column_v1`. At some point in the future a rollback to 4.1
// will not longer be supported and we want to remove `column_v1`. To do so, we will upgrade the schema to remove
// the column and update the `MinimumSupportedDBVersionSeqNum` to be the value of `CurrentDBVersionSeqNum` in 4.2
// as 4.1 will no longer be supported. The migration process will inform the user of an error when trying to migrate
// to a software version that can no longer be supported by the database.

var (
	batchSize = 2000
	log       = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	db := database.GormDB
	pgutils.CreateTableFromModel(ctx, db, newSchema.CreateTableClustersStmt)

	return populatePlatformTypeAndK8sVersionColumns(ctx, db)
}

func populatePlatformTypeAndK8sVersionColumns(ctx context.Context, database *gorm.DB) error {
	db := database.WithContext(ctx).Table(newSchema.ClustersTableName)
	rows, err := db.Rows()
	if err != nil {
		return errors.Wrapf(err, "failed to iterate table %s", newSchema.ClustersTableName)
	}

	var updatedClusters []*newSchema.Clusters
	var count int

	for rows.Next() {
		var obj newSchema.Clusters
		if err = db.ScanRows(rows, &obj); err != nil {
			return errors.Wrap(err, "failed to scan rows")
		}
		proto, err := newSchema.ConvertClusterToProto(&obj)
		if err != nil {
			return errors.Wrapf(err, "failed to convert %+v to proto", obj)
		}

		converted, err := newSchema.ConvertClusterFromProto(proto)
		if err != nil {
			return errors.Wrapf(err, "failed to convert from proto %+v", proto)
		}

		updatedClusters = append(updatedClusters, converted)
		count++
		if len(updatedClusters) == batchSize {
			if err = db.
				Clauses(clause.OnConflict{UpdateAll: true}).
				Model(newSchema.CreateTableClustersStmt.GormModel).
				Create(&updatedClusters).Error; err != nil {
				return errors.Wrapf(err, "failed to upsert converted %d objects after %d upserted", len(updatedClusters), count-len(updatedClusters))
			}
			updatedClusters = updatedClusters[:0]
		}
	}
	if rows.Err() != nil {
		return errors.Wrapf(rows.Err(), "failed to get rows for %s", newSchema.ClustersTableName)
	}
	if len(updatedClusters) > 0 {
		if err = db.
			Clauses(clause.OnConflict{UpdateAll: true}).
			Model(newSchema.CreateTableClustersStmt.GormModel).
			Create(&updatedClusters).Error; err != nil {
			return errors.Wrapf(err, "failed to upsert converted %d objects after %d upserted", len(updatedClusters), count-len(updatedClusters))
		}
	}
	log.Infof("Populated cluster type and k8s version columns for %d clusters", count)
	return nil
}

// TODO(dont-merge): remove any pending TODO
