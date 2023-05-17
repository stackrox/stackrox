package m181tom182

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	permissionsetstore "github.com/stackrox/rox/migrator/migrations/m_180_to_m_181_group_role_permission_with_access_one/permissionsetstore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/sac"
)

const (
	batchSize = 500
)

// Role: Replaced resource
const (
	Role = "Role"
)

// Access: Replacement resource
const (
	Access = "Access"
)

var (
	replacements = map[string]string{
		Role: Access,
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

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	store := permissionsetstore.New(database.PostgresDB)

	migratedPermissionSets := make([]*storage.PermissionSet, 0, batchSize)
	err := store.Walk(ctx, func(obj *storage.PermissionSet) error {
		// Copy the permission set, removing the deprecated resource permissions, and keeping the
		// lowest access level between that of deprecated resource and their replacement
		// for the replacement resource.
		newPermissionSet := obj.Clone()
		newPermissionSet.ResourceToAccess = make(map[string]storage.Access, len(obj.GetResourceToAccess()))
		newPermissionSetNeedsWriteToDB := false
		for resource, accessLevel := range obj.GetResourceToAccess() {
			if replacement, found := replacements[resource]; found {
				newPermissionSetNeedsWriteToDB = true
				resource = replacement
			}
			newPermissionSet.ResourceToAccess[resource] =
				propagateAccessForPermission(resource, accessLevel, newPermissionSet.GetResourceToAccess())
		}
		if !newPermissionSetNeedsWriteToDB {
			return nil
		}
		migratedPermissionSets = append(migratedPermissionSets, newPermissionSet)
		if len(migratedPermissionSets) >= batchSize {
			err := store.UpsertMany(ctx, migratedPermissionSets)
			if err != nil {
				return err
			}
			migratedPermissionSets = migratedPermissionSets[:0]
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(migratedPermissionSets) > 0 {
		return store.UpsertMany(ctx, migratedPermissionSets)
	}
	return nil
}
