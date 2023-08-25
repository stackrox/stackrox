package preflight

import (
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

type namespaceCheck struct{}

func (namespaceCheck) Name() string {
	return "Allowed namespaces"
}

func (namespaceCheck) Check(_ *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan, reporter checkReporter) error {
	for _, act := range execPlan.Actions() {
		if act.ObjectRef.Namespace != "" && act.ObjectRef.Namespace != common.Namespace {
			reporter.Errorf("To-be-%sd object %v is in disallowed namespace %s", act.ActionName, act.ObjectRef, common.Namespace)
		}
	}
	return nil
}
