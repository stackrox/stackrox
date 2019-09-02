package execution

import (
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

// ExecutePlan executes an upgrade execution plan.
func ExecutePlan(ctx *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan) error {
	e := &executor{ctx: ctx}
	return e.ExecutePlan(execPlan)
}
