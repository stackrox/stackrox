package openshift

import (
	"context"

	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralproxy"
)

const consoleProxyPermissionSetName = "OpenShift Secured Cluster Console Proxy"

// SeedConsoleProxyPermissionSet creates the configurable permission source for Sensor's console proxy.
// Internal tokens receive the scope validated against the requesting Sensor.
func SeedConsoleProxyPermissionSet(ctx context.Context, roleDS roleDataStore.DataStore) {
	if _, exists, err := roleDS.GetPermissionSet(ctx, centralproxy.ConsoleProxyPermissionSetID); err != nil {
		log.Warnf("Failed to check OpenShift console proxy permission set: %v", err)
		return
	} else if !exists {
		ps := &storage.PermissionSet{
			Id:          centralproxy.ConsoleProxyPermissionSetID,
			Name:        consoleProxyPermissionSetName,
			Description: "Access for Sensor-proxied OpenShift console requests",
			ResourceToAccess: map[string]storage.Access{
				"Deployment": storage.Access_READ_ACCESS,
				"Image":      storage.Access_READ_ACCESS,
			},
		}
		if err := roleDS.AddPermissionSet(ctx, ps); err != nil {
			log.Warnf("Failed to create OpenShift console proxy permission set: %v", err)
			return
		}
	}
}
