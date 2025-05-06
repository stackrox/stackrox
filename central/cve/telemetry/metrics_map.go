package telemetry

import (
	"slices"
	"strings"
)

// makeAggregationKeyInstance computes an aggregation key according to the
// labels from the provided expressions, and the map of the requested labels
// to their values.
//
// Example:
//
//	"Cluster=*prod,Deployment" => "pre-prod|backend", {"Cluster": "pre-prod", "Deployment": "backend")}
func makeAggregationKeyInstance(expressions []*expression, labelsGetter func(Label) string) (metricKey, map[Label]string) {
	sb := strings.Builder{}
	labels := make(map[Label]string)
	for i := 0; i < len(expressions); i++ {
		expr := expressions[i]
		if expr == nil {
			break
		}
		value, ok := expr.match(labelsGetter)
		if !ok {
			for i++; i < len(expressions) && expressions[i] != nil; i++ {
			}
			if i == len(expressions) {
				return "", nil
			}
			continue
		}
		if value != "" {
			if sb.Len() > 0 {
				sb.WriteRune('|')
			}
			sb.WriteString(value)
			labels[expr.label] = value
		} else {
			return "", nil
		}
	}
	return metricKey(sb.String()), labels
}

// getMetricLabels extracts the metric labels from the filter expressions and
// sort them according to the labelOrder map values.
//
// Example:
//
//	"Cluster=*prod,Namespace" => {"Cluster", "Namespace"}
func getMetricLabels(expressions []*expression) []Label {
	var labels []Label
	for _, expression := range expressions {
		labels = append(labels, expression.label)
	}
	slices.SortFunc(labels, func(a, b Label) int {
		return labelOrder[a] - labelOrder[b]
	})
	return labels
}
