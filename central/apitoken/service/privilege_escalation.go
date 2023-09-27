package service

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/logging"
)

var defaultScopeID = role.AccessScopeIncludeAll.Id

// This function ensures that no APIToken with permissions more than principal's can be created.
// For each requested tuple (access scope, resource, accessLevel) we check that either:
// * principal has permission on this resource with unrestricted access scope
// * principal has permission on this resource with requested access scope
func verifyNoPrivilegeEscalation(userRoles, requestedRoles []permissions.ResolvedRole) error {
	// Group roles by access scope.
	userRolesByScope := make(map[string][]permissions.ResolvedRole)
	for _, userRole := range userRoles {
		scopeID := userRole.GetAccessScope().GetId()
		userRolesByScope[scopeID] = append(userRolesByScope[scopeID], userRole)
	}

	// Verify that for each tuple (access scope, resource, accessLevel) we have enough permissions.
	var multiErr error
	for _, requestedRole := range requestedRoles {
		logging.GetRateLimitedLogger().Warnf("Working on a role %s", requestedRole.GetRoleName())
		scopeID := requestedRole.GetAccessScope().GetId()
		applicablePermissions := utils.NewUnionPermissions(append(userRolesByScope[scopeID], userRolesByScope[defaultScopeID]...))
		err := comparePermissions(requestedRole, applicablePermissions)
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		}
	}
	return multiErr
}

func comparePermissions(requestedRole permissions.ResolvedRole, applicablePerms map[string]storage.Access) error {
	var multiErr error
	accessScopeName := requestedRole.GetAccessScope().GetName()
	for requestedResource, requestedAccess := range requestedRole.GetPermissions() {
		userAccess := applicablePerms[requestedResource]
		if userAccess < requestedAccess {
			err := newPrivilegeEscalationError(requestedResource, accessScopeName, requestedAccess, userAccess)
			multiErr = multierror.Append(multiErr, err)
		}
	}
	return multiErr
}

func newPrivilegeEscalationError(requestedResource string, scopeName string, requestedAccess storage.Access, userAccess storage.Access) error {
	return errors.Errorf("resource=%s, access scope=%q: requested access is %s, when user access is %s",
		requestedResource, scopeName, requestedAccess, userAccess)
}
