package preflight

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

var (
	log = logging.LoggerForModule()
)

type defaultReporter struct {
	numErrors, numWarnings int
}

func (r *defaultReporter) Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
	r.numWarnings++
}

func (r *defaultReporter) Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
	r.numErrors++
}

// PerformChecks runs preflight checks against the given execution plan.
func PerformChecks(ctx *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan) error {
	for _, check := range preflightCheckList {
		log.Infof("Performing preflight check %q", check.Name())
		var reporter defaultReporter
		if err := check.Check(ctx, execPlan, &reporter); err != nil {
			return errors.Wrap(err, "error performing preflight check")
		}
		log.Infof("Preflight check %q finished with %d error(s), %d warning(s)", check.Name(), reporter.numErrors, reporter.numWarnings)

		if reporter.numErrors > 0 {
			return errors.Errorf("preflight check %q reported %d error(s)", check.Name(), reporter.numErrors)
		}
		if reporter.numWarnings > 0 {
			log.Warnf("There were %d warning(s) running preflight check %q", reporter.numWarnings, check.Name())
		}
	}

	return nil
}
