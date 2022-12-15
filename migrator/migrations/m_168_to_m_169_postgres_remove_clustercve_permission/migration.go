package m168tom169

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	permissionSetPostgresStore "github.com/stackrox/rox/migrator/migrations/m_168_to_m_169_postgres_remove_clustercve_permission/permissionsetpostgresstore"
	"github.com/stackrox/rox/migrator/types"
)

const (
	batchSize = 500

	startingSeqNum = 168
)

var (
	migration = types.Migration{
		StartingSeqNum: startingSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startingSeqNum + 1)}, // 169
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

// Replacement resources
const (
	Cluster = "Cluster"
)

// Replaced resources
const (
	ClusterCVE = "ClusterCVE"
)

var (
	replacements = map[string]string{
		ClusterCVE: Cluster,
	}
)

func propagateAccessForPermission(permission string, accessLevel storage.Access, permissionSet map[string]storage.Access) storage.Access {
	oldLevel, found := permissionSet[permission]
	if !found {
		return accessLevel
	}
	if accessLevel > oldLevel {
		return oldLevel
	}
	return accessLevel
}

func cleanupPermissionSets(db *pgxpool.Pool) error {
	ctx := context.Background()
	permissionSetStore := permissionSetPostgresStore.New(db)
	permissionSetsToInsert := make([]*storage.PermissionSet, 0, batchSize)
	err := permissionSetStore.Walk(ctx, func(obj *storage.PermissionSet) error {
		if _, ok := obj.GetResourceToAccess()[clusterCVEResourceName]; ok {
			// Copy the permission set, removing the deprecated resource permissions, and keeping the
			// lowest access level between that of deprecated resource and their replacement
			// for the replacement resource.
			newPermissionSet := obj.Clone()
			newPermissionSet.ResourceToAccess = make(map[string]storage.Access, len(obj.GetResourceToAccess()))
			for resource, accessLevel := range obj.GetResourceToAccess() {
				if _, found := replacements[resource]; found {
					resource = replacements[resource]
				}
				newPermissionSet.ResourceToAccess[resource] =
					propagateAccessForPermission(resource, accessLevel, newPermissionSet.ResourceToAccess)
			}
			delete(newPermissionSet.ResourceToAccess, clusterCVEResourceName)
			permissionSetsToInsert = append(permissionSetsToInsert, newPermissionSet)
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
