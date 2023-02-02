package datastore

import (
	"context"

	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/telemetry/centralclient"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

const securedClusterClient = "Secured Cluster"

func trackClusterRegistered(ctx context.Context, cluster *storage.Cluster) {
	if cfg := centralclient.InstanceConfig(); cfg.Enabled() {
		userID := cfg.ClientID
		if id, err := authn.IdentityFromContext(ctx); err == nil {
			userID = cfg.HashUserAuthID(id)
		}
		props := map[string]any{
			"Cluster Type": cluster.GetType().String(),
			"Cluster ID":   cluster.GetId(),
			"Managed By":   cluster.GetManagedBy().String(),
		}
		cfg.Telemeter().TrackUserAs(userID, "", "", "Secured Cluster Registered", props)

		// Add the secured cluster 'user' to the Tenant group:
		cfg.Telemeter().GroupUserAs(cluster.GetId(), "", "", cfg.GroupID, nil)

		// Update the secured cluster identity from its name:
		cfg.Telemeter().IdentifyUserAs(cluster.GetId(), cluster.GetId(), securedClusterClient,
			makeClusterProperties(cluster))
	}
}

func makeClusterProperties(cluster *storage.Cluster) map[string]any {
	return map[string]any{
		"Main Image":           cluster.GetMainImage(),
		"Admission Controller": cluster.GetAdmissionController(),
		"Collection Method":    cluster.GetCollectionMethod().String(),
		"Collector Image":      cluster.GetCollectorImage(),
		"Managed By":           cluster.GetManagedBy().String(),
		"Priority":             cluster.GetPriority(),
		"Cluster Type":         cluster.GetType().String(),
		"Slim Collector":       cluster.GetSlimCollector(),
	}
}

func trackClusterInitialized(cluster *storage.Cluster) {
	if cfg := centralclient.InstanceConfig(); cfg.Enabled() {
		// Issue an event that makes the secured cluster identity effective:
		cfg.Telemeter().TrackUserAs(cluster.GetId(), cluster.GetId(), securedClusterClient,
			"Secured Cluster Initialized", map[string]any{
				"Health": cluster.GetHealthStatus().GetOverallHealthStatus().String(),
			})
	}
}

// Gather the number of clusters.
var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster)))

	props := make(map[string]any, 1)
	if err := phonehome.AddTotal(ctx, props, "Secured Clusters", Singleton().CountClusters); err != nil {
		return nil, err
	}
	return props, nil
}

// UpdateSecuredClusterIdentity is called by the clustermetrics pipeline on
// the reception of the cluster metrics from a sensor.
func UpdateSecuredClusterIdentity(ctx context.Context, clusterID string, metrics *central.ClusterMetrics) {
	if cfg := centralclient.InstanceConfig(); cfg.Enabled() {
		ctx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Cluster)))

		cluster, ok, err := Singleton().GetCluster(ctx, clusterID)
		if err != nil || !ok {
			return
		}
		props := makeClusterProperties(cluster)
		props["Total Nodes"] = metrics.NodeCount
		props["CPU Capacity"] = metrics.CpuCapacity
		cfg.Telemeter().IdentifyUserAs(cluster.GetId(), cluster.GetId(), securedClusterClient, props)
		cfg.Telemeter().TrackUserAs(cluster.GetId(), cluster.GetId(), securedClusterClient, "Updated Secured Cluster Identity", nil)
	}
}
