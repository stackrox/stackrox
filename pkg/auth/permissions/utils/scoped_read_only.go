package utils

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/resources"
)

// ValidateScopedReadOnlyPermissions rejects writes and grants to resources without a known cluster or namespace scope.
func ValidateScopedReadOnlyPermissions(grants map[string]storage.Access) error {
	for resource, access := range grants {
		switch access {
		case storage.Access_NO_ACCESS:
			continue
		case storage.Access_READ_ACCESS:
			metadata, known := resources.MetadataForResource(permissions.Resource(resource))
			if !known || (metadata.GetScope() != permissions.ClusterScope && metadata.GetScope() != permissions.NamespaceScope) {
				return fmt.Errorf("resource %q grants unscoped read access", resource)
			}
		default:
			return fmt.Errorf("resource %q grants non-read-only access %s", resource, access)
		}
	}
	return nil
}
