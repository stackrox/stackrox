package aggregator

import (
	"iter"
	"slices"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// matchingLabels yields the labels and the values that match the expressions.
func matchingLabels(expressions map[Label][]*expression, labelsGetter func(Label) string) iter.Seq2[Label, string] {
	return func(yield func(Label, string) bool) {
		for label, expressions := range expressions {
			if len(expressions) == 0 {
				if !yield(label, labelsGetter(label)) {
					return
				}
				continue
			}
			skip := false
			for _, expr := range expressions {
				if skip {
					skip = expr.op != opOR
					continue
				}
				if value := labelsGetter(label); expr.match(value) {
					if !yield(label, value) {
						return
					}
					break
				}
				skip = true
			}
			if skip {
				return
			}
		}
	}
}

// makeAggregationKeyInstance computes an aggregation key according to the
// labels from the provided expressions, and the map of the requested labels
// to their values.
//
// Example:
//
//	"Cluster=*prod,Deployment" => "pre-prod|backend", {"Cluster": "pre-prod", "Deployment": "backend")}
func makeAggregationKeyInstance(expressions map[Label][]*expression, labelsGetter func(Label) string) (metricKey, prometheus.Labels) {
	labels := make(prometheus.Labels)
	type valueOrder struct {
		int
		string
	}
	values := []valueOrder{}
	for label, value := range matchingLabels(expressions, labelsGetter) {
		labels[string(label)] = value
		values = append(values, valueOrder{labelOrder[label], value})
	}
	if len(labels) != len(expressions) {
		return "", nil
	}
	slices.SortFunc(values, func(a, b valueOrder) int {
		return a.int - b.int
	})
	sb := strings.Builder{}
	for _, value := range values {
		if sb.Len() > 0 {
			sb.WriteRune('|')
		}
		sb.WriteString(value.string)
	}
	return metricKey(sb.String()), labels
}

// getMetricLabels extracts the metric labels from the filter expressions and
// sort them according to the labelOrder map values.
func getMetricLabels(expressions map[Label][]*expression) []string {
	var labels []string
	for label := range expressions {
		labels = append(labels, string(label))
	}
	slices.SortFunc(labels, func(a, b string) int {
		return labelOrder[Label(a)] - labelOrder[Label(b)]
	})
	return labels
}
