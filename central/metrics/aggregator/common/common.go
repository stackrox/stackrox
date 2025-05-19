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

type Label string               // Prometheus label.
type MetricName string          // Prometheus metric name.
type Finding func(Label) string // Lazy map.
type FindingGenerator func(context.Context) iter.Seq[Finding]

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

// Bind2nd binds the second argument for the future calls of f.
func Bind2nd[A any, B any, RV any](f func(A, B) RV, b B) func(A) RV {
	return func(a A) RV {
		return f(a, b)
	}
}

// MakeLabelOrderMap makes the label order lookup map.
func MakeLabelOrderMap(labels []Label) map[Label]int {
	result := make(map[Label]int, len(labels))
	for i, label := range labels {
		result[label] = i + 1
	}
	return result
}
