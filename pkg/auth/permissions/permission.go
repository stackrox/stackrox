package permissions

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Resource is a string representation of an exposed set of API endpoints (services).
type Resource string

// View returns the permission struct for viewing resource r.
func View(resource Resource) *v1.Permission {
	return &v1.Permission{Resource: string(resource), Access: storage.Access_READ_ACCESS}
}

// Modify returns the permission struct for modifying resource r.
func Modify(resource Resource) *v1.Permission {
	return &v1.Permission{Resource: string(resource), Access: storage.Access_READ_WRITE_ACCESS}
}
