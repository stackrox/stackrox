package datastore

import "github.com/stackrox/rox/generated/storage"

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

// ResolvedRole implementation for the old Role-only format.
type resolvedOnlyRoleImpl struct {
	role *storage.Role
}

func (rsor *resolvedOnlyRoleImpl) GetRoleName() string {
	return rsor.role.GetName()
}

func (rsor *resolvedOnlyRoleImpl) GetPermissions() map[string]storage.Access {
	return rsor.role.GetResourceToAccess()
}

func (rsor *resolvedOnlyRoleImpl) GetAccessScope() *storage.SimpleAccessScope {
	return nil
}
