package getters

import "bitbucket.org/stack-rox/apollo/central/dnrintegration"

// DNRIntegrationGetter provides the required access to DNR integrations for risk scoring.
type DNRIntegrationGetter interface {
	ForCluster(clusterID string) (integration dnrintegration.DNRIntegration, exists bool, err error)
}
