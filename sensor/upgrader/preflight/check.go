package preflight

import (
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

type checkReporter interface {
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

type check interface {
	Name() string
	Check(ctx *upgradectx.UpgradeContext, plan *plan.ExecutionPlan, reporter checkReporter) error
}
