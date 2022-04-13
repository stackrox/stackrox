package preflight

import (
	"github.com/stackrox/stackrox/sensor/upgrader/common"
	"github.com/stackrox/stackrox/sensor/upgrader/plan"
	"github.com/stackrox/stackrox/sensor/upgrader/upgradectx"
)

type labelsCheck struct{}

func (labelsCheck) Name() string {
	return "Required labels"
}

func (labelsCheck) Check(ctx *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan, reporter checkReporter) error {
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
