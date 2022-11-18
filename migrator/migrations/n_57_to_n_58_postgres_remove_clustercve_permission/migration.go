package n57ton58

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/n_57_to_n_58_postgres_remove_clustercve_permission/postgres"
	"github.com/stackrox/rox/migrator/types"
)

const (
	batchSize = 500
)

var (
	migration = types.Migration{
		StartingSeqNum: 57,
		VersionAfter:   &storage.Version{SeqNum: 58},
		Run: func(databases *types.Databases) error {
			err := cleanupPermissionSets(databases.PostgresDB)
			if err != nil {
				return errors.Wrap(err, "updating PermissionSet schema")
			}
			return nil
		},
	}
	clusterCVEResourceName = "ClusterCVE"
)

func cleanupPermissionSets(db *pgxpool.Pool) error {
	ctx := context.Background()
	store := postgres.New(db)
	permissionSetsToInsert := make([]*storage.PermissionSet, 0, batchSize)
	err := store.Walk(ctx, func(obj *storage.PermissionSet) error {
		if _, ok := obj.GetResourceToAccess()[clusterCVEResourceName]; ok {
			delete(obj.ResourceToAccess, clusterCVEResourceName)
			permissionSetsToInsert = append(permissionSetsToInsert, obj)
			if len(permissionSetsToInsert) >= batchSize {
				err := store.UpsertMany(ctx, permissionSetsToInsert)
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
		err = store.UpsertMany(ctx, permissionSetsToInsert)
	}
	return err
}

func init() {
	migrations.MustRegisterMigration(migration)
}
