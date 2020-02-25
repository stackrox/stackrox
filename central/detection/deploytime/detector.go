package deploytime

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/detection/deploytime"
)

// Detector provides an interface for getting and managing alerts and enforcements on deployments.
type Detector interface {
	PolicySet() detection.PolicySet

	Detect(ctx deploytime.DetectionContext, deployment *storage.Deployment, images []*storage.Image) ([]*storage.Alert, error)
	AlertsForPolicy(policyID string) ([]*storage.Alert, error)
}

// NewDetector returns a new instance of a Detector.
func NewDetector(policySet detection.PolicySet, deployments deploymentDataStore.DataStore) Detector {
	return &detectorImpl{
		policySet:      policySet,
		deployments:    deployments,
		singleDetector: deploytime.NewDetector(policySet),
	}
}
