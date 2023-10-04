package preflight

import (
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

// Resources created in system namespaces for OpenShift monitoring.
var resourceExceptions = map[string][]string{
	namespaces.KubeSystem:          {"RoleBinding"},
	namespaces.OpenShiftMonitoring: {"ServiceMonitor", "PrometheusRule"},
}

type namespaceCheck struct{}

func (namespaceCheck) Name() string {
	return "Allowed namespaces"
}

func matchesException(resource *k8sobjects.ObjectRef) bool {
	if kinds, ok := resourceExceptions[resource.Namespace]; ok {
		for _, k := range kinds {
			if k == resource.GVK.Kind {
				return true
			}
		}
	}
	return false
}

func namespaceAllowed(resource *k8sobjects.ObjectRef) bool {
	if matchesException(resource) {
		return true
	}
	return resource.Namespace == "" || resource.Namespace == common.Namespace
}

func (namespaceCheck) Check(_ *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan, reporter checkReporter) error {
	for _, act := range execPlan.Actions() {
		act := act
		if !namespaceAllowed(&act.ObjectRef) {
			reporter.Errorf("To-be-%sd object %v is in disallowed namespace %s", act.ActionName, act.ObjectRef, common.Namespace)
		}
	}
	return nil
}
