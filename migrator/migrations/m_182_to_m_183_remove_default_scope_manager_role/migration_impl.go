package m182tom183

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/stackrox/rox/generated/storage"
	apiTokenStore "github.com/stackrox/rox/migrator/migrations/m_182_to_m_183_remove_default_scope_manager_role/apitokenstore"
	groupStore "github.com/stackrox/rox/migrator/migrations/m_182_to_m_183_remove_default_scope_manager_role/groupstore"
	permissionSetStore "github.com/stackrox/rox/migrator/migrations/m_182_to_m_183_remove_default_scope_manager_role/permissionsetstore"
	roleStore "github.com/stackrox/rox/migrator/migrations/m_182_to_m_183_remove_default_scope_manager_role/rolestore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
)

const (
	scopeManagerObjectName = "Scope Manager"

	oldScopeManagerDescription = "For users: use it to create and modify scopes for the purpose of access control or vulnerability reporting"

	updatedDescriptionSuffix = ". This used to be a system object and should have been removed but was still " +
		"referenced. Please review your usage of that object and remove if not needed."

	scopeManagerPermissionSetID = "ffffffff-ffff-fff4-f5ff-fffffffffffb"

	unrestrictedAccessScopeID = "ffffffff-ffff-fff4-f5ff-ffffffffffff"

	access    = "Access"
	cluster   = "Cluster"
	namespace = "Namespace"
)

var (
	log = logging.LoggerForModule()

	imperativeObjectTraits = &storage.Traits{
		// Keep default MutabilityMode : Traits_ALLOW_MUTATE
		// Keep default Visibility : Traits_VISIBLE
		Origin: storage.Traits_IMPERATIVE,
	}
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())

	apiTokenStorage := apiTokenStore.New(database.PostgresDB)
	groupStorage := groupStore.New(database.PostgresDB)
	permissionSetStorage := permissionSetStore.New(database.PostgresDB)
	roleStorage := roleStore.New(database.PostgresDB)

	var migrateErrors *multierror.Error
	// Check whether the old default role is used
	usesScopeManagerRole, useRoleCheckErr := isScopeManagerRoleReferenced(ctx, apiTokenStorage, groupStorage)
	if useRoleCheckErr != nil {
		migrateErrors = multierror.Append(migrateErrors, useRoleCheckErr)
	}
	if usesScopeManagerRole {
		if updateRoleErr := updateScopeManagerRole(ctx, roleStorage); updateRoleErr != nil {
			log.Errorf("Failed to update %s role: error %v", scopeManagerObjectName, updateRoleErr)
			migrateErrors = multierror.Append(migrateErrors, updateRoleErr)
		}
	} else {
		if removeRoleErr := removeDefaultScopeManagerRole(ctx, roleStorage); removeRoleErr != nil {
			log.Errorf("Failed to remove %s role: error %v", scopeManagerObjectName, removeRoleErr)
			migrateErrors = multierror.Append(migrateErrors, removeRoleErr)
		}
	}

	usesScopeManagerPermissionSet, usePermissionSetCheckErr := isScopeManagerPermissionSetReferenced(ctx, roleStorage)
	if usePermissionSetCheckErr != nil {
		// In case of error, assume the permission set is still in use.
		usesScopeManagerPermissionSet = true
		migrateErrors = multierror.Append(migrateErrors, usePermissionSetCheckErr)
	}
	if usesScopeManagerPermissionSet {
		updatePermissionSetErr := updateScopeManagerPermissionSet(ctx, permissionSetStorage)
		if updatePermissionSetErr != nil {
			migrateErrors = multierror.Append(migrateErrors, updatePermissionSetErr)
		}
	} else {
		deletePermissionSetErr := removeDefaultScopeManagerPermissionSet(ctx, permissionSetStorage)
		if deletePermissionSetErr != nil {
			migrateErrors = multierror.Append(migrateErrors, deletePermissionSetErr)
		}
	}
	return migrateErrors.ErrorOrNil()
}

