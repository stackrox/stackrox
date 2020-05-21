package runtime

import (
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Detector provides an interface for performing runtime policy violation detection.
type Detector interface {
	PolicySet() detection.PolicySet

	DeploymentWhitelistedForPolicy(deploymentID, policyID string) bool
	DeploymentInactive(deploymentID string) bool
}

// NewDetector returns a new instance of a Detector.
func NewDetector(policySet detection.PolicySet, deployments datastore.DataStore) Detector {
	return &detectorImpl{
		policySet:   policySet,
		deployments: deployments,
	}
}
