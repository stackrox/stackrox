package preflight

import (
	"github.com/stackrox/stackrox/sensor/upgrader/plan"
	"github.com/stackrox/stackrox/sensor/upgrader/upgradectx"
)

type checkReporter interface {
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type check interface {
	Name() string
	Check(ctx *upgradectx.UpgradeContext, plan *plan.ExecutionPlan, reporter checkReporter) error
}
