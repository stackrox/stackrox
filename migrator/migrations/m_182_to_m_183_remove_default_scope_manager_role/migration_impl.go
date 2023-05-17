package m182tom183

import (
	"context"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	apiTokenStore "github.com/stackrox/rox/migrator/migrations/m_181_to_m_182_remove_default_scope_manager_role/apitokenstore"
	groupStore "github.com/stackrox/rox/migrator/migrations/m_181_to_m_182_remove_default_scope_manager_role/groupstore"
	permissionSetStore "github.com/stackrox/rox/migrator/migrations/m_181_to_m_182_remove_default_scope_manager_role/permissionsetstore"
	roleStore "github.com/stackrox/rox/migrator/migrations/m_181_to_m_182_remove_default_scope_manager_role/rolestore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
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

var (
	log = logging.LoggerForModule()

	errNotFound = errors.New("not found")
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())

	apiTokenStorage := apiTokenStore.New(database.PostgresDB)
	groupStorage := groupStore.New(database.PostgresDB)
	permissionSetStorage := permissionSetStore.New(database.PostgresDB)
	roleStorage := roleStore.New(database.PostgresDB)

	var migrateErrors *multierror.Error
	canRemoveRole := true
	// Check whether the old default role is used
	usesScopeManagerRole, referencingTokenIDs, useRoleCheckErr := isScopeManagerRoleReferenced(ctx, apiTokenStorage, groupStorage)
	if useRoleCheckErr != nil {
		canRemoveRole = false
		migrateErrors = multierror.Append(migrateErrors, useRoleCheckErr)
	}
	if usesScopeManagerRole {
		if addRoleErr := addDeprecatedScopeManagerRole(ctx, roleStorage); addRoleErr != nil {
			// It is safe to fail fast here. The remaining work is:
			// - Update of the references to the role (not possible as the replacement is not available).
			// - Removal of the old role (not possible as the replacement is not available).
			// - Update of the permission set to flag it as `[DEPRECATED]` (would be misleading
			// without further recovery instructions).
			migrateErrors = multierror.Append(migrateErrors, addRoleErr)
			return migrateErrors.ErrorOrNil()
		}
		tokenUpdateErr := updateScopeManagerRoleReferencesInAPITokens(ctx, apiTokenStorage, referencingTokenIDs)
		if tokenUpdateErr != nil {
			canRemoveRole = false
			migrateErrors = multierror.Append(migrateErrors, tokenUpdateErr)
		}
		groupUpdateErr := updateScopeManagerRoleReferencesInGroups(ctx, groupStorage)
		if groupUpdateErr != nil {
			canRemoveRole = false
			migrateErrors = multierror.Append(migrateErrors, tokenUpdateErr)
		}
	}
	if canRemoveRole {
		if roleRemoveErr := removeDefaultScopeManagerRole(ctx, roleStorage); roleRemoveErr != nil {
			// It is safe to fail fast here. The remaining work is around the default permission set,
			// which cannot be removed (remaining reference). It would also be misleading to flag it
			// as deprecated without further recovery instructions.
			migrateErrors = multierror.Append(migrateErrors, roleRemoveErr)
			return migrateErrors.ErrorOrNil()
		}
	}
	// If the role cannot be removed, neither can the permission set.
	// Otherwise, the ability to remove the permission set depends on
	// potential references from other roles.
	canRemovePermissionSet := canRemoveRole
	// Check for permission set references
	usesScopeManagerPermissionSet, usePermissionSetCheckErr := isScopeManagerPermissionSetReferenced(ctx, roleStorage)
	if usePermissionSetCheckErr != nil {
		canRemovePermissionSet = false
		migrateErrors = multierror.Append(migrateErrors, usePermissionSetCheckErr)
	}
	if usesScopeManagerPermissionSet {
		// The Role -> PermissionSet reference is using the PermissionSet ID which will be left untouched.
		// Therefore, there is no need to update the PermissionSet reference in the roles.
		// The permission set Name and Description will be updated.
		updatePermissionSetErr := updateScopeManagerPermissionSet(ctx, permissionSetStorage)
		if updatePermissionSetErr != nil {
			migrateErrors = multierror.Append(migrateErrors, updatePermissionSetErr)
		}
	} else if canRemovePermissionSet {
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
		if roleName == ScopeManagerRoleName {
			return true
		}
	}
	return false
}

