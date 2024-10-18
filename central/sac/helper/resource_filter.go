package sachelper

import (
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sliceutils"
)

func listReadPermissions(
	requestedPermissions []string,
	scope permissions.ResourceScope,
) []permissions.ResourceWithAccess {
	readPermissions := resources.AllResourcesViewPermissions()
	indexedScopeReadPermissions := make(map[string]permissions.ResourceWithAccess, 0)
	scopeReadPermissions := make([]permissions.ResourceWithAccess, 0, len(readPermissions))
	for _, permission := range readPermissions {
		if permission.Resource.GetScope() >= scope {
			scopeReadPermissions = append(scopeReadPermissions, permission)
			indexedScopeReadPermissions[permission.Resource.String()] = permission
		}
	}
	if len(requestedPermissions) == 0 {
		return scopeReadPermissions
	}
	scopeRequestedReadPermissions := make([]permissions.ResourceWithAccess, 0, len(scopeReadPermissions))
	for _, permission := range sliceutils.Unique(requestedPermissions) {
		if resourceWithAccess, found := indexedScopeReadPermissions[permission]; found {
			scopeRequestedReadPermissions = append(scopeRequestedReadPermissions, resourceWithAccess)
		}
	}
	return scopeRequestedReadPermissions
}
