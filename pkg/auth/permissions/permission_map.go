package permissions

import (
	"github.com/stackrox/rox/generated/storage"
)

// PermissionMap maps resources to access levels. This can be used to realize an access-mode aware
// set of permissions.
type PermissionMap map[Resource]storage.Access

// Add adds a permission to the permission map. If the access mode is lower or equal to the currently
// stored access mode for this resource, it has no effect; otherwise, the access is mode for the resource
// is updated accordingly.
func (s PermissionMap) Add(res ResourceHandle, am storage.Access) {
	currAM := s[res.GetResource()]
	if currAM < am {
		s[res.GetResource()] = am
	}
}

// IsEmpty checks if the permission map is empty.
func (s PermissionMap) IsEmpty() bool {
	return len(s) == 0
}

// IsLessOrEqual returns true if this permission map does not contain any permission that is
// stronger than any of the permissions stored in the other map.
func (s PermissionMap) IsLessOrEqual(other PermissionMap) bool {
	for resource, am := range s {
		if am > other[resource] {
			return false
		}
	}
	return true
}
