package service

import (
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/telemetry/centralclient"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

// telemetryServiceClient returns a telemetry client impersonation option.
func telemetryServiceClient(id authn.Identity, clusters clusterDS.DataStore) telemeter.Option {
	// ID cannot be nil normally.
	if id == nil {
		// The event will be reported from the central client in this case.
		// WithTraits(nil) is a no-op option.
		return telemeter.WithTraits(nil)
	}
	// Track from the name of the secured cluster, if the caller is sensor.
	switch id.Service().GetType() {
	case storage.ServiceType_SENSOR_SERVICE:
		clientID := id.Service().GetId()
		cluster, _, _ := clusters.GetCluster(clusterReadContext, clientID)
		return telemeter.WithClient(clientID, "Secured Cluster", cluster.GetMainImage())
	case storage.ServiceType_UNKNOWN_SERVICE:
		return telemeter.WithUserID(id.UID())
	default:
		return telemeter.WithClient(id.Service().GetId(), id.Service().GetType().String(), "")
	}
}

func trackRequest(id authn.Identity, req *v1.GenerateTokenForPermissionsAndScopeRequest) {
	if !centralclient.Singleton().IsActive() {
		// Avoid unnecessary calculations if telemetry is not active.
		return
	}
	maxNamespaces := 0
	fullClusterAccess := 0
	for _, cs := range req.GetClusterScopes() {
		maxNamespaces = max(maxNamespaces, len(cs.GetNamespaces()))
		if cs.GetFullClusterAccess() {
			fullClusterAccess++
		}
	}
	eventProps := make(map[string]any)
	eventProps["Total Cluster Scopes"] = len(req.GetClusterScopes())
	eventProps["Cluster Scopes With Full Access"] = fullClusterAccess
	eventProps["Max Namespaces In Scopes"] = maxNamespaces
	for p, a := range req.GetPermissions() {
		eventProps[p] = a.String()
	}
	centralclient.Singleton().Track("Internal Token Issued", eventProps, telemetryServiceClient(id, clusterDS.Singleton()),
		// Client traits:
		telemeter.WithTraits(map[string]any{
			"Has Internal Token Users": true,
		}))
}
