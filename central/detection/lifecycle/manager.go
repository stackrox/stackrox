package lifecycle

import (
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/central/detection/utils"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

// A Manager manages deployment/policy lifecycle updates.
type Manager interface {
	IndicatorAdded(indicator *v1.ProcessIndicator, deployment *v1.Deployment) (*v1.SensorEnforcement, error)
	// DeploymentUpdated processes a new or updated deployment, generating and updating alerts in the store and returning
	// enforcement action.
	DeploymentUpdated(deployment *v1.Deployment) (string, v1.EnforcementAction, error)
	UpsertPolicy(policy *v1.Policy) error

	DeploymentRemoved(deployment *v1.Deployment) error
	RemovePolicy(policyID string) error
}

// NewManager returns a new manager with the injected dependencies.
func NewManager(enricher enrichment.Enricher, deploytimeDetector deploytime.Detector,
	runtimeDetector runtime.Detector, alertManager utils.AlertManager) Manager {
	return &managerImpl{
		enricher:           enricher,
		deploytimeDetector: deploytimeDetector,
		runtimeDetector:    runtimeDetector,
		alertManager:       alertManager,
	}
}
