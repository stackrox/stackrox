package m211tom212

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_211_to_m_212_admission_control_config/schema"
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

// Perform cluster migration for admission controller config
func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	db := database.GormDB
	pgutils.CreateTableFromModel(ctx, db, schema.CreateTableClustersStmt)

	return fixAdmissionControllerConfig(ctx, db)
}

func fixAdmissionControllerConfig(ctx context.Context, database *gorm.DB) error {
	db := database.WithContext(ctx).Table(schema.ClustersTableName)
	var updatedClusters []*schema.Clusters
	var count int
	err := db.Transaction(func(tx *gorm.DB) error {
		rows, err := tx.Select("serialized").Rows()
		if err != nil {
			return errors.Wrapf(err, "failed to iterate table %s", schema.ClustersTableName)
		}
		for rows.Next() {
			var obj schema.Clusters
			if err = tx.ScanRows(rows, &obj); err != nil {
				return errors.Wrap(err, "failed to scan rows")
			}
			proto, err := ConvertClusterToProto(&obj)
			if err != nil {
				log.Errorf("failed to convert %+v to proto: %+v", obj, err)
				continue
			}

			// For clusters deployed using manifest install only - Helm and operator managed cluster config is taken care of
			// through the Helm values and operator manifest which are then communicated to Central after upgrade and after
			// secured clusters connect to Central once it is up and running post upgrade
			if proto.GetHelmConfig() == nil {
				if proto.GetDynamicConfig() != nil && proto.GetDynamicConfig().GetAdmissionControllerConfig() != nil {
					ac := proto.GetDynamicConfig().GetAdmissionControllerConfig()
					ac.ScanInline = true
					if ac.GetEnabled() || ac.GetEnforceOnUpdates() {
						ac.Enabled = true
						ac.EnforceOnUpdates = true
					}
				}

				converted, err := ConvertClusterFromProto(proto)
				if err != nil {
					return errors.Wrapf(err, "failed to convert from proto %+v", proto)
				}

				updatedClusters = append(updatedClusters, converted)
				count++
			}
		}
		if rows.Err() != nil {
			return errors.Wrapf(rows.Err(), "failed to get rows for %s", schema.ClustersTableName)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(updatedClusters) > 0 {
		if err = db.
			Clauses(clause.OnConflict{UpdateAll: true}).
			Model(schema.CreateTableClustersStmt.GormModel).
			CreateInBatches(&updatedClusters, batchSize).Error; err != nil {
			return errors.Wrap(err, "failed to upsert all converted objects")
		}
	}
	log.Infof("Fixed admission controller configuration for %d clusters", count)
	return nil
}

// ConvertClusterToProto converts Gorm model `Clusters` to its protobuf type object
func ConvertClusterToProto(m *schema.Clusters) (*storage.Cluster, error) {
	var msg storage.Cluster
	if err := msg.UnmarshalVTUnsafe(m.Serialized); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ConvertClusterFromProto converts a `*storage.Cluster` to Gorm model
func ConvertClusterFromProto(obj *storage.Cluster) (*schema.Clusters, error) {
	serialized, err := obj.MarshalVT()
	if err != nil {
		return nil, err
	}
	return &schema.Clusters{
		ID:                                obj.GetId(),
		Name:                              obj.GetName(),
		Type:                              obj.GetType(),
		Labels:                            obj.GetLabels(),
		StatusProviderMetadataClusterType: obj.GetStatus().GetProviderMetadata().GetCluster().GetType(),
		StatusOrchestratorMetadataVersion: obj.GetStatus().GetOrchestratorMetadata().GetVersion(),
		Serialized:                        serialized,
	}, nil
}
