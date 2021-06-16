package permissions

import "github.com/stackrox/rox/generated/storage"

// ResolvedRole type unites role and corresponding permission set and access scope.
type ResolvedRole struct {
	Role          *storage.Role
	PermissionSet *storage.PermissionSet
	AccessScope   *storage.SimpleAccessScope
}

// GetResourceToAccess returns resource to access map.
func (r *ResolvedRole) GetResourceToAccess() map[string]storage.Access {
	return r.PermissionSet.GetResourceToAccess()
}