func apiTokenHasScopeManagerRole(obj *storage.TokenMetadata) bool {
	tokenRoles := obj.GetRoles()
	for _, roleName := range tokenRoles {
		if roleName == scopeManagerObjectName {
			return true
		}
	}
	return false
}

func groupHasScopeManagerRole(obj *storage.Group) bool {
	return obj.GetRoleName() == scopeManagerObjectName
}

func isScopeManagerRoleReferenced(ctx context.Context, apiTokenStorage apiTokenStore.Store, groupStorage groupStore.Store) (bool, error) {
	roleReferenceFound := false
	var err *multierror.Error
	tokenWalkErr := apiTokenStorage.Walk(ctx, func(obj *storage.TokenMetadata) error {
		hasScopeManagerRole := apiTokenHasScopeManagerRole(obj)
		if hasScopeManagerRole {
			roleReferenceFound = true
		}
		return nil
	})
	if tokenWalkErr != nil {
		err = multierror.Append(err, tokenWalkErr)
	}
	if roleReferenceFound {
		return true, nil
	}
	groupWalkErr := groupStorage.Walk(ctx, func(obj *storage.Group) error {
		if roleReferenceFound {
			return nil
		}
		if groupHasScopeManagerRole(obj) {
			roleReferenceFound = true
		}
		return nil
	})
	if groupWalkErr != nil {
		err = multierror.Append(err, groupWalkErr)
	}
	if roleReferenceFound {
		return true, nil
	}
	multiErr := err.ErrorOrNil()
	if multiErr != nil {
		return false, multiErr
	}
	return roleReferenceFound, nil
}

func updateScopeManagerRole(ctx context.Context, roleStorage roleStore.Store) error {
	// Push the expected Role content to the storage regardless of any previous value
	scopeManagerRole := &storage.Role{
		Name:            scopeManagerObjectName,
		Description:     oldScopeManagerDescription + updatedDescriptionSuffix,
		PermissionSetId: scopeManagerPermissionSetID,
		AccessScopeId:   unrestrictedAccessScopeID,
		Traits:          imperativeObjectTraits,
	}
	return roleStorage.UpsertMany(ctx, []*storage.Role{scopeManagerRole})
}

func removeDefaultScopeManagerRole(ctx context.Context, roleStorage roleStore.Store) error {
	return roleStorage.DeleteMany(ctx, []string{scopeManagerObjectName})
}

func roleHasScopeManagerPermissionSet(obj *storage.Role) bool {
	return obj.GetPermissionSetId() == scopeManagerPermissionSetID
}

func isScopeManagerPermissionSetReferenced(ctx context.Context, roleStorage roleStore.Store) (bool, error) {
	permissionSetReferenceFound := false
	walkErr := roleStorage.Walk(ctx, func(obj *storage.Role) error {
		if permissionSetReferenceFound {
			return nil
		}
		if roleHasScopeManagerPermissionSet(obj) {
			permissionSetReferenceFound = true
		}
		return nil
	})
	if walkErr != nil {
		return false, walkErr
	}
	return permissionSetReferenceFound, nil
}

func updateScopeManagerPermissionSet(ctx context.Context, permissionSetStorage permissionSetStore.Store) error {
	scopeManagerPermissionSet := &storage.PermissionSet{
		Id:          scopeManagerPermissionSetID,
		Name:        scopeManagerObjectName,
		Description: oldScopeManagerDescription + updatedDescriptionSuffix,
		ResourceToAccess: map[string]storage.Access{
			// The permission set is meant to be able to create Access Scopes.
			// This now requires write to Access, and read to Cluster and Namespace
			access:    storage.Access_READ_WRITE_ACCESS,
			cluster:   storage.Access_READ_ACCESS,
			namespace: storage.Access_READ_ACCESS,
		},
		Traits: imperativeObjectTraits,
	}
	return permissionSetStorage.UpsertMany(ctx, []*storage.PermissionSet{scopeManagerPermissionSet})
}

func removeDefaultScopeManagerPermissionSet(ctx context.Context, permissionSetStorage permissionSetStore.Store) error {
	return permissionSetStorage.DeleteMany(ctx, []string{scopeManagerPermissionSetID})
}
