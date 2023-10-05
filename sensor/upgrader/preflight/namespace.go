package preflight

import (
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

// Resources created in namespaces other than common.Namespace.
var resourceExceptions = map[string]set.FrozenStringSet{
	namespaces.KubeSystem:          set.NewFrozenStringSet("RoleBinding"),
	namespaces.OpenShiftMonitoring: set.NewFrozenStringSet("ServiceMonitor", "PrometheusRule"),
}

type namespaceCheck struct{}

func (namespaceCheck) Name() string {
	return "Allowed namespaces"
}

func matchesException(resource *k8sobjects.ObjectRef) bool {
	if kinds, ok := resourceExceptions[resource.Namespace]; ok {
		if kinds.Contains(resource.GVK.Kind) {
			return true
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
