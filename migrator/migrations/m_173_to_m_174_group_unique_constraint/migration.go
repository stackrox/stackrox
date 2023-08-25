package m173tom174

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/m_173_to_m_174_group_unique_constraint/frozenschema"
	"github.com/stackrox/rox/migrator/migrations/m_173_to_m_174_group_unique_constraint/stores/previous"
	"github.com/stackrox/rox/migrator/migrations/m_173_to_m_174_group_unique_constraint/stores/updated"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"gorm.io/gorm"
)

const (
	startSeqNum = 173

	batchSize = 500
)

var (
	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)}, // 174
		Run: func(databases *types.Databases) error {
			if err := ensureUniqueGroups(databases.PostgresDB); err != nil {
				return errors.Wrap(err, "ensuring only unique groups are within table")
			}
			migrateSchema(databases.GormDB)
			if err := reUpsertGroupEntries(databases.PostgresDB); err != nil {
				return errors.Wrap(err, "re-upserting group entries after schema change")
			}
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func ensureUniqueGroups(postgresDB postgres.DB) error {
	ctx := sac.WithAllAccess(context.Background())
	previousStore := previous.New(postgresDB)

	uniqueGroups := map[string]struct{}{}
	groupsToDelete := make([]string, 0, batchSize)

	// Go through all groups, ensuring that the tuple of role name, auth provider ID, key, value is unique.
	// Delete all groups that are non-unique.
	err := previousStore.Walk(ctx, func(group *storage.Group) error {
		groupUniqueKey := groupUniqueKey(group)
		if _, exists := uniqueGroups[groupUniqueKey]; !exists {
			uniqueGroups[groupUniqueKey] = struct{}{}
			return nil
		}
		groupsToDelete = append(groupsToDelete, group.GetProps().GetId())
		if len(groupsToDelete) >= batchSize {
			if err := previousStore.DeleteMany(ctx, groupsToDelete); err != nil {
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
		if err := previousStore.DeleteMany(ctx, groupsToDelete); err != nil {
			return errors.Wrap(err, "deleting groups")
		}
	}
	return nil
}

func migrateSchema(gormDB *gorm.DB) {
	pgutils.CreateTableFromModel(context.Background(), gormDB, frozenschema.CreateTableGroupsStmt)
}

func reUpsertGroupEntries(postgresDB postgres.DB) error {
	ctx := sac.WithAllAccess(context.Background())
	updatedStore := updated.New(postgresDB)

	groupsToUpsert := make([]*storage.Group, 0, batchSize)
	// Go through all groups and re-upsert them, ensuring the new columns introduced with the schema are filled.
	err := updatedStore.Walk(ctx, func(group *storage.Group) error {
		groupsToUpsert = append(groupsToUpsert, group)

		if len(groupsToUpsert) >= batchSize {
			if err := updatedStore.UpsertMany(ctx, groupsToUpsert); err != nil {
				return errors.Wrap(err, "upserting groups")
			}
			groupsToUpsert = groupsToUpsert[:0]
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(groupsToUpsert) > 0 {
		if err := updatedStore.UpsertMany(ctx, groupsToUpsert); err != nil {
			return errors.Wrap(err, "upserting groups")
		}
	}
	return nil
}

func groupUniqueKey(group *storage.Group) string {
	return fmt.Sprintf("%s%s%s%s", group.GetRoleName(), group.GetProps().GetAuthProviderId(),
		group.GetProps().GetKey(), group.GetProps().GetValue())
}
