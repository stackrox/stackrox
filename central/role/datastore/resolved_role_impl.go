package datastore

import "github.com/stackrox/stackrox/generated/storage"

// ResolvedRole implementation for the new Role + Permission Set format.
type resolvedRoleImpl struct {
	role          *storage.Role
	permissionSet *storage.PermissionSet
	accessScope   *storage.SimpleAccessScope
}

func (rsr *resolvedRoleImpl) GetRoleName() string {
	return rsr.role.GetName()
}

func (rsr *resolvedRoleImpl) GetPermissions() map[string]storage.Access {
	return rsr.permissionSet.GetResourceToAccess()
}

func (rsr *resolvedRoleImpl) GetAccessScope() *storage.SimpleAccessScope {
	return rsr.accessScope
}
