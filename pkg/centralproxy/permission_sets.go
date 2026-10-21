package centralproxy

import "github.com/stackrox/rox/generated/storage"

const (
	// ConsoleProxyPermissionSetID identifies the fixed permission set for console tokens.
	ConsoleProxyPermissionSetID = "b3070020-ecc3-4f34-a3f6-ad28ffc9b80b"
	// MetricsPermissionSetID identifies the permission set shared with Central Prometheus metrics.
	MetricsPermissionSetID = "b3070020-ecc3-4f34-a3f6-ad28ffc9b80e"
)

// ConsoleProxyPermissions returns the permissions authorized by Sensor for console tokens.
func ConsoleProxyPermissions() map[string]storage.Access {
	return map[string]storage.Access{
		"Deployment": storage.Access_READ_ACCESS,
		"Image":      storage.Access_READ_ACCESS,
	}
}
