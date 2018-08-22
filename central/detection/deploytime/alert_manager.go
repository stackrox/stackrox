package deploytime

import (
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/generated/api/v1"
)

// AlertManager provides the interfaces for working with the alerts storage and notifier.
//go:generate mockery -name=AlertManager
type AlertManager interface {
	GetAlertsByPolicy(policyID string) ([]*v1.Alert, error)
	GetAlertsByDeployment(deploymentID string) ([]*v1.Alert, error)

	AlertAndNotify(previousAlerts, currentAlerts []*v1.Alert) error
}

// NewAlertManager returns a new instance of a AlertManager.
func NewAlertManager(notifier notifierProcessor.Processor, alerts alertDataStore.DataStore) AlertManager {
	return &alertManagerImpl{
		notifier: notifier,
		alerts:   alerts,
	}
}