func groupHasScopeManagerRole(obj *storage.Group) bool {
	return obj.GetRoleName() == ScopeManagerRoleName
}

func isScopeManagerRoleReferenced(ctx context.Context, apiTokenStorage apiTokenStore.Store, groupStorage groupStore.Store) (bool, []string, error) {
	roleReferenceFound := false
	var err *multierror.Error
	referencingTokenIDs := make([]string, 0)
	tokenWalkErr := apiTokenStorage.Walk(ctx, func(obj *storage.TokenMetadata) error {
		hasScopeManagerRole := apiTokenHasScopeManagerRole(obj)
		if hasScopeManagerRole {
			roleReferenceFound = true
			referencingTokenIDs = append(referencingTokenIDs, obj.GetId())
		}
		return nil
	})
	if tokenWalkErr != nil {
		err = multierror.Append(err, tokenWalkErr)
	}
	if roleReferenceFound {
		return true, referencingTokenIDs, nil
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
	multiErr := err.ErrorOrNil()
	if multiErr != nil {
		return false, nil, multiErr
	}
	return roleReferenceFound, nil, nil
}

func addDeprecatedScopeManagerRole(ctx context.Context, roleStorage roleStore.Store) error {
	oldRole, oldRoleFound, err := roleStorage.Get(ctx, ScopeManagerRoleName)
	if err != nil {
		return err
	}
	if !oldRoleFound {
		return errors.Wrap(errNotFound, "looking up role")
	}
	newRole := oldRole.Clone()
	newRole.Name = deprecatedPrefix + ScopeManagerRoleName
	newRole.Traits = &storage.Traits{
		// Keep default MutabilityMode : Traits_ALLOW_MUTATE
		// Keep default Visibility : Traits_VISIBLE
		Origin: storage.Traits_IMPERATIVE,
	}
	newRole.Description = oldRole.GetDescription() + deprecatedDescriptionSuffix

	return roleStorage.UpsertMany(ctx, []*storage.Role{newRole})
}

func removeDefaultScopeManagerRole(ctx context.Context, roleStorage roleStore.Store) error {
	return roleStorage.DeleteMany(ctx, []string{ScopeManagerRoleName})
}

func logGroupUpdateFailure(groups []*storage.Group, err error) {
	groupsForLogging := make([]string, 0, len(groups))
	for _, g := range groups {
		groupsForLogging = append(groupsForLogging, g.String())
	}
	log.Errorf(
		"Failed to update %q role reference for groups %q (error: %v)",
		ScopeManagerRoleName,
		strings.Join(groupsForLogging, ","),
		err,
	)
}

func updateScopeManagerRoleReferencesInGroups(ctx context.Context, groupStorage groupStore.Store) error {
	var err *multierror.Error
	groupsToUpdate := make([]*storage.Group, 0, batchSize)
	walkErr := groupStorage.Walk(ctx, func(obj *storage.Group) error {
		if !groupHasScopeManagerRole(obj) {
			return nil
		}
		updatedGroup := obj.Clone()
		updatedGroup.RoleName = deprecatedPrefix + ScopeManagerRoleName
		groupsToUpdate = append(groupsToUpdate, updatedGroup)
		if len(groupsToUpdate) >= batchSize {
			upsertErr := groupStorage.UpsertMany(ctx, groupsToUpdate)
			if upsertErr != nil {
				logGroupUpdateFailure(groupsToUpdate, upsertErr)
				err = multierror.Append(err, upsertErr)
			}
			groupsToUpdate = groupsToUpdate[:0]
		}
		return nil
	})
	if walkErr != nil {
		err = multierror.Append(err, walkErr)
	}
	if len(groupsToUpdate) >= batchSize {
		upsertErr := groupStorage.UpsertMany(ctx, groupsToUpdate)
		if upsertErr != nil {
			logGroupUpdateFailure(groupsToUpdate, upsertErr)
			err = multierror.Append(err, upsertErr)
		}
		groupsToUpdate = groupsToUpdate[:0]
	}
	return err.ErrorOrNil()
}

func logAPITokenUpdateFailure(tokenIDs []string, err error) {
	log.Errorf(
		"Failed to update %q role reference for API Tokens %q (error: %v)",
		ScopeManagerRoleName,
		strings.Join(tokenIDs, ","),
		err,
	)
}

func batchUpdateScopeManagerRoleReferencesInAPITokens(ctx context.Context, apiTokenStorage apiTokenStore.Store, tokenIDBatch []string) error {
	fetchedTokens, _, fetchErr := apiTokenStorage.GetMany(ctx, tokenIDBatch)
	if fetchErr != nil {
		return fetchErr
	}
	tokensToUpdate := make([]*storage.TokenMetadata, 0, len(fetchedTokens))
	for _, token := range fetchedTokens {
		updatedToken := token.Clone()
		updatedRoles := make([]string, 0, len(token.GetRoles()))
		for _, roleName := range token.GetRoles() {
			if roleName == ScopeManagerRoleName {
				roleName = deprecatedPrefix + ScopeManagerRoleName
			}
			updatedRoles = append(updatedRoles, roleName)
		}
		updatedToken.Roles = updatedRoles
		tokensToUpdate = append(tokensToUpdate, updatedToken)
	}
	return apiTokenStorage.UpsertMany(ctx, tokensToUpdate)
}

func updateScopeManagerRoleReferencesInAPITokens(ctx context.Context, apiTokenStorage apiTokenStore.Store, referencingTokenIDs []string) error {
	var err *multierror.Error
	tokenIDsToFetch := make([]string, 0, batchSize)
	for _, tokenID := range referencingTokenIDs {
		tokenIDsToFetch = append(tokenIDsToFetch, tokenID)
		if len(tokenIDsToFetch) >= batchSize {
			batchErr := batchUpdateScopeManagerRoleReferencesInAPITokens(ctx, apiTokenStorage, tokenIDsToFetch)
			if batchErr != nil {
				logAPITokenUpdateFailure(tokenIDsToFetch, batchErr)
				err = multierror.Append(err, batchErr)
			}
			tokenIDsToFetch = tokenIDsToFetch[:0]
		}
	}
	if len(tokenIDsToFetch) >= batchSize {
		batchErr := batchUpdateScopeManagerRoleReferencesInAPITokens(ctx, apiTokenStorage, tokenIDsToFetch)
		if batchErr != nil {
			logAPITokenUpdateFailure(tokenIDsToFetch, batchErr)
			err = multierror.Append(err, batchErr)
		}
	}
	return err.ErrorOrNil()
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
	oldPS, psFound, lookupErr := permissionSetStorage.Get(ctx, scopeManagerPermissionSetID)
	if lookupErr != nil {
		return lookupErr
	}
	if !psFound {
		return errors.Wrap(errNotFound, "looking up premission set")
	}
	newPS := oldPS.Clone()
	newPS.Traits = &storage.Traits{
		// Keep default MutabilityMode : Traits_ALLOW_MUTATE
		// Keep default Visibility : Traits_VISIBLE
		Origin: storage.Traits_IMPERATIVE,
	}
	newPS.Name = deprecatedPrefix + oldPS.GetName()
	newPS.Description = oldPS.GetDescription() + deprecatedDescriptionSuffix
	return permissionSetStorage.UpsertMany(ctx, []*storage.PermissionSet{newPS})
}

func removeDefaultScopeManagerPermissionSet(ctx context.Context, permissionSetStorage permissionSetStore.Store) error {
	return permissionSetStorage.DeleteMany(ctx, []string{scopeManagerPermissionSetID})
}
