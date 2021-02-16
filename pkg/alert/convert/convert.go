package convert

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// AlertToListAlert takes in a storage.Alert and returns a store.ListAlert
func AlertToListAlert(alert *storage.Alert) *storage.ListAlert {
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
			ClusterName: alert.GetDeployment().GetClusterName(),
			ClusterId:   alert.GetDeployment().GetClusterId(),
			Namespace:   alert.GetDeployment().GetNamespace(),
			NamespaceId: alert.GetDeployment().GetNamespaceId(),
			Inactive:    alert.GetDeployment().GetInactive(),
		},
		Tags:              alert.GetTags(),
		EnforcementAction: alert.GetEnforcement().GetAction(),
	}
	if alert.GetState() == storage.ViolationState_ACTIVE {
		listAlert.EnforcementCount = enforcementCount(alert)
	}
	return listAlert
}

func enforcementCount(alert *storage.Alert) int32 {
	if alert.GetEnforcement() == nil {
		return 0
	}

	// Since runtime enforcement is killing a pod, we can determine how many times
	// a runtime policy has been enforced.
	if alert.GetLifecycleStage() == storage.LifecycleStage_RUNTIME {
		return determineRuntimeEnforcementCount(alert)
	}
	// We assume for a given deploy time alert with enforcement, that it is currently being
	// enforced.
	if alert.GetLifecycleStage() == storage.LifecycleStage_DEPLOY {
		return 1
	}
	return 0
}

func determineRuntimeEnforcementCount(alert *storage.Alert) int32 {
	// Number of times a policy is enforced is only tracked for process violations.
	if alert.GetEnforcement().GetAction() != storage.EnforcementAction_KILL_POD_ENFORCEMENT {
		return 1
	}
	podIds := set.NewStringSet()
	for _, pi := range alert.GetProcessViolation().GetProcesses() {
		podIds.Add(pi.GetPodId())
	}
	return int32(podIds.Cardinality())
}

func toAlertDeploymentContainer(c *storage.Container) *storage.Alert_Deployment_Container {
	return &storage.Alert_Deployment_Container{
		Name:  c.GetName(),
		Image: c.GetImage(),
	}
}

// ToAlertDeployment converts a storage.Deployment to a Alert_Deployment
func ToAlertDeployment(deployment *storage.Deployment) *storage.Alert_Deployment_ {
	alertDeployment := &storage.Alert_Deployment{
		Id:          deployment.GetId(),
		Name:        deployment.GetName(),
		Type:        deployment.GetType(),
		Namespace:   deployment.GetNamespace(),
		NamespaceId: deployment.GetNamespaceId(),
		Labels:      deployment.GetLabels(),
		ClusterId:   deployment.GetClusterId(),
		ClusterName: deployment.GetClusterName(),
		Annotations: deployment.GetAnnotations(),
		Inactive:    deployment.GetInactive(),
	}

	for _, c := range deployment.GetContainers() {
		alertDeployment.Containers = append(alertDeployment.Containers, toAlertDeploymentContainer(c))
	}
	return &storage.Alert_Deployment_{Deployment: alertDeployment}
}
