package convert

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/protobuf/proto"
)

// AlertToListAlert takes in a storage.Alert and returns a store.ListAlert
func AlertToListAlert(alert *storage.Alert) *storage.ListAlert {
	lap := &storage.ListAlertPolicy{}
	lap.SetId(alert.GetPolicy().GetId())
	lap.SetName(alert.GetPolicy().GetName())
	lap.SetSeverity(alert.GetPolicy().GetSeverity())
	lap.SetDescription(alert.GetPolicy().GetDescription())
	lap.SetCategories(alert.GetPolicy().GetCategories())
	listAlert := &storage.ListAlert{}
	listAlert.SetId(alert.GetId())
	listAlert.SetTime(alert.GetTime())
	listAlert.SetState(alert.GetState())
	listAlert.SetLifecycleStage(alert.GetLifecycleStage())
	listAlert.SetPolicy(lap)
	listAlert.SetEnforcementAction(alert.GetEnforcement().GetAction())
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
	lr := &storage.ListAlert_ResourceEntity{}
	lr.SetName(resource.GetName())
	listAlert.SetResource(proto.ValueOrDefault(lr))
	resStr := resource.GetResourceType().String()
	resEnt := storage.ListAlert_ResourceType(storage.Alert_Resource_ResourceType_value[resStr])
	lc := &storage.ListAlert_CommonEntityInfo{}
	lc.SetClusterName(resource.GetClusterName())
	lc.SetClusterId(resource.GetClusterId())
	lc.SetNamespace(resource.GetNamespace())
	lc.SetNamespaceId(resource.GetNamespaceId())
	lc.SetResourceType(resEnt)
	listAlert.SetCommonEntityInfo(lc)
}

func populateListAlertEntityInfoForDeployment(listAlert *storage.ListAlert, deployment *storage.Alert_Deployment) {
	lad := &storage.ListAlertDeployment{}
	lad.SetId(deployment.GetId())
	lad.SetName(deployment.GetName())
	lad.SetClusterName(deployment.GetClusterName())
	lad.SetClusterId(deployment.GetClusterId())
	lad.SetNamespace(deployment.GetNamespace())
	lad.SetNamespaceId(deployment.GetNamespaceId())
	lad.SetInactive(deployment.GetInactive())
	lad.SetDeploymentType(deployment.GetType())
	listAlert.SetDeployment(proto.ValueOrDefault(lad))
	lc := &storage.ListAlert_CommonEntityInfo{}
	lc.SetClusterName(deployment.GetClusterName())
	lc.SetClusterId(deployment.GetClusterId())
	lc.SetNamespace(deployment.GetNamespace())
	lc.SetNamespaceId(deployment.GetNamespaceId())
	lc.SetResourceType(storage.ListAlert_DEPLOYMENT)
	listAlert.SetCommonEntityInfo(lc)
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
	adc := &storage.Alert_Deployment_Container{}
	adc.SetName(c.GetName())
	adc.SetImage(c.GetImage())
	return adc
}

// ToAlertDeployment converts a storage.Deployment to an Alert_Deployment
func ToAlertDeployment(deployment *storage.Deployment) *storage.Alert_Deployment_ {
	alertDeployment := &storage.Alert_Deployment{}
	alertDeployment.SetId(deployment.GetId())
	alertDeployment.SetName(deployment.GetName())
	alertDeployment.SetType(deployment.GetType())
	alertDeployment.SetNamespace(deployment.GetNamespace())
	alertDeployment.SetNamespaceId(deployment.GetNamespaceId())
	alertDeployment.SetLabels(deployment.GetLabels())
	alertDeployment.SetClusterId(deployment.GetClusterId())
	alertDeployment.SetClusterName(deployment.GetClusterName())
	alertDeployment.SetAnnotations(deployment.GetAnnotations())
	alertDeployment.SetInactive(deployment.GetInactive())

	for _, c := range deployment.GetContainers() {
		alertDeployment.SetContainers(append(alertDeployment.GetContainers(), toAlertDeploymentContainer(c)))
	}
	return &storage.Alert_Deployment_{Deployment: alertDeployment}
}

// ToAlertResource converts a storage.KubernetesEvent to an Alert_Resource_
func ToAlertResource(kubeEvent *storage.KubernetesEvent) *storage.Alert_Resource_ {
	// TODO: Cluster name and namespace id will have to be passed in here
	// That will come from runtime detector (currently detector.detectForDeployment). This is TBD until the detection piece is completed
	// (and ROX-7355 is done for cluster name)
	ar := &storage.Alert_Resource{}
	ar.SetResourceType(storage.Alert_Resource_ResourceType(storage.Alert_Resource_ResourceType_value[strings.ToUpper(kubeEvent.GetObject().GetResource().String())]))
	ar.SetName(kubeEvent.GetObject().GetName())
	ar.SetClusterId(kubeEvent.GetObject().GetClusterId())
	ar.SetNamespace(kubeEvent.GetObject().GetNamespace())
	return &storage.Alert_Resource_{
		Resource: ar,
	}
}
