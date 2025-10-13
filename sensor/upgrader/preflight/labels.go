package preflight

import (
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

type labelsCheck struct{}

func (labelsCheck) Name() string {
	return "Required labels"
}

func (labelsCheck) Check(_ *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan, reporter checkReporter) error {
	for _, act := range execPlan.Actions() {
		if act.Object == nil {
			continue
		}
		if act.Object.GetLabels()[common.UpgradeResourceLabelKey] != common.UpgradeResourceLabelValue {
			reporter.Errorf("To-be-%sd object %v does not carry the required %s=%s label", act.ActionName, act.ObjectRef, common.UpgradeResourceLabelKey, common.UpgradeResourceLabelValue)
		}
	}
	return nil
}
