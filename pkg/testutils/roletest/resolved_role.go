package roletest

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// NewResolvedRole creates an instance of ResolvedRole from passed parameters.
func NewResolvedRole(roleName string, permissions map[string]storage.Access, accessScope *storage.SimpleAccessScope) permissions.ResolvedRole {
	return &resolvedRoleImpl{
		roleName:    roleName,
		permissions: permissions,
		accessScope: accessScope,
	}
}

// NewResolvedRoleWithDenyAll creates an instance of ResolvedRole with the
// 'Deny All' access scope.
func NewResolvedRoleWithDenyAll(roleName string, perms map[string]storage.Access) permissions.ResolvedRole {
	return &resolvedRoleImpl{
		roleName:    roleName,
		permissions: perms,
		accessScope: nil, // i.e., Deny All.
	}
}

type resolvedRoleImpl struct {
	roleName    string
	permissions map[string]storage.Access
	accessScope *storage.SimpleAccessScope
}

func (rr *resolvedRoleImpl) GetRoleName() string {
	return rr.roleName
}
func (rr *resolvedRoleImpl) GetPermissions() map[string]storage.Access {
	return rr.permissions
}
func (rr *resolvedRoleImpl) GetAccessScope() *storage.SimpleAccessScope {
	return rr.accessScope
}
