package common

import (
	"iter"
	"slices"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// collectMatchingLabels returns an iterator over the labels and the values that
// match the expressions.
func collectMatchingLabels(labelExpression map[Label][]*Condition, getter func(Label) string) iter.Seq2[Label, string] {
	return func(yield func(Label, string) bool) {
		for label, expression := range labelExpression {
			if len(expression) == 0 {
				if !yield(label, getter(label)) {
					return
				}
				continue
			}
			skip := false
			for _, condition := range expression {
				if skip {
					skip = condition.op != opOR
					continue
				}
				if value := getter(label); condition.match(value) {
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

type valueOrder struct {
	int
	string
}

type orderedValues []valueOrder

func (ov orderedValues) sort() {
	slices.SortFunc(ov, func(a, b valueOrder) int {
		return a.int - b.int
	})
}

func (ov orderedValues) join(sep rune) string {
	ov.sort()
	sb := strings.Builder{}
	for _, value := range ov {
		if sb.Len() > 0 {
			sb.WriteRune(sep)
		}
		sb.WriteString(value.string)
	}
	return sb.String()
}

// makeAggregationKey computes an aggregation key according to the
// labels from the provided expression, and the map of the requested labels
// to their values. The values in the key are sorted according to the provided
// labelOrder map.
//
// Example:
//
//	"Cluster=*prod,Deployment" => "pre-prod|backend", {"Cluster": "pre-prod", "Deployment": "backend")}
func makeAggregationKey(labelExpression map[Label][]*Condition, getter func(Label) string, labelOrder map[Label]int) (aggregationKey, prometheus.Labels) {
	labels := make(prometheus.Labels)
	values := make(orderedValues, len(labelExpression))
	for label, value := range collectMatchingLabels(labelExpression, getter) {
		labels[string(label)] = value
		values = append(values, valueOrder{labelOrder[label], value})
	}
	if len(labels) != len(labelExpression) {
		return "", nil
	}
	return aggregationKey(values.join('|')), labels
}

// getMetricLabels extracts the metric labels from the filter expression and
// sort them according to the labelOrder map values.
// This makes the labels to appear in the stable order in the Prometheus output.
func getMetricLabels(labelExpression map[Label][]*Condition, labelOrder map[Label]int) []string {
	var labels []string
	for label := range labelExpression {
		labels = append(labels, string(label))
	}
	slices.SortFunc(labels, func(a, b string) int {
		return labelOrder[Label(a)] - labelOrder[Label(b)]
	})
	return labels
}
