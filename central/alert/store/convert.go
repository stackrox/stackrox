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
	addEnforcementCount(alert, listAlert)
	return listAlert
}

// Since runtime enforcement is killing a pod, we can determine how many times
// a runtime policy has been enforced by counting unique pod ids.
func addEnforcementCount(alert *v1.Alert, listAlert *v1.ListAlert) {
	if alert.GetLifecycleStage() != v1.LifecycleStage_RUNTIME || alert.GetEnforcement() == nil {
		return
	}
	listAlert.EnforcementCount = determineEnforcementCount(alert.GetViolations())
}

func determineEnforcementCount(violations []*v1.Alert_Violation) int32 {
	podIds := make(map[string]struct{})
	for _, violation := range violations {
		for _, pi := range violation.GetProcesses() {
			podIds[pi.GetPodId()] = struct{}{}
		}
	}
	return int32(len(podIds))
}
