package m98tom99

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/gorm/models"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
	"gorm.io/gorm"
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
	tx := postgresDB.Session(&gorm.Session{})
	if !tx.Migrator().HasTable(postgresTable) {
		if err := tx.Migrator().CreateTable(postgresTable); err != nil {
			return err
		}
	}

	postgresDB.AutoMigrate(&models.IntegrationHealth{})
	var ihs []*storage.Alert
	var conv []*models.Alert
	for it.Seek(rocksdbBucket); it.ValidForPrefix(rocksdbBucket); it.Next() {
		r := &storage.Alert{}
		if err := proto.Unmarshal(it.Value().Data(), r); err != nil {
			return errors.Wrapf(err, "Failed to unmarshal alert data for key %v", it.Key().Data())
		}
		ihs = append(ihs, r)

		conv = append(conv, &models.Alert{
			Id:         r.GetId(),
			Serialized: it.Value().Data(),

			PolicyId:                 r.GetPolicy().GetId(),
			PolicyName:               r.GetPolicy().GetName(),
			PolicyDescription:        r.GetPolicy().GetDescription(),
			PolicyDisabled:           r.GetPolicy().GetDisabled(),
			PolicyCategories:         r.GetPolicy().GetCategories(),
			PolicyLifecycleStages:    r.GetPolicy().GetLifecycleStages(),
			PolicySeverity:           r.GetPolicy().GetSeverity(),
			PolicyEnforcementActions: r.GetPolicy().GetEnforcementActions(),
			PolicyLastUpdated:        r.GetPolicy().GetLastUpdated(),
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
			Time:              r.GetTime(),
			State:             r.GetState(),
			Tags:              r.GetTags(),
		})
	}

	postgresDB.Table(models.IntegrationHealthTableName).Model(&models.IntegrationHealth{}).CreateInBatches(conv, 5000)
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
