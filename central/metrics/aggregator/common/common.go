package common

import (
	"context"
	"errors"
	"iter"
	"regexp"

	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.CreateLogger(logging.ModuleForName("central_metrics"), 1)

	metricNamePattern = regexp.MustCompile("^[a-zA-Z0-9_]+$")
)

type Label string               // Prometheus label.
type MetricName string          // Prometheus metric name.
type Finding func(Label) string // Lazy map.
type FindingGenerator func(context.Context) iter.Seq[Finding]

// MetricLabelsExpressions is the parsed aggregation configuration.
type MetricLabelsExpressions map[MetricName]map[Label][]*Expression

type aggregationKey string // e.g. IMPORTANT_VULNERABILITY_SEVERITY|true

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

// Bind2nd binds the second argument for the future calls of f.
func Bind2nd[A any, B any, RV any](f func(A, B) RV, b B) func(A) RV {
	return func(a A) RV {
		return f(a, b)
	}
}
