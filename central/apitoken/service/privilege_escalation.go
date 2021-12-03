package service

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
)

const defaultScopeName = ""

// This function ensures that no APIToken with permissions more than principal has can be created.
// For each requested tuple (access scope, resource, accessLevel) we check that either:
// * principal has permission on this resource with unrestricted access scope
// * principal has permission on this resource with requested access scope
func verifyNoPrivilegeEscalation(userRoles, requestedRoles []permissions.ResolvedRole) error {
	// Group roles by access scope.
	userRolesByScope := make(map[string][]permissions.ResolvedRole)
	accessScopeByName := make(map[string]*storage.SimpleAccessScope)
	for _, userRole := range userRoles {
		scopeName := extractScopeName(userRole)
		accessScopeByName[scopeName] = userRole.GetAccessScope()
		userRolesByScope[scopeName] = append(userRolesByScope[scopeName], userRole)
	}

	// Verify that for each tuple (access scope, resource, accessLevel) we have enough permissions.
	var multiErr error
	for _, requestedRole := range requestedRoles {
		scopeName := extractScopeName(requestedRole)
		applicablePermissions := utils.NewUnionPermissions(append(userRolesByScope[scopeName], userRolesByScope[defaultScopeName]...))

		for requestedResource, requestedAccess := range requestedRole.GetPermissions() {
			userAccess := applicablePermissions[requestedResource]
			if userAccess < requestedAccess {
				err := errors.Errorf("resource=%s, access scope=%q: requested access is %s, when user access is %s",
					requestedResource, scopeName, requestedAccess, userAccess)
				multiErr = multierror.Append(multiErr, err)
			}
		}
	}
	return multiErr
}

func extractScopeName(userRole permissions.ResolvedRole) string {
	if userRole.GetAccessScope() != nil {
		return userRole.GetAccessScope().GetName()
	}
	return defaultScopeName
}
