package m173tom174

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/m_173_to_m_174_group_unique_constraint/frozenschema"
	"github.com/stackrox/rox/migrator/migrations/m_173_to_m_174_group_unique_constraint/groupspostgresstore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"gorm.io/gorm"
)

const (
	startSeqNum = 172

	batchSize = 500
)

var (
	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)}, // 173
		Run: func(databases *types.Databases) error {
			if err := ensureUniqueGroups(databases.PostgresDB); err != nil {
				return errors.Wrap(err, "ensuring only unique groups are within table")
			}
			migrateSchema(databases.GormDB)
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func ensureUniqueGroups(postgresDB *postgres.DB) error {
	ctx := sac.WithAllAccess(context.Background())
	store := groupspostgresstore.New(postgresDB)

	uniqueGroups := map[string]struct{}{}
	groupsToDelete := make([]string, 0, batchSize)

	// Go through all groups, ensuring that the tuple of role name, auth provider ID, key, value is unique.
	// Delete all groups that are non-unique.
	err := store.Walk(ctx, func(group *storage.Group) error {
		groupUniqueKey := groupUniqueKey(group)
		if _, exists := uniqueGroups[groupUniqueKey]; !exists {
			uniqueGroups[groupUniqueKey] = struct{}{}
			return nil
		}
		groupsToDelete = append(groupsToDelete, group.GetProps().GetId())
		if len(groupsToDelete) >= batchSize {
			if err := store.DeleteMany(ctx, groupsToDelete); err != nil {
				return errors.Wrap(err, "deleting groups")
			}
			groupsToDelete = groupsToDelete[:0]
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(groupsToDelete) > 0 {
		return store.DeleteMany(ctx, groupsToDelete)
	}
	return nil
}

func migrateSchema(gormDB *gorm.DB) {
	pgutils.CreateTableFromModel(context.Background(), gormDB, frozenschema.CreateTableGroupsStmt)
}

func groupUniqueKey(group *storage.Group) string {
	return fmt.Sprintf("%s%s%s%s", group.GetRoleName(), group.GetProps().GetAuthProviderId(),
		group.GetProps().GetKey(), group.GetProps().GetValue())
}
