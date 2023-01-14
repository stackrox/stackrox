package datastore

import (
	"github.com/stackrox/rox/central/telemetry/centralclient"
	"github.com/stackrox/rox/generated/storage"
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
