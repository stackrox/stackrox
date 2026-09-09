package datastore

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/stackrox/rox/central/telemetry/centralclient"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	clusterPkg "github.com/stackrox/rox/pkg/cluster"
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
		"Main Image":                  cluster.GetMainImage(),
		"Admission Controller":        cluster.GetAdmissionController(),
		"Collection Method":           cluster.GetCollectionMethod().String(),
		"Collector Image":             cluster.GetCollectorImage(),
		"Managed By":                  cluster.GetManagedBy().String(),
		"Priority":                    cluster.GetPriority(),
		"Cluster Type":                cluster.GetType().String(),
		"Slim Collector":              cluster.GetSlimCollector(),
		"Auto Lock Process Baselines": clusterPkg.GetAutoLockProcessBaselinesEnabled(cluster),
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

type versionEntry struct {
	Version string `json:"version"`
	Count   int32  `json:"count"`
}

// buildVMTraits returns Segment trait key/value pairs for VM telemetry.
// Returns nil when the Sensor lacks VirtualMachineTelemetryCap (old Sensor,
// leave existing traits untouched).
func buildVMTraits(hasCapability bool, vmm *central.VirtualMachineMetrics) map[string]any {
	if !hasCapability {
		return nil
	}
	if vmm == nil {
		return map[string]any{
			"VM Scanning Enabled":     false,
			"VM Tracked Count":        int32(0),
			"VM Scanned Count":        int32(0),
			"VM Unscanned Count":      int32(0),
			"Roxagent Version Counts": "[]",
		}
	}

	traits := map[string]any{
		"VM Scanning Enabled":     true,
		"VM Tracked Count":        vmm.GetTrackedVms(),
		"VM Scanned Count":        vmm.GetVmsScanned(),
		"VM Unscanned Count":      max(int32(0), vmm.GetTrackedVms()-vmm.GetVmsScanned()),
		"Roxagent Version Counts": "[]",
	}

	if vc := vmm.GetRoxagentVersionCounts(); len(vc) > 0 {
		entries := make([]versionEntry, 0, len(vc))
		for v, c := range vc {
			entries = append(entries, versionEntry{Version: v, Count: c})
		}
		slices.SortFunc(entries, func(a, b versionEntry) int {
			return strings.Compare(a.Version, b.Version)
		})
		raw, err := json.Marshal(entries)
		if err != nil {
			log.Warnf("Failed to marshal roxagent version counts: %v", err)
			return traits
		}
		traits["Roxagent Version Counts"] = string(raw)
	}

	return traits
}

// UpdateSecuredClusterIdentity is called by the clustermetrics pipeline on
// the reception of the cluster metrics from a sensor.
func UpdateSecuredClusterIdentity(ctx context.Context, clusterID string, metrics *central.ClusterMetrics, hasVMTelemetryCap bool) {
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

	props["Sensor Version Compatibility"] = cluster.GetStatus().GetSensorVersionCompatibility().String()

	if vmTraits := buildVMTraits(hasVMTelemetryCap, metrics.GetVirtualMachineMetrics()); vmTraits != nil {
		maps.Copy(props, vmTraits)
	}

	c.Track("Updated Secured Cluster Identity", nil, append(
		c.WithGroups(),
		telemeter.WithClient(clusterID, securedClusterClient, cluster.GetMainImage()),
		telemeter.WithTraits(props),
		telemeter.WithNoDuplicates(time.Now().Format(time.DateOnly)),
	)...)
}
