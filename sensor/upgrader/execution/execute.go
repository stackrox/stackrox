package execution

import (
	"github.com/stackrox/stackrox/sensor/upgrader/plan"
	"github.com/stackrox/stackrox/sensor/upgrader/upgradectx"
)

// ExecutePlan executes an upgrade execution plan.
func ExecutePlan(ctx *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan) error {
	e := &executor{ctx: ctx}
	return e.ExecutePlan(execPlan)
}
