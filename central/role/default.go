package role

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
)

var (
	// defaultScopesIDs is a string set containing the names of all default (built-in) scopes.
	defaultScopesIDs = set.NewFrozenStringSet(AccessScopeIncludeAll.Id, AccessScopeExcludeAll.Id)

	// AccessScopeExcludeAll has empty rules and hence excludes all
	// scoped resources. Global resources must be unaffected.
	AccessScopeExcludeAll = &storage.SimpleAccessScope{
		Id:          getAccessScopeExcludeAllID(),
		Name:        accesscontrol.DenyAllAccessScope,
		Description: "No access to scoped resources",
		Rules:       &storage.SimpleAccessScope_Rules{},
		Traits: &storage.Traits{
			Origin: storage.Traits_DEFAULT,
		},
	}

	// AccessScopeIncludeAll gives access to all resources. It is checked by ID, as
	// Rules cannot represent unrestricted scope.
	AccessScopeIncludeAll = &storage.SimpleAccessScope{
		Id:          getAccessScopeIncludeAllID(),
		Name:        accesscontrol.UnrestrictedAccessScope,
		Description: "Access to all clusters and namespaces",
		Traits: &storage.Traits{
			Origin: storage.Traits_DEFAULT,
		},
	}
)

func getAccessScopeExcludeAllID() string {
	return accesscontrol.DefaultAccessScopeIDs[accesscontrol.DenyAllAccessScope]
}

func getAccessScopeIncludeAllID() string {
	return accesscontrol.DefaultAccessScopeIDs[accesscontrol.UnrestrictedAccessScope]
}

// IsDefaultRole checks if a given role corresponds to a default role.
func IsDefaultRole(role *storage.Role) bool {
	return role.GetTraits().GetOrigin() == storage.Traits_DEFAULT || accesscontrol.IsDefaultRole(role.GetName())
}

// IsDefaultPermissionSet checks if a given permission set corresponds to a default role.
func IsDefaultPermissionSet(permissionSet *storage.PermissionSet) bool {
	return permissionSet.GetTraits().GetOrigin() == storage.Traits_DEFAULT ||
		accesscontrol.IsDefaultPermissionSet(permissionSet.GetName())
}

// IsDefaultAccessScope checks if a given access scope corresponds to a
// default access scope.
func IsDefaultAccessScope(scope *storage.SimpleAccessScope) bool {
	return scope.GetTraits().GetOrigin() == storage.Traits_DEFAULT || defaultScopesIDs.Contains(scope.GetId())
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
