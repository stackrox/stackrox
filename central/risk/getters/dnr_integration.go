package getters

import "github.com/stackrox/rox/central/dnrintegration"

// DNRIntegrationGetter provides the required access to DNR integrations for risk scoring.
type DNRIntegrationGetter interface {
	ForCluster(clusterID string) (integration dnrintegration.DNRIntegration, exists bool)
}
