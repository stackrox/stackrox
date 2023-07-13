package m169tom170

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	permissionSetPostgresStore "github.com/stackrox/rox/migrator/migrations/m_169_to_m_170_collections_sac_resource_migration/permissionsetpostgresstore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
)

const (
	batchSize = 500

	workflowAdminResource       = "WorkflowAdministration"
	reportConfigurationResource = "ReportConfiguration"

	startSeqNum = 169
)

var (
	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)}, // 170
		Run: func(databases *types.Databases) error {
			err := migrateWorkflowAdministrationPermissionSet(databases.DBCtx, databases.PostgresDB)
			if err != nil {
				return errors.Wrapf(err, "updating %q permissions", workflowAdminResource)
			}
			return nil
		},
	}
)

func migrateWorkflowAdministrationPermissionSet(ctx context.Context, db postgres.DB) error {
	pgStore := permissionSetPostgresStore.New(db)
	permissionSetsToInsert := make([]*storage.PermissionSet, 0, batchSize)
	err := pgStore.Walk(ctx, func(obj *storage.PermissionSet) error {
		if accessLevel, found := obj.GetResourceToAccess()[reportConfigurationResource]; found {
			newPermissionSet := obj.Clone()
			newPermissionSet.ResourceToAccess[workflowAdminResource] = accessLevel
			permissionSetsToInsert = append(permissionSetsToInsert, newPermissionSet)
			if len(permissionSetsToInsert) >= batchSize {
				err := pgStore.UpsertMany(ctx, permissionSetsToInsert)
				if err != nil {
					return err
				}
				permissionSetsToInsert = permissionSetsToInsert[:0]
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(permissionSetsToInsert) > 0 {
		err = pgStore.UpsertMany(ctx, permissionSetsToInsert)
	}
	return err
}

func init() {
	migrations.MustRegisterMigration(migration)
}
