package utils

import (
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
)

// FillAccessList fills in the access list if the role uses the GlobalAccess field.
func FillAccessList(role *storage.Role) {
	if role.GetGlobalAccess() == storage.Access_NO_ACCESS {
		return
	}
	// If the role has global access, fill in the full list of resources with the max of the role's current access and global access.
	if role.GetResourceToAccess() == nil {
		role.ResourceToAccess = make(map[string]storage.Access)
	}
	for _, resource := range resources.ListAll() {
		if role.ResourceToAccess[string(resource)] < role.GetGlobalAccess() {
			role.ResourceToAccess[string(resource)] = role.GetGlobalAccess()
		}
	}
}
