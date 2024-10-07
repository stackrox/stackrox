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

// IsDeployTimeAlertResult returns whether or not the passed results are from a deploy time policy
func IsDeployTimeAlertResult(alert *central.AlertResults) bool {
	return alert.GetStage() == storage.LifecycleStage_DEPLOY
}

// IsRuntimeAlertResult returns whether or not the passed results are from a runtime policy
func IsRuntimeAlertResult(alert *central.AlertResults) bool {
	return alert.GetStage() == storage.LifecycleStage_RUNTIME
}

// IsAlertResultResolved returns if there is a resolved alert within the alert result
func IsAlertResultResolved(alert *central.AlertResults) bool {
	for _, a := range alert.GetAlerts() {
		if a.GetState() == storage.ViolationState_RESOLVED {
			return true
		}
	}
	return false
}
