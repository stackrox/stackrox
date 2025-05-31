package common

import (
	"context"
	"iter"
	"regexp"
	"slices"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.CreateLogger(logging.ModuleForName("central_metrics"), 1)

	metricNamePattern = regexp.MustCompile("^[a-zA-Z0-9_]+$")
)

type Label string      // Prometheus label.
type MetricName string // Prometheus metric name.
type FindingGenerator[Finding Count] func(context.Context, *v1.Query, MetricsConfiguration) iter.Seq[Finding]
type LabelGetter[Finding Count] struct {
	Label  Label
	Getter func(Finding) string
}

// MetricsConfiguration is the parsed aggregation configuration.
type MetricsConfiguration map[MetricName]map[Label]Expression

func (mcfg MetricsConfiguration) HasAnyLabelOf(labels []Label) bool {
	for _, labelExpr := range mcfg {
		for label := range labelExpr {
			if slices.Contains(labels, label) {
				return true
			}
		}
	}
	return false
}

type aggregationKey string // e.g. IMPORTANT_VULNERABILITY_SEVERITY|true

var ErrStopIterator = errors.New("stopped")

// validateMetricName ensures the name is alnum_.
func validateMetricName(name string) error {
	if len(name) == 0 {
		return errors.New("empty")
	}
	if !metricNamePattern.MatchString(name) {
		return errors.New(`doesn't match "` + metricNamePattern.String() + `"`)
	}
	return nil
}

// Bind4th binds the fourth function argument:
//
//	f(a, b, c, d) == Bind4th(f, d)(a, b, c)
func Bind4th[A1 any, A2 any, A3 any, A4 any, RV any](f func(A1, A2, A3, A4) RV, bound A4) func(A1, A2, A3) RV {
	return func(a1 A1, a2 A2, a3 A3) RV {
		return f(a1, a2, a3, bound)
	}
}
