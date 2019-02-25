package utils

import (
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
)

// FillAccessList fills in the access list if the role uses the GlobalAccess field.
func FillAccessList(role *storage.Role) {
	// If the role has global access, fill in the full list of resources with R/W.
	if role.GetGlobalAccess() != storage.Access_NO_ACCESS {
		if len(role.GetResourceToAccess()) == 0 {
			role.ResourceToAccess = make(map[string]storage.Access)
		}
		for _, resource := range resources.ListAll() {
			if role.ResourceToAccess[string(resource)] == storage.Access_NO_ACCESS {
				role.ResourceToAccess[string(resource)] = role.GetGlobalAccess()
			}
		}
	}
}
