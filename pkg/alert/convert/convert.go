package convert

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// AlertToListAlert takes in a storage.Alert and returns a store.ListAlert
func AlertToListAlert(alert *storage.Alert) *storage.ListAlert {
	alertId := alert.GetId()
	alertState := alert.GetState()
	alertLifecycleStage := alert.GetLifecycleStage()
	policyId := alert.GetPolicy().GetId()
	policyName := alert.GetPolicy().GetName()
	policySeverity := alert.GetPolicy().GetSeverity()
	policyDescription := alert.GetPolicy().GetDescription()
	enforcementAction := alert.GetEnforcement().GetAction()

	listAlert := storage.ListAlert_builder{
		Id:             &alertId,
		Time:           alert.GetTime(),
		State:          &alertState,
		LifecycleStage: &alertLifecycleStage,
		Policy: storage.ListAlertPolicy_builder{
			Id:          &policyId,
			Name:        &policyName,
			Severity:    &policySeverity,
			Description: &policyDescription,
			Categories:  alert.GetPolicy().GetCategories(),
		}.Build(),
		EnforcementAction: &enforcementAction,
	}.Build()
	if alert.GetState() == storage.ViolationState_ACTIVE {
		listAlert.SetEnforcementCount(enforcementCount(alert))
	}

	if alert.GetDeployment() != nil {
		populateListAlertEntityInfoForDeployment(listAlert, alert.GetDeployment())
	} else if alert.GetResource() != nil {
		populateListAlertEntityInfoForResource(listAlert, alert.GetResource())
	}

	return listAlert
}

func populateListAlertEntityInfoForResource(listAlert *storage.ListAlert, resource *storage.Alert_Resource) {
	// TODO: Fix this after determining correct opaque API types
}

func populateListAlertEntityInfoForDeployment(listAlert *storage.ListAlert, deployment *storage.Alert_Deployment) {
	// TODO: Fix this after determining correct opaque API types
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
	containerName := c.GetName()
	return storage.Alert_Deployment_Container_builder{
		Name:  &containerName,
		Image: c.GetImage(),
	}.Build()
}

// ToAlertDeployment converts a storage.Deployment to an Alert_Deployment_
func ToAlertDeployment(deployment *storage.Deployment) interface{} {
	// TODO: Fix return type and implementation
	return nil
}

// ToAlertResource converts a storage.KubernetesEvent to an Alert_Resource_
func ToAlertResource(kubeEvent *storage.KubernetesEvent) interface{} {
	// TODO: Fix return type and implementation
	return nil
}
