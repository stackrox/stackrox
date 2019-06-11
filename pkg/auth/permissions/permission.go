package permissions

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// View returns the permission struct for viewing resource r.
func View(resource ResourceHandle) *v1.Permission {
	return &v1.Permission{Resource: string(resource.GetResource()), Access: storage.Access_READ_ACCESS}
}

// Modify returns the permission struct for modifying resource r.
func Modify(resource ResourceHandle) *v1.Permission {
	return &v1.Permission{Resource: string(resource.GetResource()), Access: storage.Access_READ_WRITE_ACCESS}
}
