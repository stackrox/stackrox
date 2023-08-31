package m168tom169

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	permissionSetPostgresStore "github.com/stackrox/rox/migrator/migrations/m_168_to_m_169_postgres_remove_clustercve_permission/permissionsetpostgresstore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
)

const (
	batchSize = 500

	startSeqNum = 168

	// Cluster is the replacement resource
	Cluster = "Cluster"
	// ClusterCVE is the replaced resource
	ClusterCVE = "ClusterCVE"
)

var (
	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)}, // 169
		Run: func(databases *types.Databases) error {
			err := cleanupPermissionSets(databases.PostgresDB)
			if err != nil {
				return errors.Wrap(err, "updating PermissionSet schema")
			}
			return nil
		},
	}

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

func cleanupPermissionSets(db postgres.DB) error {
	ctx := sac.WithAllAccess(context.Background())
	permissionSetStore := permissionSetPostgresStore.New(db)
	permissionSetsToInsert := make([]*storage.PermissionSet, 0, batchSize)
	err := permissionSetStore.Walk(ctx, func(obj *storage.PermissionSet) error {
		if _, ok := obj.GetResourceToAccess()[ClusterCVE]; ok {
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
			delete(newPermissionSet.ResourceToAccess, ClusterCVE)
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
