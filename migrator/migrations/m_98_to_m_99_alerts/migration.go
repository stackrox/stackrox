package m98tom99

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/gorm/models"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/tecbot/gorocksdb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	migration = types.Migration{
		StartingSeqNum: 98,
		VersionAfter:   storage.Version{SeqNum: 99},
		Run: func(databases *types.Databases) error {
			if err := moveAlerts(databases.RocksDB, databases.PostgresDB); err != nil {
				return errors.Wrap(err,
					"moving alerts from rocksdb to postgres")
			}
			return nil
		},
	}
	rocksdbBucket = []byte("alerts")
	postgresTable = []byte("alerts")
)

func moveAlerts(rocksDB *gorocksdb.DB, postgresDB *gorm.DB) error {
	it := rocksDB.NewIterator(gorocksdb.NewDefaultReadOptions())
	defer it.Close()

	db := postgresDB.Table(models.AlertsTableName)
	if err := db.AutoMigrate(&models.Alert{}); err != nil {
		log.WriteToStderrf("failed to auto migrate alerts %v", err)
		return err
	}
	var conv []*models.Alert
	for it.Seek(rocksdbBucket); it.ValidForPrefix(rocksdbBucket); it.Next() {
		r := &storage.Alert{}
		if err := proto.Unmarshal(it.Value().Data(), r); err != nil {
			return errors.Wrapf(err, "Failed to unmarshal alert data for key %v", it.Key().Data())
		}
		conv = append(conv, &models.Alert{
			Id:         r.GetId(),
			Serialized: it.Value().Data(),

			PolicyId:                 r.GetPolicy().GetId(),
			PolicyName:               r.GetPolicy().GetName(),
			PolicyDescription:        r.GetPolicy().GetDescription(),
			PolicyDisabled:           r.GetPolicy().GetDisabled(),
			PolicyCategories:         pq.Array(r.GetPolicy().GetCategories()).(*pq.StringArray),
			PolicyLifecycleStages:    pq.Array(pgutils.ConvertEnumSliceToIntArray(r.GetPolicy().GetLifecycleStages())).(*pq.Int32Array),
			PolicySeverity:           r.GetPolicy().GetSeverity(),
			PolicyEnforcementActions: pq.Array(pgutils.ConvertEnumSliceToIntArray(r.GetPolicy().GetEnforcementActions())).(*pq.Int32Array),
			PolicyLastUpdated:        pgutils.NilOrTime(r.GetPolicy().GetLastUpdated()),
			PolicySORTName:           r.GetPolicy().GetSORTName(),
			PolicySORTLifecycleStage: r.GetPolicy().GetSORTLifecycleStage(),
			PolicySORTEnforcement:    r.GetPolicy().GetSORTEnforcement(),

			LifecycleStage: r.GetLifecycleStage(),
			ClusterId:      r.GetClusterId(),
			ClusterName:    r.GetClusterName(),
			Namespace:      r.GetNamespace(),
			NamespaceId:    r.GetNamespaceId(),

			DeploymentId:          r.GetDeployment().GetId(),
			DeploymentName:        r.GetDeployment().GetName(),
			DeploymentNamespace:   r.GetDeployment().GetNamespace(),
			DeploymentNamespaceId: r.GetDeployment().GetNamespaceId(),
			DeploymentClusterId:   r.GetDeployment().GetClusterId(),
			DeploymentClusterName: r.GetDeployment().GetClusterName(),
			DeploymentInactive:    r.GetDeployment().GetInactive(),

			ImageId:           r.GetImage().GetId(),
			ImageNameRegistry: r.GetImage().GetName().GetRegistry(),
			ImageNameRemote:   r.GetImage().GetName().GetRemote(),
			ImageNameTag:      r.GetImage().GetName().GetTag(),
			ImageNameFullName: r.GetImage().GetName().GetFullName(),

			ResourceResourceType: r.GetResource().GetResourceType(),
			ResourceName:         r.GetResource().GetName(),

			EnforcementAction: r.GetEnforcement().GetAction(),
			Time:              pgutils.NilOrTime(r.GetTime()),
			State:             r.GetState(),
			Tags:              pq.Array(r.GetTags()).(*pq.StringArray),
		})
	}

	log.WriteToStderr(fmt.Sprintf("converted %d alerts", len(conv)))
	tx := postgresDB.Table(models.AlertsTableName).Model(&models.Alert{}).Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(conv, 5000)
	if tx.Error != nil {
		tx.Rollback()
		return tx.Error
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
