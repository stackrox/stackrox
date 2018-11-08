package runtime

import (
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/deployment"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

// Detector provides an interface for performing runtime policy violation detection.
type Detector interface {
	AlertsForDeployments(deploymentIDs ...string) ([]*v1.Alert, error)
	AlertsForPolicy(policyID string) ([]*v1.Alert, error)
	DeploymentWhitelistedForPolicy(deploymentID, policyID string) bool
	UpsertPolicy(policy *v1.Policy) error
	RemovePolicy(policyID string) error
}

// NewDetector returns a new instance of a Detector.
func NewDetector(policySet deployment.PolicySet, deployments datastore.DataStore) Detector {
	return &detectorImpl{
		policySet:   policySet,
		deployments: deployments,
	}
}
