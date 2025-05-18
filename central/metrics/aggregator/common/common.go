package common

import (
	"context"
	"errors"
	"iter"
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
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

// MetricLabelExpressions is the parsed aggregation configuration.
type MetricLabelExpressions map[MetricName]map[Label][]*Expression

type metricKey string // e.g. IMPORTANT_VULNERABILITY_SEVERITY|true

// record is a single gauge metric record.
type record struct {
	labels prometheus.Labels
	total  int
}

// result is the aggregation result.
type result struct {
	aggregated map[MetricName]map[metricKey]*record
	mle        MetricLabelExpressions
	labelOrder map[Label]int
}

func makeResult(mle MetricLabelExpressions, labelOrder map[Label]int) *result {
	aggregated := make(map[MetricName]map[metricKey]*record)
	for metric := range mle {
		aggregated[metric] = make(map[metricKey]*record)
	}
	return &result{aggregated, mle, labelOrder}
}

func (r *result) count(finding Finding) {
	for metric, expressions := range r.mle {
		if key, labels := makeAggregationKeyInstance(expressions, finding, r.labelOrder); key != "" {
			if rec, ok := r.aggregated[metric][key]; ok {
				rec.total++
			} else {
				r.aggregated[metric][key] = &record{labels, 1}
			}
		}
	}
}

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
