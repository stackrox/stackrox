package preflight

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

var (
	log = logging.LoggerForModule()
)

type defaultReporter struct {
	errors   []string
	warnings []string
}

func (r *defaultReporter) Warnf(format string, args ...interface{}) {
	warning := fmt.Sprintf(format, args...)
	log.Warn(warning)
	r.warnings = append(r.warnings, warning)
}

func (r *defaultReporter) Errorf(format string, args ...interface{}) {
	errStr := fmt.Sprintf(format, args...)
	log.Error(errStr)
	r.errors = append(r.errors, errStr)
}

func formatWarnings(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}
	return fmt.Sprintf(". Additionally, it reported the following warnings:\n%s", strings.Join(warnings, "\n"))
}

// PerformChecks runs preflight checks against the given execution plan.
func PerformChecks(ctx *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan) error {
	for _, check := range preflightCheckList {
		log.Infof("Performing preflight check %q", check.Name())
		var reporter defaultReporter
		if err := check.Check(ctx, execPlan, &reporter); err != nil {
			return errors.Wrap(err, "error performing preflight check")
		}
		log.Infof("Preflight check %q finished with %d error(s), %d warning(s)", check.Name(), len(reporter.errors), len(reporter.warnings))

		if len(reporter.errors) > 0 {
			return errors.Errorf("preflight check %q reported errors:\n%s%s", check.Name(), strings.Join(sliceutils.Unique(reporter.errors), "\n"), formatWarnings(reporter.warnings))
		}
		if len(reporter.warnings) > 0 {
			log.Warnf("There were %d warning(s) running preflight check %q", len(reporter.warnings), check.Name())
		}
	}

	return nil
}
