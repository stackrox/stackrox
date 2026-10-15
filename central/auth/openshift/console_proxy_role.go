package openshift

import (
	"context"

	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
)

const (
	consoleProxyPermissionSetID = "b3070020-ecc3-4f34-a3f6-ad28ffc9b80b"
	consoleProxyPermissionSetName = "OpenShift Secured Cluster Console Reader"
)

// SeedConsoleProxyRole creates the permission source for Sensor's console proxy.
// The stored role grants no cluster access when assigned directly to a user;
// internal tokens receive the scope validated against the requesting Sensor.
func SeedConsoleProxyRole(ctx context.Context, roleDS roleDataStore.DataStore) {
	if _, exists, err := roleDS.GetPermissionSet(ctx, consoleProxyPermissionSetID); err != nil {
		log.Warnf("Failed to check OpenShift console proxy permission set: %v", err)
		return
	} else if !exists {
		ps := &storage.PermissionSet{
			Id:          consoleProxyPermissionSetID,
			Name:        consoleProxyPermissionSetName,
			Description: "Read access for Sensor-proxied OpenShift console requests",
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

	if _, exists, err := roleDS.GetRole(ctx, centralproxy.ConsoleProxyRole); err != nil {
		log.Warnf("Failed to check OpenShift console proxy role: %v", err)
	} else if !exists {
		role := &storage.Role{
			Name:            centralproxy.ConsoleProxyRole,
			Description:     "Read access for OpenShift console requests on the Sensor's cluster",
			PermissionSetId: consoleProxyPermissionSetID,
			AccessScopeId:   accesscontrol.DefaultAccessScopeIDs[accesscontrol.DenyAllAccessScope],
		}
		if err := roleDS.AddRole(ctx, role); err != nil {
			log.Warnf("Failed to create OpenShift console proxy role: %v", err)
		}
	}
}
