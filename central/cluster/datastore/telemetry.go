package datastore

import (
	"context"

	"github.com/stackrox/rox/central/telemetry/centralclient"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

const securedClusterClient = "Secured Cluster"

func trackClusterRegistered(ctx context.Context, cluster *storage.Cluster) {
	if cfg := centralclient.InstanceConfig(); cfg.Enabled() {
		var userID string
		if id, err := authn.IdentityFromContext(ctx); err == nil {
			userID = cfg.HashUserAuthID(id)
		}
		props := map[string]any{
			"Cluster Type": cluster.GetType().String(),
			"Cluster ID":   cluster.GetId(),
			"Managed By":   cluster.GetManagedBy().String(),
		}
		groups := telemeter.WithGroups(cfg.GroupType, cfg.GroupID)

		cfg.Telemeter().Track("Secured Cluster Registered", props, telemeter.WithUserID(userID), groups)

		// Update the secured cluster identity from its name and add the secured
		// cluster 'user' to the Tenant group:
		cfg.Telemeter().Track("Secured Cluster Static Properties", nil,
			telemeter.WithTraits(makeClusterProperties(cluster)),
			telemeter.WithClient(cluster.GetId(), securedClusterClient),
			groups)
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
		cfg.Telemeter().
			Track("Secured Cluster Initialized", map[string]any{
				"Health": cluster.GetHealthStatus().GetOverallHealthStatus().String(),
			},
				telemeter.WithClient(cluster.GetId(), securedClusterClient),
				telemeter.WithGroups(cfg.GroupType, cfg.GroupID))
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

		if pmd := cluster.GetStatus().GetProviderMetadata(); pmd.GetProvider() != nil {
			switch pmd.GetProvider().(type) {
			case *storage.ProviderMetadata_Aws:
				props["Provider"] = "AWS"
			case *storage.ProviderMetadata_Azure:
				props["Provider"] = "Azure"
			case *storage.ProviderMetadata_Google:
				props["Provider"] = "Google"
			default:
				props["Provider"] = "Unknown"
			}
			props["Provider Region"] = pmd.GetRegion()
			props["Provider Zone"] = pmd.GetZone()
			props["Provider Verified"] = pmd.GetVerified()
		}

		omd := cluster.GetStatus().GetOrchestratorMetadata()
		if omd.GetIsOpenshift() != nil {
			props["Openshift"] = omd.GetIsOpenshift()
		}
		props["Orchestrator Version"] = omd.GetVersion()

		opts := []telemeter.Option{
			telemeter.WithClient(cluster.GetId(), securedClusterClient),
			telemeter.WithGroups(cfg.GroupType, cfg.GroupID),
			telemeter.WithTraits(props),
		}
		cfg.Telemeter().Track("Updated Secured Cluster Identity", nil, opts...)
	}
}
