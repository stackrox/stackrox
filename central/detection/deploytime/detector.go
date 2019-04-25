package deploytime

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/generated/storage"
)

// DetectionContext is the context for detection
type DetectionContext struct {
	EnforcementOnly bool
}

// Detector provides an interface for getting and managing alerts and enforcements on deployments.
type Detector interface {
	PolicySet() detection.PolicySet

	Detect(ctx DetectionContext, deployment *storage.Deployment) ([]*storage.Alert, error)
	AlertsForDeployment(deployment *storage.Deployment) ([]*storage.Alert, error)
	AlertsForPolicy(policyID string) ([]*storage.Alert, error)
}

// NewDetector returns a new instance of a Detector.
func NewDetector(policySet detection.PolicySet, deployments deploymentDataStore.DataStore) Detector {
	return &detectorImpl{
		policySet:   policySet,
		deployments: deployments,
	}
}
