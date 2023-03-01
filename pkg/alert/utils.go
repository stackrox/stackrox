package alert

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// IsDeployTimeAttemptedAlert indicates whether an alert is an attempted deploy-time alert.
func IsDeployTimeAttemptedAlert(alert *storage.Alert) bool {
	return IsAttemptedAlert(alert) &&
		alert.GetLifecycleStage() == storage.LifecycleStage_DEPLOY
}

// IsAttemptedAlert indicates whether an alert is an attempted alert.
func IsAttemptedAlert(alert *storage.Alert) bool {
	return alert.GetState() == storage.ViolationState_ATTEMPTED
}

// AnyAttemptedAlert indicates whether any alert is an attempted alert.
func AnyAttemptedAlert(alerts ...*storage.Alert) bool {
	for _, alert := range alerts {
		if IsAttemptedAlert(alert) {
			return true
		}
	}
	return false
}

// IsRuntimeAlertResult returns whether or not the passed results are from a runtime policy
func IsRuntimeAlertResult(alert *central.AlertResults) bool {
	return alert.GetStage() == storage.LifecycleStage_RUNTIME
}
