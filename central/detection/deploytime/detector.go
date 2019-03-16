package deploytime

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/deployment"
	"github.com/stackrox/rox/generated/storage"
)

// DetectionContext is the context for detection
type DetectionContext struct {
	EnforcementOnly bool
}

// Detector provides an interface for getting and managing alerts and enforcements on deployments.
type Detector interface {
	Detect(ctx DetectionContext, deployment *storage.Deployment) ([]*storage.Alert, error)
	AlertsForDeployment(deployment *storage.Deployment) ([]*storage.Alert, error)
	AlertsForPolicy(policyID string) ([]*storage.Alert, error)
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
