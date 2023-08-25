package m181tom182

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	permissionsetstore "github.com/stackrox/rox/migrator/migrations/m_181_to_m_182_group_role_permission_with_access_one/permissionsetstore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/sac"
)

const (
	batchSize = 500

	// Role : Replaced resource
	Role = "Role"

	// Access : Replacement resource
	Access = "Access"
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	store := permissionsetstore.New(database.PostgresDB)

	migratedPermissionSets := make([]*storage.PermissionSet, 0, batchSize)
	err := store.Walk(ctx, func(obj *storage.PermissionSet) error {
		needsRewrite := false
		for resource, accessLevel := range obj.GetResourceToAccess() {
			// If Role permission found, merge with Access one (keep lowest access of the two),
			// remove role permission and batch for DB update.
			if resource == Role {
				needsRewrite = true
				accessLevelForPermissionAccess, entryFound := obj.GetResourceToAccess()[Access]
				if !entryFound {
					// Access not set, Role level propagated
					accessLevelForPermissionAccess = accessLevel
				} else if accessLevel < accessLevelForPermissionAccess {
					// Access set with les restrictive level than Role, propagate minimum level
					accessLevelForPermissionAccess = accessLevel
				}
				obj.ResourceToAccess[Access] = accessLevelForPermissionAccess
				delete(obj.ResourceToAccess, Role)
			}
		}
		if !needsRewrite {
			return nil
		}
		migratedPermissionSets = append(migratedPermissionSets, obj)
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
