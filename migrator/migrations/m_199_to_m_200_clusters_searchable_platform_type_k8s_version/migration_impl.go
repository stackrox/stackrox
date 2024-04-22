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
