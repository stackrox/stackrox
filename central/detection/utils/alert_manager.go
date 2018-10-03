package utils

import (
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/generated/api/v1"
)

// AlertManager is a simplified interface for fetching and updating alerts.
//go:generate mockery -name=AlertManager
type AlertManager interface {
	GetAlertsByPolicy(policyID string) ([]*v1.Alert, error)
	GetAlertsByDeployment(deploymentID string) ([]*v1.Alert, error)
	GetAlertsByDeploymentAndPolicy(deploymentID, policyID string) (*v1.Alert, error)
	GetAlertsByDeploymentAndPolicyLifecycle(deploymentID string, lifecycle v1.LifecycleStage) ([]*v1.Alert, error)
	AlertAndNotify(previousAlerts, currentAlerts []*v1.Alert) error
}

// NewAlertManager returns a new instance of AlertManager. You should just use the singleton instance instead.
func NewAlertManager(notifier notifierProcessor.Processor, alerts alertDataStore.DataStore) AlertManager {
	return &alertManagerImpl{
		notifier: notifier,
		alerts:   alerts,
	}
}
