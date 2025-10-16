package datastore

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/telemetry/centralclient"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

const securedClusterClient = "Secured Cluster"

func trackClusterRegistered(cluster *storage.Cluster) {
	props := map[string]any{
		"Cluster Type": cluster.GetType().String(),
		"Cluster ID":   cluster.GetId(),
		"Managed By":   cluster.GetManagedBy().String(),
	}

	c := centralclient.Singleton()
	groups := c.WithGroups()

	// Reported as the Central client.
	go c.Track("Secured Cluster Registered", props, groups...)

	// Update the secured cluster identity from its name and add the secured
	// cluster 'user' to the Tenant group:
	go c.Track("Secured Cluster Static Properties", nil,
		append(groups,
			telemeter.WithTraits(makeClusterProperties(cluster)),
			telemeter.WithClient(cluster.GetId(),
				securedClusterClient, cluster.GetMainImage()),
		)...)
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
	c := centralclient.Singleton()
	// Issue an event that makes the secured cluster identity effective:
	go c.Track("Secured Cluster Initialized", map[string]any{
		"Health": cluster.GetHealthStatus().GetOverallHealthStatus().String(),
	},
		append(c.WithGroups(),
			telemeter.WithClient(cluster.GetId(), securedClusterClient, cluster.GetMainImage()),
		)...)

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
	c := centralclient.Singleton()
	// This is a shortcut to avoid calling the cluster datastore in case
	// telemetry is for sure not enabled.
	// This call will block until the telemetry configuration is read from the
	// database.
	if !c.IsActive() {
		return
	}

	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster)))

	cluster, ok, err := Singleton().GetCluster(ctx, clusterID)
	if err != nil || !ok {
		return
	}
	props := makeClusterProperties(cluster)
	props["Total Nodes"] = metrics.GetNodeCount()
	props["CPU Capacity"] = metrics.GetCpuCapacity()
	props["Compliance Operator Version"] = metrics.GetComplianceOperatorVersion()

	if pmd := cluster.GetStatus().GetProviderMetadata(); pmd.GetProvider() != nil {
		switch pmd.WhichProvider() {
		case storage.ProviderMetadata_Aws_case:
			props["Provider"] = "AWS"
		case storage.ProviderMetadata_Azure_case:
			props["Provider"] = "Azure"
		case storage.ProviderMetadata_Google_case:
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

	c.Track("Updated Secured Cluster Identity", nil, append(
		c.WithGroups(),
		telemeter.WithClient(clusterID, securedClusterClient, cluster.GetMainImage()),
		telemeter.WithTraits(props),
		telemeter.WithNoDuplicates(time.Now().Format(time.DateOnly)),
	)...)
}
