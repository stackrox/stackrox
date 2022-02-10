package service

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
)

const defaultScopeName = ""

// This function ensures that no APIToken with permissions more than principal's can be created.
// For each requested tuple (access scope, resource, accessLevel) we check that either:
// * principal has permission on this resource with unrestricted access scope
// * principal has permission on this resource with requested access scope
func verifyNoPrivilegeEscalation(userRoles, requestedRoles []permissions.ResolvedRole) error {
	// Group roles by access scope.
	userRolesByScope := make(map[string][]permissions.ResolvedRole)
	for _, userRole := range userRoles {
		scopeName := userRole.GetAccessScope().GetName()
		userRolesByScope[scopeName] = append(userRolesByScope[scopeName], userRole)
	}

	// Verify that for each tuple (access scope, resource, accessLevel) we have enough permissions.
	var multiErr error
	for _, requestedRole := range requestedRoles {
		scopeName := requestedRole.GetAccessScope().GetName()
		applicablePermissions := utils.NewUnionPermissions(append(userRolesByScope[scopeName], userRolesByScope[defaultScopeName]...))
		err := comparePermissions(requestedRole.GetPermissions(), applicablePermissions, scopeName)
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		}
	}
	return multiErr
}

func comparePermissions(requestedPerms, applicablePerms map[string]storage.Access, scopeName string) error {
	var multiErr error
	for requestedResource, requestedAccess := range requestedPerms {
		userAccess := applicablePerms[requestedResource]
		if userAccess < requestedAccess {
			err := newPrivilegeEscalationError(requestedResource, scopeName, requestedAccess, userAccess)
			multiErr = multierror.Append(multiErr, err)
		}
	}
	return multiErr
}

func newPrivilegeEscalationError(requestedResource string, scopeName string, requestedAccess storage.Access, userAccess storage.Access) error {
	return errors.Errorf("resource=%s, access scope=%q: requested access is %s, when user access is %s",
		requestedResource, scopeName, requestedAccess, userAccess)
}
