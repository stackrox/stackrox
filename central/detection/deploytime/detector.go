package deploytime

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/deployment"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Detector provides an interface for getting and managing alerts and enforcements on deployments.
type Detector interface {
	AlertsForDeployment(deployment *storage.Deployment) ([]*v1.Alert, error)
	AlertsForPolicy(policyID string) ([]*v1.Alert, error)
	UpsertPolicy(policy *storage.Policy) error
	RemovePolicy(policyID string) error
}

// NewDetector returns a new instance of a Detector.
func NewDetector(policySet deployment.PolicySet, deployments deploymentDataStore.DataStore) Detector {
	return &detectorImpl{
		policySet:   policySet,
		deployments: deployments,
	}
}
