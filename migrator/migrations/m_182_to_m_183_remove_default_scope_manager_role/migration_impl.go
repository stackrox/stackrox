package m182tom183

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	groupStore "github.com/stackrox/rox/migrator/migrations/m_181_to_m_182_remove_default_scope_manager_role/groupstore"
	permissionSetStore "github.com/stackrox/rox/migrator/migrations/m_181_to_m_182_remove_default_scope_manager_role/permissionsetstore"
	roleStore "github.com/stackrox/rox/migrator/migrations/m_181_to_m_182_remove_default_scope_manager_role/rolestore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
)

// Pre- and Post- migration role and permission set names
const (
	ScopeManagerRoleName = "Scope Manager"

	deprecatedPrefix            = "[DEPRECATED] "
	deprecatedDescriptionSuffix = ". DEPRECATED, please use \"Vulnerability Report Creator\" " +
		"instead for vulnerability report management purposes"

	scopeManagerPermissionSetID = "ffffffff-ffff-fff4-f5ff-fffffffffffb"
)

const (
	batchSize = 500
)

func pushDeprecatedScopeManagerRoleAndPermissionSet(
	ctx context.Context,
	roleStorage roleStore.Store,
	permissionSetStorage permissionSetStore.Store,
) error {
	oldPermissionSet, permissionSetFound, err := permissionSetStorage.Get(ctx, scopeManagerPermissionSetID)
	if err != nil {
		// TODO: log failure
		return err
	} else if !permissionSetFound {
		return errox.NotFound
	}
	newPermissionSet := oldPermissionSet.Clone()
	newPermissionSet.Name = deprecatedPrefix + oldPermissionSet.GetName()
	newPermissionSet.Description = oldPermissionSet.GetDescription() + deprecatedDescriptionSuffix
	newPermissionSet.Traits = &storage.Traits{
		Origin: storage.Traits_IMPERATIVE,
	}
	permissionSetUpsertErr := permissionSetStorage.UpsertMany(ctx, []*storage.PermissionSet{newPermissionSet})
	if permissionSetUpsertErr != nil {
		// TODO: log failure
		return permissionSetUpsertErr
	}

	oldRole, roleFound, err := roleStorage.Get(ctx, ScopeManagerRoleName)
	if err != nil {
		// TODO: log failure
		return err
	} else if !roleFound {
		return errox.NotFound
	}
	newRole := oldRole.Clone()
	newRole.Name = deprecatedPrefix + oldRole.GetName()
	newRole.Description = oldRole.GetDescription() + deprecatedDescriptionSuffix
	newRole.Traits = &storage.Traits{
		Origin: storage.Traits_IMPERATIVE,
	}
	roleUpsertErr := roleStorage.UpsertMany(ctx, []*storage.Role{newRole})
	if roleUpsertErr != nil {
		// TODO: log failure
		return roleUpsertErr
	}

	return nil
}

func removeOldDefaultScopeManagerRole(
	ctx context.Context,
	roleStorage roleStore.Store,
) error {
	return roleStorage.DeleteMany(ctx, []string{ScopeManagerRoleName})
}

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())

	groupStorage := groupStore.New(database.PostgresDB)
	permissionSetStorage := permissionSetStore.New(database.PostgresDB)
	roleStorage := roleStore.New(database.PostgresDB)
	usesScopeManager := false
	// Check whether the old default role is used
	useCheckErr := groupStorage.Walk(ctx, func(obj *storage.Group) error {
		if !usesScopeManager && obj.GetRoleName() == ScopeManagerRoleName {
			usesScopeManager = true
		}
		return nil
	})
	if useCheckErr != nil {
		return useCheckErr
	}
	if usesScopeManager {
		err := pushDeprecatedScopeManagerRoleAndPermissionSet(ctx, roleStorage, permissionSetStorage)
		if err != nil {
			return err
		}
		// Update groups to use the modified role
		updatedGroups := make([]*storage.Group, 0)
		scanGroupErr := groupStorage.Walk(ctx, func(obj *storage.Group) error {
			if obj.GetRoleName() == ScopeManagerRoleName {
				obj.RoleName = deprecatedPrefix + ScopeManagerRoleName
				updatedGroups = append(updatedGroups, obj)
				if len(updatedGroups) >= batchSize {
					groupUpdateErr := groupStorage.UpsertMany(ctx, updatedGroups)
					if groupUpdateErr != nil {
						return groupUpdateErr
					}
					updatedGroups = updatedGroups[:0]
				}
			}
		})
		if scanGroupErr != nil {
			return scanGroupErr
		}
		if len(updatedGroups) > 0 {
			groupUpdateErr := groupStorage.UpsertMany(ctx, updatedGroups)
			if groupUpdateErr != nil {
				return groupUpdateErr
			}
		}
	}
	// Remove default role
	defaultRoleRemovalErr := roleStorage.DeleteMany(ctx, []string{ScopeManagerRoleName})
	if defaultRoleRemovalErr != nil {
		return defaultRoleRemovalErr
	}

	return nil
}

// TODO: Write the additional code to support the migration

// TODO: remove any pending TODO
