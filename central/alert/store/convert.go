package store

import (
	"github.com/stackrox/rox/generated/api/v1"
)

func convertAlertsToListAlerts(alert *v1.Alert) *v1.ListAlert {
	listAlert := &v1.ListAlert{
		Id:             alert.GetId(),
		Time:           alert.GetTime(),
		State:          alert.GetState(),
		LifecycleStage: alert.GetLifecycleStage(),
		Policy: &v1.ListAlertPolicy{
			Id:          alert.GetPolicy().GetId(),
			Name:        alert.GetPolicy().GetName(),
			Severity:    alert.GetPolicy().GetSeverity(),
			Description: alert.GetPolicy().GetDescription(),
			Categories:  alert.GetPolicy().GetCategories(),
		},
		Deployment: &v1.ListAlertDeployment{
			Id:          alert.GetDeployment().GetId(),
			Name:        alert.GetDeployment().GetName(),
			UpdatedAt:   alert.GetDeployment().GetUpdatedAt(),
			ClusterName: alert.GetDeployment().GetClusterName(),
			Namespace:   alert.GetDeployment().GetNamespace(),
		},
	}
	if alert.GetState() == v1.ViolationState_ACTIVE {
		addEnforcementCount(alert, listAlert)
	}
	return listAlert
}

func addEnforcementCount(alert *v1.Alert, listAlert *v1.ListAlert) {
	if alert.GetEnforcement() == nil {
		return
	}

	// Since runtime enforcement is killing a pod, we can determine how many times
	// a runtime policy has been enforced.
	if alert.GetLifecycleStage() == v1.LifecycleStage_RUNTIME {
		listAlert.EnforcementCount = determineRuntimeEnforcementCount(alert.GetViolations())
		return
	}
	// We assume for a given deploy time alert with enforcement, that it is currently being
	// enforced.
	if alert.GetLifecycleStage() == v1.LifecycleStage_DEPLOY {
		listAlert.EnforcementCount = 1
		return
	}
}

func determineRuntimeEnforcementCount(violations []*v1.Alert_Violation) int32 {
	podIds := make(map[string]struct{})
	for _, violation := range violations {
		for _, pi := range violation.GetProcesses() {
			podIds[pi.GetPodId()] = struct{}{}
		}
	}
	return int32(len(podIds))
}
