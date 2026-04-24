package m223tom224

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	"github.com/stackrox/rox/migrator/migrations/m_223_to_m_224_set_deployment_state/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	log       = loghelper.LogWrapper{}
	batchSize = 500
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())

	// Add deleted and state columns if they do not already exist.
	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableDeploymentsStmt)

	// Set all existing deployments with STATE_UNSPECIFIED (0) to STATE_ACTIVE (1).
	// This updates both the state column and the serialized proto to keep them in sync.
	if err := setDeploymentsToActive(ctx, database.GormDB); err != nil {
		return err
	}

	return nil
}

func setDeploymentsToActive(ctx context.Context, database *gorm.DB) error {
	db := database.WithContext(ctx).Table(schema.DeploymentsTableName)
	var updatedDeployments []*schema.Deployments
	var count int

	err := db.Transaction(func(tx *gorm.DB) error {
		// Query deployments where state is NULL or STATE_UNSPECIFIED (0).
		rows, err := tx.Where("state IS NULL OR state = 0").Select("id", "serialized").Rows()
		if err != nil {
			return errors.Wrapf(err, "failed to query table %s", schema.DeploymentsTableName)
		}
		defer rows.Close()

		for rows.Next() {
			var obj schema.Deployments
			if err = tx.ScanRows(rows, &obj); err != nil {
				return errors.Wrap(err, "failed to scan rows")
			}

			proto, err := ConvertDeploymentToProto(&obj)
			if err != nil {
				log.WriteToStderrf("failed to convert deployment %s to proto: %v", obj.ID, err)
				continue
			}

			// Set state to STATE_ACTIVE.
			proto.State = storage.DeploymentState_STATE_ACTIVE

			converted, err := ConvertDeploymentFromProto(proto)
			if err != nil {
				return errors.Wrapf(err, "failed to convert deployment %s from proto", obj.ID)
			}

			updatedDeployments = append(updatedDeployments, converted)
			count++
		}

		if rows.Err() != nil {
			return errors.Wrapf(rows.Err(), "failed to get rows for %s", schema.DeploymentsTableName)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(updatedDeployments) > 0 {
		if err = db.
			Clauses(clause.OnConflict{UpdateAll: true}).
			Model(schema.CreateTableDeploymentsStmt.GormModel).
			CreateInBatches(&updatedDeployments, batchSize).Error; err != nil {
			return errors.Wrap(err, "failed to upsert all converted deployments")
		}
	}

	log.WriteToStderrf("Set state to STATE_ACTIVE for %d deployments", count)
	return nil
}

// ConvertDeploymentToProto converts Gorm model `Deployments` to its protobuf type object.
func ConvertDeploymentToProto(m *schema.Deployments) (*storage.Deployment, error) {
	var msg storage.Deployment
	if err := msg.UnmarshalVT(m.Serialized); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ConvertDeploymentFromProto converts a `*storage.Deployment` to Gorm model.
func ConvertDeploymentFromProto(obj *storage.Deployment) (*schema.Deployments, error) {
	serialized, err := obj.MarshalVT()
	if err != nil {
		return nil, err
	}
	return &schema.Deployments{
		ID:         obj.GetId(),
		State:      obj.GetState(),
		Serialized: serialized,
	}, nil
}
