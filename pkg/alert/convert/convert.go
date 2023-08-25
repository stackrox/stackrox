package convert

import (
	"strings"

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
		EnforcementAction: alert.GetEnforcement().GetAction(),
	}
	if alert.GetState() == storage.ViolationState_ACTIVE {
		listAlert.EnforcementCount = enforcementCount(alert)
	}

	if alert.GetDeployment() != nil {
		populateListAlertEntityInfoForDeployment(listAlert, alert.GetDeployment())
	} else if alert.GetResource() != nil {
		populateListAlertEntityInfoForResource(listAlert, alert.GetResource())
	}

	return listAlert
}

func populateListAlertEntityInfoForResource(listAlert *storage.ListAlert, resource *storage.Alert_Resource) {
	listAlert.Entity = &storage.ListAlert_Resource{
		Resource: &storage.ListAlert_ResourceEntity{
			Name: resource.GetName(),
		},
	}
	resStr := resource.GetResourceType().String()
	resEnt := storage.ListAlert_ResourceType(storage.Alert_Resource_ResourceType_value[resStr])
	listAlert.CommonEntityInfo = &storage.ListAlert_CommonEntityInfo{
		ClusterName:  resource.GetClusterName(),
		ClusterId:    resource.GetClusterId(),
		Namespace:    resource.GetNamespace(),
		NamespaceId:  resource.GetNamespaceId(),
		ResourceType: resEnt,
	}
}

func populateListAlertEntityInfoForDeployment(listAlert *storage.ListAlert, deployment *storage.Alert_Deployment) {
	listAlert.Entity = &storage.ListAlert_Deployment{
		Deployment: &storage.ListAlertDeployment{
			Id:          deployment.GetId(),
			Name:        deployment.GetName(),
			ClusterName: deployment.GetClusterName(),
			ClusterId:   deployment.GetClusterId(),
			Namespace:   deployment.GetNamespace(),
			NamespaceId: deployment.GetNamespaceId(),
			Inactive:    deployment.GetInactive(),
		},
	}
	listAlert.CommonEntityInfo = &storage.ListAlert_CommonEntityInfo{
		ClusterName:  deployment.GetClusterName(),
		ClusterId:    deployment.GetClusterId(),
		Namespace:    deployment.GetNamespace(),
		NamespaceId:  deployment.GetNamespaceId(),
		ResourceType: storage.ListAlert_DEPLOYMENT,
	}
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

// ToAlertDeployment converts a storage.Deployment to an Alert_Deployment
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

// ToAlertResource converts a storage.KubernetesEvent to an Alert_Resource_
func ToAlertResource(kubeEvent *storage.KubernetesEvent) *storage.Alert_Resource_ {
	// TODO: Cluster name and namespace id will have to be passed in here
	// That will come from runtime detector (currently detector.detectForDeployment). This is TBD until the detection piece is completed
	// (and ROX-7355 is done for cluster name)
	return &storage.Alert_Resource_{
		Resource: &storage.Alert_Resource{
			ResourceType: storage.Alert_Resource_ResourceType(storage.Alert_Resource_ResourceType_value[strings.ToUpper(kubeEvent.GetObject().GetResource().String())]),
			Name:         kubeEvent.GetObject().GetName(),
			ClusterId:    kubeEvent.GetObject().GetClusterId(),
			Namespace:    kubeEvent.GetObject().GetNamespace(),
		},
	}
}
