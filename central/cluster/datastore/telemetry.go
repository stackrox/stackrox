package datastore

import (
	"context"

	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/telemetry/centralclient"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

func trackClusterRegistered(cluster *storage.Cluster) {
	if cfg := centralclient.InstanceConfig(); cfg.Enabled() {
		props := map[string]any{
			"Cluster Type": cluster.GetType().String(),
			"Cluster ID":   cluster.GetId(),
			"Managed By":   cluster.GetManagedBy().String(),
		}
		cfg.Telemeter().Track("Secured Cluster Registered", cfg.ClientID, props)
	}
}

func trackClusterInitialized(cluster *storage.Cluster) {
	if cfg := centralclient.InstanceConfig(); cfg.Enabled() {
		props := map[string]any{
			"Health":       cluster.GetHealthStatus().OverallHealthStatus.String(),
			"Cluster Type": cluster.GetType().String(),
			"Cluster ID":   cluster.GetId(),
			"Managed By":   cluster.GetManagedBy().String(),
		}
		cfg.Telemeter().Track("Secured Cluster Initialized", cfg.ClientID, props)
	}
}

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
