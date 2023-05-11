package m182tom183

import (
	"context"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/stackrox/rox/generated/storage"
	apiTokenStore "github.com/stackrox/rox/migrator/migrations/m_181_to_m_182_remove_default_scope_manager_role/apitokenstore"
	groupStore "github.com/stackrox/rox/migrator/migrations/m_181_to_m_182_remove_default_scope_manager_role/groupstore"
	permissionSetStore "github.com/stackrox/rox/migrator/migrations/m_181_to_m_182_remove_default_scope_manager_role/permissionsetstore"
	roleStore "github.com/stackrox/rox/migrator/migrations/m_181_to_m_182_remove_default_scope_manager_role/rolestore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/errox"
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
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())

	apiTokenStorage := apiTokenStore.New(database.PostgresDB)
	groupStorage := groupStore.New(database.PostgresDB)
	permissionSetStorage := permissionSetStore.New(database.PostgresDB)
	roleStorage := roleStore.New(database.PostgresDB)

	var err *multierror.Error
	canRemoveRole := true
	canRemovePermissionSet := true
	// Check whether the old default role is used
	usesScopeManagerRole, useRoleCheckErr := isScopeManagerRoleReferenced(ctx, apiTokenStorage, groupStorage)
	if useRoleCheckErr != nil {
		canRemoveRole = false
		err = multierror.Append(err, useRoleCheckErr)
	}
	if usesScopeManagerRole {
		addRoleErr := addDeprecatedScopeManagerRole(ctx, roleStorage)
		if addRoleErr != nil {
			canRemoveRole = false
			err = multierror.Append(err, addRoleErr)
		} else {
			tokenUpdateErr := updateScopeManagerRoleReferencesInAPITokens(ctx, apiTokenStorage)
			if tokenUpdateErr != nil {
				canRemoveRole = false
				err = multierror.Append(err, tokenUpdateErr)
			}
			groupUpdateErr := updateScopeManagerRoleReferencesInGroups(ctx, groupStorage)
			if groupUpdateErr != nil {
				canRemoveRole = false
				err = multierror.Append(err, tokenUpdateErr)
			}
		}
	}
	if canRemoveRole {
		roleRemoveErr := removeDefaultScopeManagerRole(ctx, roleStorage)
		if roleRemoveErr != nil {
			canRemovePermissionSet = false
			err = multierror.Append(err, roleRemoveErr)
		}
	} else {
		canRemovePermissionSet = false
	}
	// Check for permission set references
	usesScopeManagerPermissionSet, usePermissionSetCheckErr := isScopeManagerPermissionSetReferenced(ctx, roleStorage)
	if usePermissionSetCheckErr != nil {
		canRemovePermissionSet = false
		err = multierror.Append(err, usePermissionSetCheckErr)
	}
	if usesScopeManagerPermissionSet {
		// The Role -> PermissionSet reference is using the PermissionSet ID which will be left untouched.
		// Therefore, there is no need to update the PermissionSet reference in the roles.
		// The permission set Name and Description will be updated.
		updatePermissionSetErr := updateScopeManagerPermissionSet(ctx, permissionSetStorage)
		if updatePermissionSetErr != nil {
			err = multierror.Append(err, updatePermissionSetErr)
		}
	} else {
		if canRemovePermissionSet {
			deletePermissionSetErr := removeDefaultScopeManagerPermissionSet(ctx, permissionSetStorage)
			if deletePermissionSetErr != nil {
				err = multierror.Append(err, deletePermissionSetErr)
			}
		}
	}
	return err.ErrorOrNil()
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

func isScopeManagerRoleReferenced(ctx context.Context, apiTokenStorage apiTokenStore.Store, groupStorage groupStore.Store) (bool, error) {
	roleReferenceFound := false
	var err *multierror.Error
	tokenWalkErr := apiTokenStorage.Walk(ctx, func(obj *storage.TokenMetadata) error {
		if roleReferenceFound {
			return nil
		}
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
	multiErr := err.ErrorOrNil()
	if multiErr != nil {
		return false, multiErr
	}
	return roleReferenceFound, nil
}

func addDeprecatedScopeManagerRole(ctx context.Context, roleStorage roleStore.Store) error {
	oldRole, oldRoleFound, err := roleStorage.Get(ctx, ScopeManagerRoleName)
	if err != nil {
		return err
	}
	if !oldRoleFound {
		return errox.NotFound
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
		if groupHasScopeManagerRole(obj) {
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

func logAPITokenUpdateFailure(tokens []*storage.TokenMetadata, err error) {
	tokenIDs := make([]string, 0, len(tokens))
	for _, t := range tokens {
		tokenIDs = append(tokenIDs, t.GetId())
	}
	log.Errorf(
		"Failed to update %q role reference for API Tokens %q (error: %v)",
		ScopeManagerRoleName,
		strings.Join(tokenIDs, ","),
		err,
	)
}

func updateScopeManagerRoleReferencesInAPITokens(ctx context.Context, apiTokenStorage apiTokenStore.Store) error {
	var err *multierror.Error
	tokensToUpdate := make([]*storage.TokenMetadata, 0, batchSize)
	walkErr := apiTokenStorage.Walk(ctx, func(obj *storage.TokenMetadata) error {
		if apiTokenHasScopeManagerRole(obj) {
			updatedToken := obj.Clone()
			updatedRoles := make([]string, 0, len(obj.GetRoles()))
			for _, roleName := range obj.GetRoles() {
				if roleName == ScopeManagerRoleName {
					updatedRoles = append(updatedRoles, deprecatedPrefix+ScopeManagerRoleName)
				} else {
					updatedRoles = append(updatedRoles, roleName)
				}
			}
			updatedToken.Roles = updatedRoles
			tokensToUpdate = append(tokensToUpdate, updatedToken)
		}
		if len(tokensToUpdate) >= batchSize {
			upsertErr := apiTokenStorage.UpsertMany(ctx, tokensToUpdate)
			if upsertErr != nil {
				logAPITokenUpdateFailure(tokensToUpdate, upsertErr)
				err = multierror.Append(err, upsertErr)
			}
			tokensToUpdate = tokensToUpdate[:0]
		}
		return nil
	})
	if walkErr != nil {
		err = multierror.Append(walkErr)
	}
	if len(tokensToUpdate) >= batchSize {
		upsertErr := apiTokenStorage.UpsertMany(ctx, tokensToUpdate)
		if upsertErr != nil {
			logAPITokenUpdateFailure(tokensToUpdate, upsertErr)
			err = multierror.Append(err, upsertErr)
		}
		tokensToUpdate = tokensToUpdate[:0]
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
		return errox.NotFound
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
