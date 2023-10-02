package preflight

import (
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

// Resources created in system namespaces for OpenShift monitoring.
var resourceExceptions = map[string][]string{
	"kube-system":          {"RoleBinding"},
	"openshift-monitoring": {"ServiceMonitor", "PrometheusRule"},
}

type namespaceCheck struct{}

func (namespaceCheck) Name() string {
	return "Allowed namespaces"
}

func isException(resource *k8sobjects.ObjectRef) bool {
	if kinds, ok := resourceExceptions[resource.Namespace]; ok {
		for _, k := range kinds {
			if k == resource.GVK.Kind {
				return true
			}
		}
	}
	return false
}

func (namespaceCheck) Check(_ *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan, reporter checkReporter) error {
	for _, act := range execPlan.Actions() {
		act := act
		doCheck := !isException(&act.ObjectRef)
		if doCheck && act.ObjectRef.Namespace != "" && act.ObjectRef.Namespace != common.Namespace {
			reporter.Errorf("To-be-%sd object %v is in disallowed namespace %s", act.ActionName, act.ObjectRef, common.Namespace)
		}
	}
	return nil
}
