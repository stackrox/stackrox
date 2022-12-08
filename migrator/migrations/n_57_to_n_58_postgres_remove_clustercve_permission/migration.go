package n57ton58

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	permissionsetpostgresstore "github.com/stackrox/rox/migrator/migrations/n_57_to_n_58_postgres_remove_clustercve_permission/permissionsetpostgresstore"
	"github.com/stackrox/rox/migrator/types"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
)

const (
	batchSize = 500
)

var (
	migration = types.Migration{
		StartingSeqNum: pkgMigrations.CurrentDBVersionSeqNumWithoutPostgres() + 57,
		VersionAfter:   &storage.Version{SeqNum: int32(pkgMigrations.CurrentDBVersionSeqNumWithoutPostgres()) + 58},
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
	permissionSetStore := permissionsetpostgresstore.New(db)
	permissionSetsToInsert := make([]*storage.PermissionSet, 0, batchSize)
	err := permissionSetStore.Walk(ctx, func(obj *storage.PermissionSet) error {
		if _, ok := obj.GetResourceToAccess()[clusterCVEResourceName]; ok {
			delete(obj.ResourceToAccess, clusterCVEResourceName)
			permissionSetsToInsert = append(permissionSetsToInsert, obj)
			if len(permissionSetsToInsert) >= batchSize {
				err := permissionSetStore.UpsertMany(ctx, permissionSetsToInsert)
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
		err = permissionSetStore.UpsertMany(ctx, permissionSetsToInsert)
	}
	return err
}

func init() {
	migrations.MustRegisterMigration(migration)
}
