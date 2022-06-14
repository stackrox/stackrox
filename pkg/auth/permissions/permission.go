package permissions

import (
	"github.com/stackrox/rox/generated/storage"
)

// ResourceWithAccess stored a resource handle paired with an access level for configuring permissions.
type ResourceWithAccess struct {
	Resource ResourceMetadata
	Access   storage.Access
}

// View returns the permission struct for viewing resource r.
func View(resource ResourceMetadata) ResourceWithAccess {
	return ResourceWithAccess{Resource: resource, Access: storage.Access_READ_ACCESS}
}

// Modify returns the permission struct for modifying resource r.
func Modify(resource ResourceMetadata) ResourceWithAccess {
	return ResourceWithAccess{Resource: resource, Access: storage.Access_READ_WRITE_ACCESS}
}
