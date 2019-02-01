package store

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

func convertAlertsToListAlerts(alert *storage.Alert) *storage.ListAlert {
	listAlert := &storage.ListAlert{
		Id:             alert.GetId(),
		Time:           alert.GetTime(),
		State:          alert.GetState(),
		LifecycleStage: alert.GetLifecycleStage(),
		Policy: &storage.ListAlertPolicy{
			Id:          alert.GetPolicy().GetId(),
			Name:        alert.GetPolicy().GetName(),
			Severity:    alert.GetPolicy().GetSeverity(),
			Description: alert.GetPolicy().GetDescription(),
			Categories:  alert.GetPolicy().GetCategories(),
		},
		Deployment: &storage.ListAlertDeployment{
			Id:          alert.GetDeployment().GetId(),
			Name:        alert.GetDeployment().GetName(),
			UpdatedAt:   alert.GetDeployment().GetUpdatedAt(),
			ClusterName: alert.GetDeployment().GetClusterName(),
			Namespace:   alert.GetDeployment().GetNamespace(),
		},
	}
	if alert.GetState() == storage.ViolationState_ACTIVE {
		addEnforcementCount(alert, listAlert)
	}
	return listAlert
}

func addEnforcementCount(alert *storage.Alert, listAlert *storage.ListAlert) {
	if alert.GetEnforcement() == nil {
		return
	}

	// Since runtime enforcement is killing a pod, we can determine how many times
	// a runtime policy has been enforced.
	if alert.GetLifecycleStage() == storage.LifecycleStage_RUNTIME {
		listAlert.EnforcementCount = determineRuntimeEnforcementCount(alert)
		return
	}
	// We assume for a given deploy time alert with enforcement, that it is currently being
	// enforced.
	if alert.GetLifecycleStage() == storage.LifecycleStage_DEPLOY {
		listAlert.EnforcementCount = 1
		return
	}
}

func determineRuntimeEnforcementCount(alert *storage.Alert) int32 {
	podIds := set.NewStringSet()
	for _, pi := range alert.GetProcessViolation().GetProcesses() {
		podIds.Add(pi.GetPodId())
	}
	return int32(podIds.Cardinality())
}
