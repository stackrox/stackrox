package common

import (
	"context"
	"iter"
	"regexp"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.CreateLogger(logging.ModuleForName("central_metrics"), 1)

	metricNamePattern = regexp.MustCompile("^[a-zA-Z0-9_]+$")
)

type Label string      // Prometheus label.
type MetricName string // Prometheus metric name.
type FindingGenerator[Finding any] func(context.Context, MetricLabelsExpressions) iter.Seq[Finding]
type LabelGetter[Finding any] struct {
	Label  Label
	Getter func(Finding) string
}

// MetricLabelsExpressions is the parsed aggregation configuration.
type MetricLabelsExpressions map[MetricName]map[Label][]*Expression

type aggregationKey string // e.g. IMPORTANT_VULNERABILITY_SEVERITY|true

var ErrStopIterator = errors.New("stopped")

// validateMetricName ensures the name is alnum_.
func validateMetricName(name string) error {
	if len(name) == 0 {
		return errors.New("empty")
	}
	if !metricNamePattern.MatchString(name) {
		return errors.New("bad characters")
	}
	return nil
}

// Bind3rd binds the third function argument:
//
//	f(a, b, c) == Bind3rd(f, c)(a, b)
func Bind3rd[A any, B any, C any, RV any](f func(A, B, C) RV, c C) func(A, B) RV {
	return func(a A, b B) RV {
		return f(a, b, c)
	}
}
