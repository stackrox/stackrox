package defaults

import (
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/role/validator"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionsUtils "github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/env"
)

// GetDefaultRoles returns the default ACS roles.
func GetDefaultRoles() []*storage.Role {
	roles := make([]*storage.Role, 0, DefaultRoleNames.Cardinality())

	defaultPermSetsAsMap := make(map[string]*storage.PermissionSet)
	for _, ps := range GetDefaultPermissionSets() {
		defaultPermSetsAsMap[ps.GetName()] = ps
	}

	for _, roleName := range DefaultRoleNames.AsSlice() {
		// Historically, we have named default roles and permission sets same to make it easier for users to map one to the other.
		permSet := defaultPermSetsAsMap[roleName]

		role := &storage.Role{
			Name:          roleName,
			Description:   permSet.GetDescription(),
			AccessScopeId: AccessScopeIncludeAll.GetId(),
			Traits: &storage.Traits{
				Origin: storage.Traits_DEFAULT,
			},
			PermissionSetId: permSet.GetId(),
		}
		roles = append(roles, role)
	}
	return roles
}

// GetDefaultRole returns the default ACS role with specified name.
func GetDefaultRole(name string) *storage.Role {
	// Historically, we have named default roles and permission sets same to make it easier for users to map one to the other.
	permSet := GetDefaultPermissionSet(name)
	if permSet == nil {
		return nil
	}

	return &storage.Role{
		Name:          name,
		Description:   permSet.GetDescription(),
		AccessScopeId: AccessScopeIncludeAll.GetId(),
		Traits: &storage.Traits{
			Origin: storage.Traits_DEFAULT,
		},
		PermissionSetId: permSet.GetId(),
	}
}

// IsDefaultRole checks if a given role corresponds to a default role.
func IsDefaultRole(role *storage.Role) bool {
	return role.GetTraits().GetOrigin() == storage.Traits_DEFAULT || DefaultRoleNames.Contains(role.GetName())
}

// GetDefaultPermissionSets returns the default permission sets are shipped OOTB with ACS.
func GetDefaultPermissionSets() []*storage.PermissionSet {
	permissionSets := make([]*storage.PermissionSet, 0, len(defaultPermissionSets))

	for name, attributes := range defaultPermissionSets {
		resourceToAccess := permissionsUtils.FromResourcesWithAccess(attributes.resourceWithAccess...)

		permissionSet := &storage.PermissionSet{
			Id:               attributes.getID(),
			Name:             name,
			Description:      attributes.description,
			ResourceToAccess: resourceToAccess,
			Traits: &storage.Traits{
				Origin: storage.Traits_DEFAULT,
			},
		}
		permissionSets = append(permissionSets, permissionSet)
	}
	return permissionSets
}

// GetDefaultPermissionSet returns the default permission set with specified name.
func GetDefaultPermissionSet(name string) *storage.PermissionSet {
	for pname, attributes := range defaultPermissionSets {
		if pname == name {
			resourceToAccess := permissionsUtils.FromResourcesWithAccess(attributes.resourceWithAccess...)

			return &storage.PermissionSet{
				Id:               attributes.getID(),
				Name:             name,
				Description:      attributes.description,
				ResourceToAccess: resourceToAccess,
				Traits: &storage.Traits{
					Origin: storage.Traits_DEFAULT,
				},
			}
		}
	}
	return nil
}

// IsDefaultPermissionSet checks if a given permission set corresponds to a default role.
func IsDefaultPermissionSet(permissionSet *storage.PermissionSet) bool {
	_, defaultPermSet := defaultPermissionSets[permissionSet.GetName()]
	return permissionSet.GetTraits().GetOrigin() == storage.Traits_DEFAULT || defaultPermSet
}

// GetAnalystPermissions returns permissions for `Analyst` role.
func GetAnalystPermissions() []permissions.ResourceWithAccess {
	resourceToAccess := resources.AllResourcesViewPermissions()
	for i, resourceWithAccess := range resourceToAccess {
		if resourceWithAccess.Resource.GetResource() == resources.Administration.GetResource() {
			return append(resourceToAccess[:i], resourceToAccess[i+1:]...)
		}
	}
	panic("Administration resource was not found amongst all resources.")
}

// GetDefaultAccessScopes returns the default access scopes.
func GetDefaultAccessScopes() []*storage.SimpleAccessScope {
	return []*storage.SimpleAccessScope{
		AccessScopeIncludeAll,
		AccessScopeExcludeAll,
	}
}

// IsDefaultAccessScope checks if a given access scope corresponds to a
// default access scope.
func IsDefaultAccessScope(scope *storage.SimpleAccessScope) bool {
	return scope.GetTraits().GetOrigin() == storage.Traits_DEFAULT || defaultScopesIDs.Contains(scope.GetId())
}

func getAccessScopeExcludeAllID() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return denyAllAccessScopeID
	}
	return validator.EnsureValidAccessScopeID("denyall")
}

func getAccessScopeIncludeAllID() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return unrestrictedAccessScopeID
	}
	return validator.EnsureValidAccessScopeID("unrestricted")
}
