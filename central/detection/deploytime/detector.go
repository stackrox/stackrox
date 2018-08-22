package deploytime

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/generated/api/v1"
)

// Detector provides an interface for getting and managing alerts and enforcements on deployments.
type Detector interface {
	DeploymentUpdated(deployment *v1.Deployment) (string, v1.EnforcementAction, error)
	UpsertPolicy(policy *v1.Policy) error

	DeploymentRemoved(deployment *v1.Deployment) error
	RemovePolicy(policyID string) error
}

// NewDetector returns a new instance of a Detector.
func NewDetector(policySet PolicySet,
	alertManager AlertManager,
	enricher enrichment.Enricher,
	deployments deploymentDataStore.DataStore) Detector {
	return &detectorImpl{
		policySet:    policySet,
		alertManager: alertManager,
		enricher:     enricher,
		deployments:  deployments,
	}
}
